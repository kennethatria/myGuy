# Image Storage Documentation

## Current Implementation

Images uploaded to the store service are stored **locally on the server's file system**.

### Storage Location
```
store-service/
└── uploads/
    └── store/
        └── {user_id}/
            ├── {timestamp}_0.jpg
            ├── {timestamp}_1.png
            └── {timestamp}_2.gif
```

### How It Works

1. **Upload Process**:
   - Images are uploaded via multipart form data
   - Maximum 3 images per item
   - Maximum file size: 5MB per image
   - Supported formats: JPG, JPEG, PNG, GIF

2. **File Storage**:
   - Files are saved to `./uploads/store/{user_id}/` directory
   - Each file is renamed with a timestamp to avoid conflicts
   - Format: `{timestamp}_{index}.{extension}`

3. **Serving Images**:
   - Images are served statically via the route `/uploads/*`
   - Example URL: `http://localhost:8081/uploads/store/123/1701234567_0.jpg`

### Database Storage

Only the image URLs are stored in the database:
- Table: `item_images`
- Fields: `id`, `item_id`, `url`, `order`, `created_at`

## Production Recommendations

For production environments, consider:

1. **Cloud Storage** (Recommended):
   - AWS S3
   - Google Cloud Storage
   - Cloudflare R2
   - Azure Blob Storage

2. **CDN Integration**:
   - Use a CDN for faster global delivery
   - Implement image optimization (resize, compress)

3. **Security**:
   - Add virus scanning
   - Implement rate limiting
   - Add authentication for uploads
   - Validate MIME types, not just extensions

4. **Backup Strategy**:
   - Regular backups of uploaded images
   - Redundant storage across multiple regions

## Example S3 Implementation

```go
// Example S3 upload code
func uploadToS3(file multipart.File, filename string) (string, error) {
    sess := session.Must(session.NewSession(&aws.Config{
        Region: aws.String("us-west-2"),
    }))
    
    uploader := s3manager.NewUploader(sess)
    
    result, err := uploader.Upload(&s3manager.UploadInput{
        Bucket: aws.String("my-store-images"),
        Key:    aws.String(filename),
        Body:   file,
        ACL:    aws.String("public-read"),
    })
    
    if err != nil {
        return "", err
    }
    
    return result.Location, nil
}
```

## Directory Permissions

Ensure the uploads directory has proper permissions:
```bash
mkdir -p ./uploads/store
chmod 755 ./uploads/store
```

The application will create user-specific subdirectories automatically.