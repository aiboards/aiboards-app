package services

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"

	appconfig "github.com/garrettallen/aiboards/backend/config"
)

// FileInfo represents metadata about a stored file
type FileInfo struct {
	URL        string    `json:"url"`
	Filename   string    `json:"filename"`
	Size       int64     `json:"size"`
	MimeType   string    `json:"mime_type"`
	UploadedAt time.Time `json:"uploaded_at"`
}

// StorageService defines the interface for file storage operations
type StorageService interface {
	// UploadFile uploads a file and returns its public URL and metadata
	UploadFile(ctx context.Context, file io.Reader, filename, contentType string, size int64, agentID uuid.UUID) (*FileInfo, error)

	// DeleteFile deletes a file from storage
	DeleteFile(ctx context.Context, fileURL string) error
}

// R2StorageService implements StorageService using Cloudflare R2
type R2StorageService struct {
	client     *s3.Client
	bucketName string
	baseURL    string
}

// NewR2StorageService creates a new R2 storage service
func NewR2StorageService(cfg *appconfig.Config) (*R2StorageService, error) {
	// Create custom resolver to use R2 endpoint
	customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		return aws.Endpoint{
			URL: cfg.MediaStorageEndpoint,
		}, nil
	})

	// Configure AWS SDK
	awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(),
		awsconfig.WithRegion(cfg.MediaStorageRegion),
		awsconfig.WithEndpointResolverWithOptions(customResolver),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			cfg.MediaStorageKey,
			cfg.MediaStorageSecret,
			"",
		)),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create S3 client
	client := s3.NewFromConfig(awsCfg)

	// Construct public base URL for the bucket
	baseURL := fmt.Sprintf("https://%s", cfg.MediaStorageEndpoint)
	if !strings.HasPrefix(baseURL, "https://") {
		baseURL = fmt.Sprintf("https://%s", baseURL)
	}
	baseURL = fmt.Sprintf("%s/%s", baseURL, cfg.MediaStorageBucket)

	return &R2StorageService{
		client:     client,
		bucketName: cfg.MediaStorageBucket,
		baseURL:    baseURL,
	}, nil
}

// UploadFile implements StorageService.UploadFile for R2 storage
func (s *R2StorageService) UploadFile(ctx context.Context, file io.Reader, filename, contentType string, size int64, agentID uuid.UUID) (*FileInfo, error) {
	// Generate unique filename
	ext := filepath.Ext(filename)
	uniqueFilename := fmt.Sprintf("%s-%s%s", agentID.String(), uuid.New().String(), ext)

	// Define object key with agent ID as prefix
	objectKey := fmt.Sprintf("%s/%s", agentID.String(), uniqueFilename)

	// Upload file to R2
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucketName),
		Key:         aws.String(objectKey),
		Body:        file,
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to upload file to R2: %w", err)
	}

	// Generate public URL
	publicURL := fmt.Sprintf("%s/%s", s.baseURL, objectKey)

	return &FileInfo{
		URL:        publicURL,
		Filename:   filename,
		Size:       size,
		MimeType:   contentType,
		UploadedAt: time.Now(),
	}, nil
}

// DeleteFile implements StorageService.DeleteFile for R2 storage
func (s *R2StorageService) DeleteFile(ctx context.Context, fileURL string) error {
	// Extract object key from URL
	if !strings.HasPrefix(fileURL, s.baseURL) {
		return fmt.Errorf("invalid file URL: %s", fileURL)
	}

	objectKey := strings.TrimPrefix(fileURL, s.baseURL+"/")

	// Delete object from R2
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(objectKey),
	})
	if err != nil {
		return fmt.Errorf("failed to delete file from R2: %w", err)
	}

	return nil
}

// NewStorageService creates a storage service based on configuration
func NewStorageService(cfg *appconfig.Config) (StorageService, error) {
	return NewR2StorageService(cfg)
}
