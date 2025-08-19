package iamcollect

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// StorageClient provides an interface for storing and retrieving IAM data
type StorageClient interface {
	Store(ctx context.Context, path string, data []byte) error
	Retrieve(ctx context.Context, path string) ([]byte, error)
	List(ctx context.Context, prefix string) ([]string, error)
	ListAccountIDs(ctx context.Context) ([]string, error)
	Exists(ctx context.Context, path string) (bool, error)
}

// FileStorageClient implements file-based storage
type FileStorageClient struct {
	basePath string
}

// NewFileStorageClient creates a new file storage client
func NewFileStorageClient(basePath string) *FileStorageClient {
	return &FileStorageClient{
		basePath: basePath,
	}
}

// Store stores data to a file
func (fs *FileStorageClient) Store(ctx context.Context, path string, data []byte) error {
	fullPath := filepath.Join(fs.basePath, path)
	
	// Create directory if it doesn't exist
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// Write file
	if err := os.WriteFile(fullPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write file %s: %w", fullPath, err)
	}

	return nil
}

// Retrieve retrieves data from a file
func (fs *FileStorageClient) Retrieve(ctx context.Context, path string) ([]byte, error) {
	fullPath := filepath.Join(fs.basePath, path)
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", fullPath, err)
	}
	return data, nil
}

// List lists files with the given prefix
func (fs *FileStorageClient) List(ctx context.Context, prefix string) ([]string, error) {
	fullPrefix := filepath.Join(fs.basePath, prefix)
	var files []string

	err := filepath.Walk(fullPrefix, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			relPath, err := filepath.Rel(fs.basePath, path)
			if err != nil {
				return err
			}
			files = append(files, relPath)
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk directory %s: %w", fullPrefix, err)
	}

	return files, nil
}

// ListAccountIDs lists account IDs by examining the directory structure
func (fs *FileStorageClient) ListAccountIDs(ctx context.Context) ([]string, error) {
	accountsPath := filepath.Join(fs.basePath, "aws", "aws", "accounts")
	
	entries, err := os.ReadDir(accountsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to read accounts directory: %w", err)
	}

	var accountIDs []string
	for _, entry := range entries {
		if entry.IsDir() && len(entry.Name()) == 12 {
			// AWS account IDs are 12 digits
			accountIDs = append(accountIDs, entry.Name())
		}
	}

	return accountIDs, nil
}

// Exists checks if a file exists
func (fs *FileStorageClient) Exists(ctx context.Context, path string) (bool, error) {
	fullPath := filepath.Join(fs.basePath, path)
	_, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to check if file exists %s: %w", fullPath, err)
	}
	return true, nil
}

// S3StorageClient implements S3-based storage
type S3StorageClient struct {
	client aws.Config
	bucket string
	prefix string
}

// NewS3StorageClient creates a new S3 storage client
func NewS3StorageClient(cfg aws.Config, bucket, prefix string) *S3StorageClient {
	return &S3StorageClient{
		client: cfg,
		bucket: bucket,
		prefix: prefix,
	}
}

// Store stores data to S3
func (s3c *S3StorageClient) Store(ctx context.Context, path string, data []byte) error {
	s3Client := s3.NewFromConfig(s3c.client)
	
	key := s3c.getKey(path)
	_, err := s3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(s3c.bucket),
		Key:    aws.String(key),
		Body:   strings.NewReader(string(data)),
	})

	if err != nil {
		return fmt.Errorf("failed to store object %s: %w", key, err)
	}

	return nil
}

// Retrieve retrieves data from S3
func (s3c *S3StorageClient) Retrieve(ctx context.Context, path string) ([]byte, error) {
	s3Client := s3.NewFromConfig(s3c.client)
	
	key := s3c.getKey(path)
	result, err := s3Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s3c.bucket),
		Key:    aws.String(key),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to retrieve object %s: %w", key, err)
	}
	defer result.Body.Close()

	data, err := io.ReadAll(result.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read object body %s: %w", key, err)
	}

	return data, nil
}

