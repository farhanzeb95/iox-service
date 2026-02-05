package services

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"log"
	"image/jpeg"
	"image/png"
	"io"
	"mime/multipart"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/disintegration/imaging"
)

var (
	s3Client           *s3.Client
	defaultBucket      string
	publicURL          string // default bucket public URL (backward compat)
	bucketPublicURLs   map[string]string // bucket name -> public URL base
	storageInitialized bool
)

// initS3Storage initializes the S3-compatible client (used by Supabase Storage).
func initS3Storage(endpoint, accessKeyID, accessKeySecret, bucket, publicURLBase string) error {
	defaultBucket = bucket
	if bucketPublicURLs == nil {
		bucketPublicURLs = make(map[string]string)
	}
	if publicURLBase != "" {
		publicURL = publicURLBase
	} else {
		publicURL = fmt.Sprintf("%s/%s", endpoint, bucket)
	}
	bucketPublicURLs[bucket] = strings.TrimSuffix(publicURL, "/")

	resolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		return aws.Endpoint{
			URL:           endpoint,
			SigningRegion: "us-east-1",
		}, nil
	})

	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithEndpointResolverWithOptions(resolver),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKeyID, accessKeySecret, "")),
		config.WithRegion("us-east-1"),
	)
	if err != nil {
		return err
	}

	s3Client = s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = true // Required for Supabase S3 API
	})

	err = ensureBucketExists(context.TODO(), bucket)
	if err != nil {
		return fmt.Errorf("failed to ensure bucket exists: %v", err)
	}

	storageInitialized = true
	return nil
}

// InitSupabaseStorage initializes the S3 client for Supabase Storage.
// Endpoint: https://<project_ref>.storage.supabase.co/storage/v1/s3
// Credentials: Supabase Dashboard → Project Settings → Storage → S3 Access Keys
func InitSupabaseStorage(endpoint, accessKeyID, accessKeySecret, bucket, publicURLBase string) error {
	return initS3Storage(endpoint, accessKeyID, accessKeySecret, bucket, publicURLBase)
}

// RegisterBucket ensures a bucket exists and registers its public URL (e.g. users bucket).
// Call after InitSupabaseStorage. Same S3 credentials are used for all buckets.
func RegisterBucket(bucketName, publicURLBase string) error {
	if !IsStorageInitialized() || s3Client == nil {
		return fmt.Errorf("storage not initialized - call InitSupabaseStorage first")
	}
	publicURLBase = strings.TrimSuffix(publicURLBase, "/")
	if bucketPublicURLs == nil {
		bucketPublicURLs = make(map[string]string)
	}
	bucketPublicURLs[bucketName] = publicURLBase
	return ensureBucketExists(context.TODO(), bucketName)
}

// IsStorageInitialized returns whether storage is available
func IsStorageInitialized() bool {
	return storageInitialized && s3Client != nil
}

// ensureBucketExists checks if bucket exists and creates it if it doesn't.
// If the bucket was already created in the Supabase dashboard, CreateBucket may return
// BucketAlreadyExists (409) — we treat that as success. Supabase often does not support
// S3 PutBucketPolicy; make buckets public in Dashboard → Storage if needed.
func ensureBucketExists(ctx context.Context, bucket string) error {
	_, err := s3Client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(bucket),
	})

	if err != nil {
		_, createErr := s3Client.CreateBucket(ctx, &s3.CreateBucketInput{
			Bucket: aws.String(bucket),
		})
		if createErr != nil {
			// Bucket may already exist (e.g. created in Supabase dashboard) — continue
			if !strings.Contains(createErr.Error(), "BucketAlreadyExists") {
				return fmt.Errorf("failed to create bucket: %v", createErr)
			}
		}
	}

	// Try to set public read policy; Supabase Storage often does not support S3 bucket policies
	publicReadPolicy := fmt.Sprintf(`{
		"Version": "2012-10-17",
		"Statement": [
			{
				"Effect": "Allow",
				"Principal": {"AWS": ["*"]},
				"Action": ["s3:GetObject"],
				"Resource": ["arn:aws:s3:::%s/*"]
			}
		]
	}`, bucket)

	_, err = s3Client.PutBucketPolicy(ctx, &s3.PutBucketPolicyInput{
		Bucket: aws.String(bucket),
		Policy: aws.String(publicReadPolicy),
	})
	if err != nil {
		// Supabase manages public access via Dashboard → Storage; 409/BucketAlreadyExists is expected — ignore
		if !strings.Contains(err.Error(), "BucketAlreadyExists") && !strings.Contains(err.Error(), "409") {
			log.Printf("⚠️  Bucket %q policy not set: %v", bucket, err)
		}
	}

	return nil
}

// UploadImage uploads an image file to Supabase Storage and returns the public URL
func UploadImage(file multipart.File, header *multipart.FileHeader, folder string) (string, error) {
	if !IsStorageInitialized() {
		return "", fmt.Errorf("storage service not initialized - set SUPABASE_STORAGE_* in .env")
	}

	// Validate file type
	ext := strings.ToLower(filepath.Ext(header.Filename))
	if ext != ".jpg" && ext != ".jpeg" && ext != ".png" && ext != ".webp" {
		return "", fmt.Errorf("unsupported file type: %s. Only jpg, jpeg, png, and webp are allowed", ext)
	}

	// Validate file size (max 10MB)
	if header.Size > 10*1024*1024 {
		return "", fmt.Errorf("file too large: max 10MB, got %d bytes", header.Size)
	}

	// Read file into memory
	fileBytes, err := io.ReadAll(file)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %v", err)
	}

	// Resize image (optional - create thumbnail and medium size)
	resizedBytes, err := resizeImage(fileBytes, ext, 1200) // Max width 1200px
	if err != nil {
		// If resize fails, use original
		resizedBytes = fileBytes
	}

	// Generate unique filename
	timestamp := time.Now().UnixNano()
	filename := fmt.Sprintf("%s/%d_%s", folder, timestamp, header.Filename)
	filename = sanitizeFilename(filename)

	// Upload to default bucket (products)
	_, err = s3Client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket:      aws.String(defaultBucket),
		Key:         aws.String(filename),
		Body:        bytes.NewReader(resizedBytes),
		ContentType: aws.String(getContentType(ext)),
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload: %v", err)
	}

	// Return public URL
	return fmt.Sprintf("%s/%s", publicURL, filename), nil
}

