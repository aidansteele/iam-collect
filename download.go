package iamcollect

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/sts"
)

// DownloadData downloads IAM data from specified AWS accounts
func DownloadData(ctx context.Context, options *DownloadOptions) error {
	if options == nil {
		return fmt.Errorf("download options are required")
	}

	if len(options.Configs) == 0 {
		return fmt.Errorf("at least one configuration is required")
	}

	// Set default concurrency if not specified
	if options.Concurrency <= 0 {
		options.Concurrency = runtime.NumCPU()
	}

	credProvider := NewCredentialsProvider(options.Configs)
	
	// Determine accounts to process
	accountIDs := options.AccountIDs
	if len(accountIDs) == 0 {
		// Get configured accounts or discover from credentials
		accountIDs = getConfiguredAccounts(options.Configs)
		if len(accountIDs) == 0 {
			// Use current credentials to get account ID
			defaultCfg, err := credProvider.GetCredentials(ctx, "default")
			if err != nil {
				return fmt.Errorf("failed to get default credentials: %w", err)
			}
			
			stsClient := sts.NewFromConfig(defaultCfg)
			callerID, err := stsClient.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
			if err != nil {
				return fmt.Errorf("failed to get caller identity: %w", err)
			}
			
			accountIDs = []string{*callerID.Account}
		}
	}

	// Get storage configuration
	storageConfig := getStorageConfig(options.Configs)
	if storageConfig == nil {
		return fmt.Errorf("no storage configuration found")
	}

	storageClient, err := CreateStorageClient(ctx, storageConfig, credProvider)
	if err != nil {
		return fmt.Errorf("failed to create storage client: %w", err)
	}

	// Set default services if not specified
	services := options.Services
	if len(services) == 0 {
		allSvcs := AllServices()
		services = make([]string, len(allSvcs))
		for i, svc := range allSvcs {
			services[i] = string(svc)
		}
	}

	// Download data from each account
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, options.Concurrency)
	errChan := make(chan error, len(accountIDs))

	for _, accountID := range accountIDs {
		wg.Add(1)
		go func(accID string) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			if err := downloadAccountData(ctx, accID, options, credProvider, storageClient); err != nil {
				errChan <- fmt.Errorf("failed to download data for account %s: %w", accID, err)
			}
		}(accountID)
	}

	wg.Wait()
	close(errChan)

	// Check for errors
	var downloadErrors []error
	for err := range errChan {
		downloadErrors = append(downloadErrors, err)
	}

	if len(downloadErrors) > 0 {
		return fmt.Errorf("download failed for %d accounts: %v", len(downloadErrors), downloadErrors)
	}

	// Run indexing if not skipped
	if !options.SkipIndex {
		indexOptions := &IndexOptions{
			Configs:     options.Configs,
			Partition:   "aws", // Default partition
			AccountIDs:  accountIDs,
			Regions:     options.Regions,
			Services:    services,
			Concurrency: options.Concurrency,
		}

		if err := Index(ctx, indexOptions); err != nil {
			return fmt.Errorf("indexing failed: %w", err)
		}
	}

	return nil
}

// downloadAccountData downloads data for a specific account
func downloadAccountData(ctx context.Context, accountID string, options *DownloadOptions, credProvider *CredentialsProvider, storageClient StorageClient) error {
	// Get credentials for the account
	awsConfig, err := credProvider.GetCredentials(ctx, accountID)
	if err != nil {
		return fmt.Errorf("failed to get credentials for account %s: %w", accountID, err)
	}

	// Get regions to process
	regions := options.Regions
	if len(regions) == 0 {
		regions = getDefaultRegions()
	}

	// Create collectors for each service
	collectors := createCollectors(awsConfig, accountID)

	// Process each service
	for _, service := range options.Services {
		collector, exists := collectors[service]
		if !exists {
			continue // Skip unsupported services
		}

		// Collect data from all regions for this service
		for _, region := range regions {
			if err := collectServiceData(ctx, collector, service, region, accountID, storageClient); err != nil {
				return fmt.Errorf("failed to collect %s data from region %s: %w", service, region, err)
			}
		}
	}

	return nil
}

// collectServiceData collects data for a specific service in a region
func collectServiceData(ctx context.Context, collector ServiceCollector, service, region, accountID string, storageClient StorageClient) error {
	resources, err := collector.CollectResources(ctx, region)
	if err != nil {
		return fmt.Errorf("failed to collect resources: %w", err)
	}

	// Store each resource
	for _, resource := range resources {
		// Build storage path
		path := buildResourcePath("aws", accountID, service, region, resource.ResourceType, resource.ResourceID)
		
		// Add collection metadata
		resource.CollectedAt = time.Now()
		resource.Account = accountID
		resource.Service = service
		resource.Region = region

		// Marshal to JSON
		data, err := json.Marshal(resource)
		if err != nil {
			return fmt.Errorf("failed to marshal resource %s: %w", resource.ARN, err)
		}

		// Store the resource
		if err := storageClient.Store(ctx, path+"/metadata.json", data); err != nil {
			return fmt.Errorf("failed to store resource %s: %w", resource.ARN, err)
		}
	}

	return nil
}

// buildResourcePath builds the storage path for a resource
func buildResourcePath(partition, account, service, region, resourceType, resourceID string) string {
	if region == "" {
		// Global service
		return fmt.Sprintf("%s/%s/accounts/%s/%s/%s/%s", partition, partition, account, service, resourceType, resourceID)
	}
	return fmt.Sprintf("%s/%s/accounts/%s/%s/%s/%s/%s", partition, partition, account, service, region, resourceType, resourceID)
}

// getConfiguredAccounts extracts configured account IDs from configs
func getConfiguredAccounts(configs []*Config) []string {
	for i := len(configs) - 1; i >= 0; i-- {
		config := configs[i]
		if config.Accounts != nil && len(config.Accounts.Included) > 0 {
			return config.Accounts.Included
		}
	}
	return []string{}
}

// getStorageConfig extracts storage configuration from configs
func getStorageConfig(configs []*Config) *StorageConfig {
	for i := len(configs) - 1; i >= 0; i-- {
		config := configs[i]
		if config.Storage != nil {
			return config.Storage
		}
	}
	return nil
}

// getDefaultRegions returns default AWS regions to process
func getDefaultRegions() []string {
	return []string{
		"us-east-1", "us-east-2", "us-west-1", "us-west-2",
		"eu-west-1", "eu-west-2", "eu-west-3", "eu-central-1",
		"ap-northeast-1", "ap-northeast-2", "ap-southeast-1", "ap-southeast-2",
		"ap-south-1", "ca-central-1", "sa-east-1",
	}
}