package iamcollect

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime"
	"sync"
)

// Index creates search-friendly JSON indexes of the collected IAM data
func Index(ctx context.Context, options *IndexOptions) error {
	if options == nil {
		return fmt.Errorf("index options are required")
	}

	if len(options.Configs) == 0 {
		return fmt.Errorf("at least one configuration is required")
	}

	// Set default concurrency if not specified
	if options.Concurrency <= 0 {
		options.Concurrency = runtime.NumCPU()
	}

	// Set default partition
	if options.Partition == "" {
		options.Partition = "aws"
	}

	credProvider := NewCredentialsProvider(options.Configs)

	// Get storage configuration
	storageConfig := getStorageConfig(options.Configs)
	if storageConfig == nil {
		return fmt.Errorf("no storage configuration found")
	}

	storageClient, err := CreateStorageClient(ctx, storageConfig, credProvider)
	if err != nil {
		return fmt.Errorf("failed to create storage client: %w", err)
	}

	// Determine accounts to index
	accountIDs := options.AccountIDs
	if len(accountIDs) == 0 {
		accountIDs, err = storageClient.ListAccountIDs(ctx)
		if err != nil {
			return fmt.Errorf("failed to list account IDs: %w", err)
		}
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

	// Create indexes for each account and service
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, options.Concurrency)
	errChan := make(chan error, len(accountIDs)*len(services))

	for _, accountID := range accountIDs {
		for _, service := range services {
			wg.Add(1)
			go func(accID, svc string) {
				defer wg.Done()
				semaphore <- struct{}{}
				defer func() { <-semaphore }()

				if err := indexAccountService(ctx, accID, svc, options, storageClient); err != nil {
					errChan <- fmt.Errorf("failed to index %s for account %s: %w", svc, accID, err)
				}
			}(accountID, service)
		}
	}

	wg.Wait()
	close(errChan)

	// Check for errors
	var indexErrors []error
	for err := range errChan {
		indexErrors = append(indexErrors, err)
	}

	if len(indexErrors) > 0 {
		return fmt.Errorf("indexing failed for %d service/account combinations: %v", len(indexErrors), indexErrors)
	}

	// Create global indexes
	if err := createGlobalIndexes(ctx, options, storageClient, accountIDs, services); err != nil {
		return fmt.Errorf("failed to create global indexes: %w", err)
	}

	return nil
}

// indexAccountService creates indexes for a specific account and service
func indexAccountService(ctx context.Context, accountID, service string, options *IndexOptions, storageClient StorageClient) error {
	// List all resources for this account and service
	prefix := fmt.Sprintf("%s/%s/accounts/%s/%s/", options.Partition, options.Partition, accountID, service)
	
	resources, err := listResourcesInPath(ctx, storageClient, prefix)
	if err != nil {
		return fmt.Errorf("failed to list resources: %w", err)
	}

	if len(resources) == 0 {
		return nil // No resources to index
	}

	// Create service-specific index
	serviceIndex := make(map[string]*IndexEntry)
	resourcesByType := make(map[string][]*IndexEntry)

	for _, resource := range resources {
		entry := &IndexEntry{
			ARN:          resource.ARN,
			Account:      resource.Account,
			Service:      resource.Service,
			Region:       resource.Region,
			ResourceType: resource.ResourceType,
			ResourceID:   resource.ResourceID,
			Tags:         resource.Tags,
			Metadata:     extractIndexMetadata(resource),
		}

		serviceIndex[resource.ARN] = entry
		resourcesByType[resource.ResourceType] = append(resourcesByType[resource.ResourceType], entry)
	}

	// Store service index
	serviceIndexPath := fmt.Sprintf("%s/%s/indexes/accounts/%s/%s/index.json", options.Partition, options.Partition, accountID, service)
	if err := storeIndex(ctx, storageClient, serviceIndexPath, serviceIndex); err != nil {
		return fmt.Errorf("failed to store service index: %w", err)
	}

	// Store resource type indexes
	for resourceType, entries := range resourcesByType {
		typeIndex := make(map[string]*IndexEntry)
		for _, entry := range entries {
			typeIndex[entry.ARN] = entry
		}

		typeIndexPath := fmt.Sprintf("%s/%s/indexes/accounts/%s/%s/%s/index.json", options.Partition, options.Partition, accountID, service, resourceType)
		if err := storeIndex(ctx, storageClient, typeIndexPath, typeIndex); err != nil {
			return fmt.Errorf("failed to store resource type index for %s: %w", resourceType, err)
		}
	}

	return nil
}

