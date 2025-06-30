#!/bin/bash

# Create a test image
echo "Creating test image..."
convert -size 100x100 xc:red /tmp/test_image.jpg 2>/dev/null || {
    # If ImageMagick is not available, create a simple test file
    echo "Creating simple test file..."
    echo "fake image data" > /tmp/test_image.jpg
}

# Get a valid token first (you'll need to replace with a real token)
TOKEN="your_jwt_token_here"

echo "Testing image upload to store service..."
curl -X POST http://localhost:8081/api/v1/items \
  -H "Authorization: Bearer $TOKEN" \
  -F "title=Test Item with Image" \
  -F "description=Testing image upload functionality" \
  -F "category=electronics" \
  -F "condition=new" \
  -F "price_type=fixed" \
  -F "fixed_price=1000" \
  -F "images=@/tmp/test_image.jpg" \
  -v

echo "Done!"