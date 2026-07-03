package storage

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// R2Config holds configuration for Cloudflare R2 storage
type R2Config struct {
	AccountID   string
	AccessKeyID string
	SecretKey   string
	BucketName  string
	PublicURL   string // e.g., https://cdn-dev.smartscan.com
}

// R2Client provides methods to interact with Cloudflare R2 storage
type R2Client struct {
	client    *s3.Client
	bucket    string
	publicURL string
}

// NewR2Client creates a new R2 client with the given configuration
func NewR2Client(ctx context.Context, cfg R2Config) (*R2Client, error) {
	// Validate required fields
	if cfg.AccountID == "" || cfg.AccessKeyID == "" || cfg.SecretKey == "" || cfg.BucketName == "" {
		return nil, fmt.Errorf("R2 configuration incomplete: account_id, access_key_id, secret_key, and bucket_name are required")
	}

	// R2 endpoint format: https://<account_id>.r2.cloudflarestorage.com
	endpoint := fmt.Sprintf("https://%s.r2.cloudflarestorage.com", cfg.AccountID)

	// Create custom resolver for R2 endpoint
	r2Resolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		return aws.Endpoint{
			URL: endpoint,
		}, nil
	})

	// Create AWS config with static credentials
	awsCfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion("auto"), // R2 uses "auto" region
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			cfg.AccessKeyID,
			cfg.SecretKey,
			"", // session token (not used for R2)
		)),
		config.WithEndpointResolverWithOptions(r2Resolver),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create S3 client
	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.UsePathStyle = true // R2 requires path-style URLs
	})

	// Ensure public URL doesn't have trailing slash
	publicURL := strings.TrimSuffix(cfg.PublicURL, "/")

	return &R2Client{
		client:    client,
		bucket:    cfg.BucketName,
		publicURL: publicURL,
	}, nil
}

// Upload uploads a file to R2 and returns the public URL
// key should be the full path including directories (e.g., "products/tenant-id/product-id/gallery/image.jpg")
func (r *R2Client) Upload(ctx context.Context, key string, data []byte, contentType string) (string, error) {
	// Ensure key doesn't start with /
	key = strings.TrimPrefix(key, "/")

	input := &s3.PutObjectInput{
		Bucket:      aws.String(r.bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(data),
		ContentType: aws.String(contentType),
	}

	_, err := r.client.PutObject(ctx, input)
	if err != nil {
		return "", fmt.Errorf("failed to upload to R2: %w", err)
	}

	return r.GetPublicURL(key), nil
}

// Delete removes a file from R2
func (r *R2Client) Delete(ctx context.Context, key string) error {
	// Ensure key doesn't start with /
	key = strings.TrimPrefix(key, "/")

	input := &s3.DeleteObjectInput{
		Bucket: aws.String(r.bucket),
		Key:    aws.String(key),
	}

	_, err := r.client.DeleteObject(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to delete from R2: %w", err)
	}

	return nil
}

// GetPublicURL returns the public URL for a given key
func (r *R2Client) GetPublicURL(key string) string {
	// Ensure key doesn't start with /
	key = strings.TrimPrefix(key, "/")
	return fmt.Sprintf("%s/%s", r.publicURL, key)
}

// ExtractKeyFromURL extracts the R2 key from a public URL
// Returns the key without leading slash, or empty string if not an R2 URL
func (r *R2Client) ExtractKeyFromURL(url string) string {
	if !strings.HasPrefix(url, r.publicURL) {
		return ""
	}
	key := strings.TrimPrefix(url, r.publicURL)
	key = strings.TrimPrefix(key, "/")
	return key
}

// IsR2URL checks if a URL is an R2 URL managed by this client
func (r *R2Client) IsR2URL(url string) bool {
	return strings.HasPrefix(url, r.publicURL)
}

// HeadObject checks if an object exists in R2
func (r *R2Client) Exists(ctx context.Context, key string) (bool, error) {
	key = strings.TrimPrefix(key, "/")

	input := &s3.HeadObjectInput{
		Bucket: aws.String(r.bucket),
		Key:    aws.String(key),
	}

	_, err := r.client.HeadObject(ctx, input)
	if err != nil {
		// Check if it's a "not found" error
		if strings.Contains(err.Error(), "NotFound") || strings.Contains(err.Error(), "404") {
			return false, nil
		}
		return false, fmt.Errorf("failed to check object existence: %w", err)
	}

	return true, nil
}
