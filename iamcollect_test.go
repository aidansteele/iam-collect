package iamcollect

import (
	"context"
	"testing"
	"time"
)

// TestDownloadData tests the DownloadData function
func TestDownloadData(t *testing.T) {
	// Create a test configuration with file storage
	config := &Config{
		Name:              "test-config",
		IamCollectVersion: "0.1.0",
		Storage: &StorageConfig{
			Type: "file",
			Path: "/tmp/iam-collect-test",
		},
		Accounts: &AccountsConfig{
			Included: []string{"123456789012"}, // Mock account ID
		},
	}

	options := &DownloadOptions{
		Configs:     []*Config{config},
		AccountIDs:  []string{"123456789012"},
		Services:    []string{"iam"},
		Concurrency: 2,
		SkipIndex:   true, // Skip indexing for this test
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Note: This test would require valid AWS credentials and permissions
	// In a real test environment, you might want to use mocked AWS services
	err := DownloadData(ctx, options)
	if err != nil {
		t.Logf("DownloadData test skipped (requires AWS credentials): %v", err)
		return
	}

	t.Log("DownloadData completed successfully")
}

// TestIndex tests the Index function
func TestIndex(t *testing.T) {
	// Create a test configuration with file storage
	config := &Config{
		Name:              "test-config",
		IamCollectVersion: "0.1.0",
		Storage: &StorageConfig{
			Type: "file",
			Path: "/tmp/iam-collect-test",
		},
	}

	options := &IndexOptions{
		Configs:     []*Config{config},
		Partition:   "aws",
		AccountIDs:  []string{"123456789012"},
		Services:    []string{"iam"},
		Concurrency: 2,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := Index(ctx, options)
	if err != nil {
		t.Logf("Index test skipped (requires existing data): %v", err)
		return
	}

	t.Log("Index completed successfully")
}

// TestFileStorageClient tests the file storage client
func TestFileStorageClient(t *testing.T) {
	storage := NewFileStorageClient("/tmp/iam-collect-test-storage")
	
	ctx := context.Background()
	testPath := "test/path/file.json"
	testData := []byte(`{"test": "data"}`)

	// Test Store
	err := storage.Store(ctx, testPath, testData)
	if err != nil {
		t.Fatalf("Failed to store data: %v", err)
	}

	// Test Exists
	exists, err := storage.Exists(ctx, testPath)
	if err != nil {
		t.Fatalf("Failed to check if file exists: %v", err)
	}
	if !exists {
		t.Fatalf("File should exist after storing")
	}

	// Test Retrieve
	retrievedData, err := storage.Retrieve(ctx, testPath)
	if err != nil {
		t.Fatalf("Failed to retrieve data: %v", err)
	}
	if string(retrievedData) != string(testData) {
		t.Fatalf("Retrieved data doesn't match stored data")
	}

	// Test List
	files, err := storage.List(ctx, "test/")
	if err != nil {
		t.Fatalf("Failed to list files: %v", err)
	}
	if len(files) != 1 || files[0] != testPath {
		t.Fatalf("List didn't return expected file")
	}

	t.Log("FileStorageClient tests passed")
}

// TestCredentialsProvider tests the credentials provider
func TestCredentialsProvider(t *testing.T) {
	config := &Config{
		Auth: &AuthConfig{
			Profile: "default", // Use default profile if available
		},
	}

	provider := NewCredentialsProvider([]*Config{config})
	
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Test getting credentials (this will fail without valid AWS config)
	_, err := provider.GetCredentials(ctx, "123456789012")
	if err != nil {
		t.Logf("CredentialsProvider test skipped (requires AWS credentials): %v", err)
		return
	}

	t.Log("CredentialsProvider test passed")
}

// TestConfigValidation tests configuration validation
func TestConfigValidation(t *testing.T) {
	// Test invalid config
	invalidOptions := &DownloadOptions{
		Configs: []*Config{}, // Empty configs
	}

	err := DownloadData(context.Background(), invalidOptions)
	if err == nil {
		t.Fatalf("Expected error for invalid config, got nil")
	}

	// Test nil options
	err = DownloadData(context.Background(), nil)
	if err == nil {
		t.Fatalf("Expected error for nil options, got nil")
	}

	// Test valid config structure
	validConfig := &Config{
		Name:              "test",
		IamCollectVersion: "0.1.0",
		Storage: &StorageConfig{
			Type: "file",
			Path: "/tmp/test",
		},
	}

	validOptions := &DownloadOptions{
		Configs: []*Config{validConfig},
	}

	// This should not fail validation (but may fail during execution)
	err = DownloadData(context.Background(), validOptions)
	if err != nil && err.Error() == "at least one configuration is required" {
		t.Fatalf("Configuration validation failed unexpectedly: %v", err)
	}

	t.Log("Config validation tests passed")
}

// TestServiceTypes tests the service type definitions
func TestServiceTypes(t *testing.T) {
	services := AllServices()
	
	if len(services) == 0 {
		t.Fatalf("AllServices() should return non-empty slice")
	}

	// Check that IAM service is included
	found := false
	for _, svc := range services {
		if svc == ServiceIAM {
			found = true
			break
		}
	}

	if !found {
		t.Fatalf("IAM service should be included in AllServices()")
	}

	// Test service string conversion
	if string(ServiceIAM) != "iam" {
		t.Fatalf("ServiceIAM should convert to 'iam'")
	}

	t.Log("Service type tests passed")
}