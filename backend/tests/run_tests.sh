#!/bin/bash
set -e

# Store the original directory
ORIGINAL_DIR=$(pwd)
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

# Function to clean up resources
cleanup() {
  echo "Cleaning up..."
  # Always return to the tests directory for cleanup
  cd "$SCRIPT_DIR"
  docker-compose -f docker-compose.test.yml down
  echo "Done!"
}

# Set up trap to ensure cleanup happens even if the script fails
trap cleanup EXIT

echo "Starting test database..."
cd "$SCRIPT_DIR" # Ensure we're in the tests directory
docker-compose -f docker-compose.test.yml down -v 2>/dev/null || true # Clean up any existing containers
docker-compose -f docker-compose.test.yml up -d

# Wait for the database to be ready
echo "Waiting for database to be ready..."
max_attempts=30
attempt=0
until docker exec aiboards-postgres-test pg_isready -U aiboards_test -d aiboards_test; do
  attempt=$((attempt+1))
  if [ $attempt -ge $max_attempts ]; then
    echo "Database did not become ready in time. Exiting."
    exit 1
  fi
  echo "Database is not ready yet... waiting 1 second (attempt $attempt/$max_attempts)"
  sleep 1
done

echo "Database is ready!"

# Set environment variables for tests
export TEST_DATABASE_URL="postgres://aiboards_test:aiboards_test@localhost:5433/aiboards_test?sslmode=disable"
export MIGRATION_PATH="$(cd "$SCRIPT_DIR/.." && pwd)/migrations"

# Run API tests
echo "Running API tests..."
cd "$SCRIPT_DIR/.."
if ! go test -v ./tests/api/...; then
  echo "API tests failed!"
  exit 1
fi

echo "API tests passed successfully!"

# Run integration tests
echo "Running integration tests..."
cd "$SCRIPT_DIR/.."
if ! go test -v ./tests/integration/...; then
  echo "Integration tests failed!"
  exit 1
fi

echo "Integration tests passed successfully!"

# Run unit tests
echo "Running unit tests..."
cd "$SCRIPT_DIR/.."
if ! go test -v ./tests/unit/...; then
  echo "Unit tests failed!"
  exit 1
fi

echo "All tests passed successfully!"