// UploadImageToBucket uploads an image to a specific bucket (e.g. "users" for avatars/identity docs).
// The bucket must have been registered via InitSupabaseStorage (products) or RegisterBucket (e.g. users).
func UploadImageToBucket(bucket, folder string, file multipart.File, header *multipart.FileHeader) (string, error) {
	if !IsStorageInitialized() {
		return "", fmt.Errorf("storage service not initialized - set SUPABASE_STORAGE_* in .env")
	}
	if _, ok := bucketPublicURLs[bucket]; !ok {
		return "", fmt.Errorf("unknown bucket %q - register it with RegisterBucket at startup", bucket)
	}

	ext := strings.ToLower(filepath.Ext(header.Filename))
	if ext != ".jpg" && ext != ".jpeg" && ext != ".png" && ext != ".webp" {
		return "", fmt.Errorf("unsupported file type: %s. Only jpg, jpeg, png, and webp are allowed", ext)
	}
	if header.Size > 10*1024*1024 {
		return "", fmt.Errorf("file too large: max 10MB, got %d bytes", header.Size)
	}

	fileBytes, err := io.ReadAll(file)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %v", err)
	}
	resizedBytes, err := resizeImage(fileBytes, ext, 1200)
	if err != nil {
		resizedBytes = fileBytes
	}

	timestamp := time.Now().UnixNano()
	filename := fmt.Sprintf("%s/%d_%s", folder, timestamp, header.Filename)
	filename = sanitizeFilename(filename)

	_, err = s3Client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket:      aws.String(bucket),
		Key:         aws.String(filename),
		Body:        bytes.NewReader(resizedBytes),
		ContentType: aws.String(getContentType(ext)),
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload: %v", err)
	}

	base := bucketPublicURLs[bucket]
	return fmt.Sprintf("%s/%s", base, filename), nil
}

// UploadMultipleImages uploads multiple images to the default (products) bucket.
func UploadMultipleImages(files []*multipart.FileHeader, folder string) ([]string, error) {
	var urls []string
	for _, header := range files {
		file, err := header.Open()
		if err != nil {
			return nil, fmt.Errorf("failed to open file %s: %v", header.Filename, err)
		}

		url, err := UploadImage(file, header, folder)
		file.Close()
		if err != nil {
			return nil, err
		}
		urls = append(urls, url)
	}
	return urls, nil
}

// UploadMultipleImagesToBucket uploads multiple images to a specific bucket (e.g. "users").
func UploadMultipleImagesToBucket(bucket, folder string, files []*multipart.FileHeader) ([]string, error) {
	var urls []string
	for _, header := range files {
		file, err := header.Open()
		if err != nil {
			return nil, fmt.Errorf("failed to open file %s: %v", header.Filename, err)
		}
		url, err := UploadImageToBucket(bucket, folder, file, header)
		file.Close()
		if err != nil {
			return nil, err
		}
		urls = append(urls, url)
	}
	return urls, nil
}

// DeleteImage deletes an image from storage. Supports URLs from any registered bucket.
func DeleteImage(imageURL string) error {
	if !IsStorageInitialized() {
		return fmt.Errorf("storage service not initialized - set SUPABASE_STORAGE_* in .env")
	}

	// Resolve bucket and key from URL (try each known public URL prefix)
	for bucket, base := range bucketPublicURLs {
		base = strings.TrimSuffix(base, "/")
		key := strings.TrimPrefix(imageURL, base+"/")
		if key != imageURL {
			_, err := s3Client.DeleteObject(context.TODO(), &s3.DeleteObjectInput{
				Bucket: aws.String(bucket),
				Key:    aws.String(key),
			})
			return err
		}
	}
	return fmt.Errorf("invalid image URL (no matching bucket): %s", imageURL)
}

// Helper functions
func resizeImage(data []byte, ext string, maxWidth int) ([]byte, error) {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// Only resize if larger than maxWidth
	if width <= maxWidth {
		return data, nil
	}

	// Calculate new height maintaining aspect ratio
	newHeight := int(float64(height) * float64(maxWidth) / float64(width))
	resized := imaging.Resize(img, maxWidth, newHeight, imaging.Lanczos)

	var buf bytes.Buffer
	switch ext {
	case ".jpg", ".jpeg":
		err = jpeg.Encode(&buf, resized, &jpeg.Options{Quality: 85})
	case ".png":
		err = png.Encode(&buf, resized)
	default:
		return data, nil // Return original if can't encode
	}

	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func sanitizeFilename(filename string) string {
	// Remove special characters, keep only alphanumeric, dots, dashes, underscores, and slashes
	filename = strings.ReplaceAll(filename, " ", "_")
	// Remove any other potentially problematic characters
	filename = strings.ReplaceAll(filename, "(", "")
	filename = strings.ReplaceAll(filename, ")", "")
	filename = strings.ReplaceAll(filename, "[", "")
	filename = strings.ReplaceAll(filename, "]", "")
	return filename
}

func getContentType(ext string) string {
	switch ext {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".webp":
		return "image/webp"
	default:
		return "application/octet-stream"
	}
}