// createGlobalIndexes creates account-wide and cross-account indexes
func createGlobalIndexes(ctx context.Context, options *IndexOptions, storageClient StorageClient, accountIDs, services []string) error {
	// Create account-wide indexes
	for _, accountID := range accountIDs {
		if err := createAccountIndex(ctx, options, storageClient, accountID, services); err != nil {
			return fmt.Errorf("failed to create account index for %s: %w", accountID, err)
		}
	}

	// Create cross-account service indexes
	for _, service := range services {
		if err := createServiceIndex(ctx, options, storageClient, service, accountIDs); err != nil {
			return fmt.Errorf("failed to create cross-account service index for %s: %w", service, err)
		}
	}

	// Create global index
	if err := createGlobalIndex(ctx, options, storageClient, accountIDs, services); err != nil {
		return fmt.Errorf("failed to create global index: %w", err)
	}

	return nil
}

// createAccountIndex creates an index for all resources in an account
func createAccountIndex(ctx context.Context, options *IndexOptions, storageClient StorageClient, accountID string, services []string) error {
	accountIndex := make(map[string]*IndexEntry)

	for _, service := range services {
		prefix := fmt.Sprintf("%s/%s/accounts/%s/%s/", options.Partition, options.Partition, accountID, service)
		resources, err := listResourcesInPath(ctx, storageClient, prefix)
		if err != nil {
			continue // Skip services with no data
		}

		for _, resource := range resources {
			entry := &IndexEntry{
				ARN:          resource.ARN,
				Account:      resource.Account,
				Service:      resource.Service,
				Region:       resource.Region,
				ResourceType: resource.ResourceType,
				ResourceID:   resource.ResourceID,
				Tags:         resource.Tags,
				Metadata:     extractIndexMetadata(resource),
			}
			accountIndex[resource.ARN] = entry
		}
	}

	accountIndexPath := fmt.Sprintf("%s/%s/indexes/accounts/%s/index.json", options.Partition, options.Partition, accountID)
	return storeIndex(ctx, storageClient, accountIndexPath, accountIndex)
}

// createServiceIndex creates a cross-account index for a service
func createServiceIndex(ctx context.Context, options *IndexOptions, storageClient StorageClient, service string, accountIDs []string) error {
	serviceIndex := make(map[string]*IndexEntry)

	for _, accountID := range accountIDs {
		prefix := fmt.Sprintf("%s/%s/accounts/%s/%s/", options.Partition, options.Partition, accountID, service)
		resources, err := listResourcesInPath(ctx, storageClient, prefix)
		if err != nil {
			continue // Skip accounts with no data for this service
		}

		for _, resource := range resources {
			entry := &IndexEntry{
				ARN:          resource.ARN,
				Account:      resource.Account,
				Service:      resource.Service,
				Region:       resource.Region,
				ResourceType: resource.ResourceType,
				ResourceID:   resource.ResourceID,
				Tags:         resource.Tags,
				Metadata:     extractIndexMetadata(resource),
			}
			serviceIndex[resource.ARN] = entry
		}
	}

	serviceIndexPath := fmt.Sprintf("%s/%s/indexes/services/%s/index.json", options.Partition, options.Partition, service)
	return storeIndex(ctx, storageClient, serviceIndexPath, serviceIndex)
}

