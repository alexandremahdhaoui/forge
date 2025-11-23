package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// S3Client wraps AWS SDK for S3 operations.
// It provides a simplified interface for downloading chart tarballs from S3-compatible storage.
type S3Client struct {
	client *s3.Client
}

// NewS3Client creates an S3 client with the specified endpoint and region.
// The endpoint should be a full URL (e.g., "http://localhost:9000" for MinIO).
// If region is empty, it defaults to "us-east-1".
// Returns an error if the endpoint is invalid or client creation fails.
func NewS3Client(endpoint, region string) (*S3Client, error) {
	// Validate endpoint
	if err := validateS3Endpoint(endpoint); err != nil {
		return nil, err
	}

	// Normalize region (default to us-east-1)
	region = normalizeS3Region(region)

	// Create AWS config with base endpoint
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(region),
		config.WithBaseEndpoint(endpoint),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create S3 client
	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		// Force path-style addressing for MinIO and other S3-compatible services
		o.UsePathStyle = true
	})

	return &S3Client{
		client: client,
	}, nil
}

// NewS3ClientWithCredentials creates an S3 client with explicit credentials.
// This is used when credentials are provided via Kubernetes Secret.
func NewS3ClientWithCredentials(endpoint, region, accessKeyID, secretAccessKey, sessionToken string) (*S3Client, error) {
	// Validate endpoint
	if err := validateS3Endpoint(endpoint); err != nil {
		return nil, err
	}

	// Normalize region
	region = normalizeS3Region(region)

	// Create credentials provider
	creds := credentials.NewStaticCredentialsProvider(accessKeyID, secretAccessKey, sessionToken)

	// Create AWS config with explicit credentials
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(region),
		config.WithCredentialsProvider(creds),
		config.WithBaseEndpoint(endpoint),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config with credentials: %w", err)
	}

	// Create S3 client with path-style addressing
	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = true
	})

	return &S3Client{
		client: client,
	}, nil
}

// DownloadFile downloads an object from S3 bucket to a local file.
// Returns an error if the download fails.
func (c *S3Client) DownloadFile(bucket, key, destPath string) error {
	// Validate inputs
	if bucket == "" {
		return fmt.Errorf("bucket name is required")
	}
	if key == "" {
		return fmt.Errorf("object key is required")
	}
	if destPath == "" {
		return fmt.Errorf("destination path is required")
	}

	// Create context with timeout (5 minutes for download)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Get object from S3
	result, err := c.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("S3 download timed out after 5 minutes")
		}
		return fmt.Errorf("failed to get object from S3: %w", err)
	}
	defer func() {
		if err := result.Body.Close(); err != nil {
			log.Printf("Warning: failed to close S3 result body: %v", err)
		}
	}()

	// Create destination file
	destFile, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer func() {
		if err := destFile.Close(); err != nil {
			log.Printf("Warning: failed to close destination file: %v", err)
		}
	}()

	// Copy from S3 to file
	_, err = io.Copy(destFile, result.Body)
	if err != nil {
		return fmt.Errorf("failed to write S3 object to file: %w", err)
	}

	return nil
}

// validateS3Endpoint validates that the endpoint is a valid HTTP/HTTPS URL.
func validateS3Endpoint(endpoint string) error {
	if endpoint == "" {
		return fmt.Errorf("endpoint is required")
	}

	if !strings.HasPrefix(endpoint, "http://") && !strings.HasPrefix(endpoint, "https://") {
		return fmt.Errorf("endpoint must start with http:// or https://")
	}

	return nil
}

// normalizeS3Region returns the region, defaulting to "us-east-1" if empty.
func normalizeS3Region(region string) string {
	region = strings.TrimSpace(region)
	if region == "" {
		return "us-east-1"
	}
	return region
}
