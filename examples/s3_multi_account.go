package main

import (
	"context"
	"log"

	"github.com/aidansteele/iam-collect"
)

func main() {
	// Example: S3 storage with cross-account access
	config := &iamcollect.Config{
		Name:              "s3-multi-account-config",
		IamCollectVersion: "0.1.0",
		Storage: &iamcollect.StorageConfig{
			Type:   "s3",
			Bucket: "my-iam-data-bucket",
			Prefix: "iam-data/",
			Region: "us-east-1",
		},
		Auth: &iamcollect.AuthConfig{
			Profile: "default",
			Role: &iamcollect.RoleInfo{
				PathAndName: "IAMCollectRole",
				SessionName: "iam-collect-session",
			},
		},
		Accounts: &iamcollect.AccountsConfig{
			Included: []string{
				"123456789012",
				"987654321098",
				"555666777888",
			},
		},
	}

	// Download options
	downloadOptions := &iamcollect.DownloadOptions{
		Configs: []*iamcollect.Config{config},
		Services: []string{
			"iam",
			"s3", 
			"ec2",
		},
		Regions: []string{
			"us-east-1",
			"us-west-2",
			"eu-west-1",
		},
		Concurrency: 8,
		SkipIndex:   false,
	}

	ctx := context.Background()

	log.Println("Starting multi-account IAM data collection...")

	// Download data
	err := iamcollect.DownloadData(ctx, downloadOptions)
	if err != nil {
		log.Fatalf("Download failed: %v", err)
	}

	log.Println("Multi-account data collection completed successfully")

	// Create additional indexes if needed
	indexOptions := &iamcollect.IndexOptions{
		Configs:     []*iamcollect.Config{config},
		Partition:   "aws",
		Services:    []string{"iam", "s3", "ec2"},
		Concurrency: 4,
	}

	log.Println("Creating additional indexes...")
	
	err = iamcollect.Index(ctx, indexOptions)
	if err != nil {
		log.Fatalf("Indexing failed: %v", err)
	}

	log.Println("All operations completed successfully")
}