// createGlobalIndex creates a global index of all resources
func createGlobalIndex(ctx context.Context, options *IndexOptions, storageClient StorageClient, accountIDs, services []string) error {
	globalIndex := make(map[string]*IndexEntry)

	for _, accountID := range accountIDs {
		for _, service := range services {
			prefix := fmt.Sprintf("%s/%s/accounts/%s/%s/", options.Partition, options.Partition, accountID, service)
			resources, err := listResourcesInPath(ctx, storageClient, prefix)
			if err != nil {
				continue // Skip combinations with no data
			}

			for _, resource := range resources {
				entry := &IndexEntry{
					ARN:          resource.ARN,
					Account:      resource.Account,
					Service:      resource.Service,
					Region:       resource.Region,
					ResourceType: resource.ResourceType,
					ResourceID:   resource.ResourceID,
					Tags:         resource.Tags,
					Metadata:     extractIndexMetadata(resource),
				}
				globalIndex[resource.ARN] = entry
			}
		}
	}

	globalIndexPath := fmt.Sprintf("%s/%s/indexes/global/index.json", options.Partition, options.Partition)
	return storeIndex(ctx, storageClient, globalIndexPath, globalIndex)
}

// listResourcesInPath lists all resources in a given storage path
func listResourcesInPath(ctx context.Context, storageClient StorageClient, prefix string) ([]*ResourceMetadata, error) {
	files, err := storageClient.List(ctx, prefix)
	if err != nil {
		return nil, err
	}

	var resources []*ResourceMetadata
	for _, file := range files {
		if !json.Valid([]byte(file)) && len(file) > 0 && file[len(file)-13:] == "metadata.json" {
			data, err := storageClient.Retrieve(ctx, file)
			if err != nil {
				continue // Skip files that can't be read
			}

			var resource ResourceMetadata
			if err := json.Unmarshal(data, &resource); err != nil {
				continue // Skip files that can't be parsed
			}

			resources = append(resources, &resource)
		}
	}

	return resources, nil
}

// storeIndex stores an index to storage
func storeIndex(ctx context.Context, storageClient StorageClient, path string, index map[string]*IndexEntry) error {
	data, err := json.MarshalIndent(index, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal index: %w", err)
	}

	return storageClient.Store(ctx, path, data)
}

// extractIndexMetadata extracts relevant metadata for indexing
func extractIndexMetadata(resource *ResourceMetadata) map[string]interface{} {
	metadata := make(map[string]interface{})

	// Extract commonly indexed fields from the resource data
	if resource.Data != nil {
		// Add creation date if available
		if createDate, ok := resource.Data["CreateDate"]; ok {
			metadata["createDate"] = createDate
		}
		if creationDate, ok := resource.Data["CreationDate"]; ok {
			metadata["creationDate"] = creationDate
		}

		// Add policy document if available (for IAM resources)
		if policyDoc, ok := resource.Data["PolicyDocument"]; ok {
			metadata["policyDocument"] = policyDoc
		}

		// Add resource-specific metadata based on service
		switch resource.Service {
		case "iam":
			extractIAMMetadata(resource.Data, metadata)
		case "s3":
			extractS3Metadata(resource.Data, metadata)
		case "ec2":
			extractEC2Metadata(resource.Data, metadata)
		}
	}

	metadata["collectedAt"] = resource.CollectedAt

	return metadata
}

// extractIAMMetadata extracts IAM-specific metadata
func extractIAMMetadata(data map[string]interface{}, metadata map[string]interface{}) {
	if path, ok := data["Path"]; ok {
		metadata["path"] = path
	}
	if userID, ok := data["UserId"]; ok {
		metadata["userId"] = userID
	}
	if roleID, ok := data["RoleId"]; ok {
		metadata["roleId"] = roleID
	}
}

// extractS3Metadata extracts S3-specific metadata
func extractS3Metadata(data map[string]interface{}, metadata map[string]interface{}) {
	if location, ok := data["LocationConstraint"]; ok {
		metadata["location"] = location
	}
	if versioning, ok := data["VersioningConfiguration"]; ok {
		metadata["versioning"] = versioning
	}
}

// extractEC2Metadata extracts EC2-specific metadata
func extractEC2Metadata(data map[string]interface{}, metadata map[string]interface{}) {
	if state, ok := data["State"]; ok {
		metadata["state"] = state
	}
	if instanceType, ok := data["InstanceType"]; ok {
		metadata["instanceType"] = instanceType
	}
}