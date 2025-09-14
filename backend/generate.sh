#!/bin/bash

# Generate OpenAPI server code into the generated package
# This script should be run from the backend directory

echo "Generating OpenAPI server code..."

# Remove existing generated files (except our custom ones)
rm -rf generated/temp_gen

# Generate clean files in temp directory
docker run --rm -v ${PWD}:/local openapitools/openapi-generator-cli generate \
  -i /local/openapi.yaml \
  -g go-server \
  -o /local/generated/temp_gen \
  --additional-properties=packageName=generated

# Move generated Go files to the generated package, excluding main.go
mv generated/temp_gen/go/*.go generated/
rm -f generated/main.go  # We have our own main.go

# Clean up
rm -rf generated/temp_gen

echo "OpenAPI code generation complete!"
echo "Generated files are in the 'generated' package."
echo "Business logic remains in the 'business' package."
echo "Application entry point is in 'cmd/main.go'."