// List lists objects with the given prefix
func (s3c *S3StorageClient) List(ctx context.Context, prefix string) ([]string, error) {
	s3Client := s3.NewFromConfig(s3c.client)
	
	fullPrefix := s3c.getKey(prefix)
	
	var objects []string
	paginator := s3.NewListObjectsV2Paginator(s3Client, &s3.ListObjectsV2Input{
		Bucket: aws.String(s3c.bucket),
		Prefix: aws.String(fullPrefix),
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list objects: %w", err)
		}

		for _, obj := range page.Contents {
			if obj.Key != nil {
				// Remove the prefix to get relative path
				relativePath := strings.TrimPrefix(*obj.Key, s3c.prefix)
				objects = append(objects, relativePath)
			}
		}
	}

	return objects, nil
}

// ListAccountIDs lists account IDs by examining S3 object keys
func (s3c *S3StorageClient) ListAccountIDs(ctx context.Context) ([]string, error) {
	accountsPrefix := "aws/aws/accounts/"
	objects, err := s3c.List(ctx, accountsPrefix)
	if err != nil {
		return nil, err
	}

	accountIDSet := make(map[string]bool)
	for _, obj := range objects {
		// Extract account ID from path like "aws/aws/accounts/123456789012/..."
		if strings.HasPrefix(obj, accountsPrefix) {
			parts := strings.Split(strings.TrimPrefix(obj, accountsPrefix), "/")
			if len(parts) > 0 && len(parts[0]) == 12 {
				accountIDSet[parts[0]] = true
			}
		}
	}

	var accountIDs []string
	for accountID := range accountIDSet {
		accountIDs = append(accountIDs, accountID)
	}

	return accountIDs, nil
}

// Exists checks if an object exists in S3
func (s3c *S3StorageClient) Exists(ctx context.Context, path string) (bool, error) {
	s3Client := s3.NewFromConfig(s3c.client)
	
	key := s3c.getKey(path)
	_, err := s3Client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s3c.bucket),
		Key:    aws.String(key),
	})

	if err != nil {
		// Check if it's a "not found" error
		if strings.Contains(err.Error(), "NotFound") || strings.Contains(err.Error(), "NoSuchKey") {
			return false, nil
		}
		return false, fmt.Errorf("failed to check if object exists %s: %w", key, err)
	}

	return true, nil
}

// getKey builds the full S3 key including prefix
func (s3c *S3StorageClient) getKey(path string) string {
	if s3c.prefix == "" {
		return path
	}
	return s3c.prefix + path
}

// CreateStorageClient creates a storage client based on configuration
func CreateStorageClient(ctx context.Context, config *StorageConfig, credProvider *CredentialsProvider) (StorageClient, error) {
	if config == nil {
		return nil, fmt.Errorf("storage configuration is required")
	}

	switch config.Type {
	case "file":
		if config.Path == "" {
			return nil, fmt.Errorf("path is required for file storage")
		}
		return NewFileStorageClient(config.Path), nil

	case "s3":
		if config.Bucket == "" {
			return nil, fmt.Errorf("bucket is required for S3 storage")
		}

		// Get AWS config for S3 operations
		var awsConfig aws.Config
		var err error

		if config.Auth != nil {
			// Use dedicated credentials for storage
			tempCredProvider := NewCredentialsProvider([]*Config{{Auth: config.Auth}})
			awsConfig, err = tempCredProvider.GetCredentials(ctx, "storage")
			if err != nil {
				return nil, fmt.Errorf("failed to get storage credentials: %w", err)
			}
		} else {
			// Use default credentials
			awsConfig, err = credProvider.GetCredentials(ctx, "default")
			if err != nil {
				return nil, fmt.Errorf("failed to get default credentials for storage: %w", err)
			}
		}

		if config.Region != "" {
			awsConfig.Region = config.Region
		}

		return NewS3StorageClient(awsConfig, config.Bucket, config.Prefix), nil

	default:
		return nil, fmt.Errorf("unsupported storage type: %s", config.Type)
	}
}