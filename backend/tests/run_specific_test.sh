#!/bin/bash
set -e

# Store the original directory
ORIGINAL_DIR=$(pwd)
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

# Default values
TEST_PACKAGE=""
TEST_PATTERN=""
DB_URL="postgres://aiboards_test:aiboards_test@localhost:5433/aiboards_test?sslmode=disable"
MIGRATION_PATH="$(cd "$SCRIPT_DIR/.." && pwd)/migrations"
VERBOSE=false

# Function to show usage
show_usage() {
  echo "Usage: $0 [options] -- [go test flags]"
  echo "Options:"
  echo "  -p, --package PACKAGE       Test package path (e.g., ./integration/... or ./api/...)"
  echo "  -t, --test PATTERN          Test pattern to run (e.g., TestPostService/CreatePost_InactiveBoard)"
  echo "  -d, --db-url URL            Database URL (default: $DB_URL)"
  echo "  -m, --migration-path PATH   Migration path (default: $MIGRATION_PATH)"
  echo "  -v, --verbose               Enable verbose output"
  echo "  -h, --help                  Show this help message"
  echo ""
  echo "Examples:"
  echo "  $0 -p ./integration/... -t TestPostService/CreatePost_InactiveBoard"
  echo "  $0 --package ./api/... --test TestListBoardsEndpoint --verbose"
  echo "  $0 -p ./integration/... -t TestPostService -- -v -count=1"
  exit 1
}

# Parse arguments
while [[ $# -gt 0 ]]; do
  case "$1" in
    -p|--package)
      TEST_PACKAGE="$2"
      shift 2
      ;;
    -t|--test)
      TEST_PATTERN="$2"
      shift 2
      ;;
    -d|--db-url)
      DB_URL="$2"
      shift 2
      ;;
    -m|--migration-path)
      MIGRATION_PATH="$2"
      shift 2
      ;;
    -v|--verbose)
      VERBOSE=true
      shift
      ;;
    -h|--help)
      show_usage
      ;;
    --)
      shift
      GO_TEST_FLAGS="$*"
      break
      ;;
    *)
      echo "Unknown option: $1"
      show_usage
      ;;
  esac
done

# Validate required arguments
if [ -z "$TEST_PACKAGE" ]; then
  echo "Error: Test package is required"
  show_usage
fi

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
export TEST_DATABASE_URL="$DB_URL"
export MIGRATION_PATH="$MIGRATION_PATH"

# Build the test command
TEST_CMD="go test"
if [ "$VERBOSE" = true ]; then
  TEST_CMD="$TEST_CMD -v"
fi
if [ -n "$TEST_PATTERN" ]; then
  TEST_CMD="$TEST_CMD -run \"$TEST_PATTERN\""
fi
if [ -n "$GO_TEST_FLAGS" ]; then
  TEST_CMD="$TEST_CMD $GO_TEST_FLAGS"
fi
TEST_CMD="$TEST_CMD $TEST_PACKAGE"

# Run the specific test
echo "Running test command: $TEST_CMD"
cd "$SCRIPT_DIR/.."
if ! eval "$TEST_CMD"; then
  echo "Test failed!"
  exit 1
fi

echo "Test passed successfully!"
