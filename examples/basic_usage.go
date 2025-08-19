package main

import (
	"context"
	"log"

	"github.com/aidansteele/iam-collect"
)

func main() {
	// Example: Basic usage with file storage
	config := &iamcollect.Config{
		Name:              "example-config",
		IamCollectVersion: "0.1.0",
		Storage: &iamcollect.StorageConfig{
			Type: "file",
			Path: "./iam-data",
		},
	}

	// Download options
	downloadOptions := &iamcollect.DownloadOptions{
		Configs:     []*iamcollect.Config{config},
		Services:    []string{"iam", "s3"},
		Concurrency: 4,
		SkipIndex:   false, // Create indexes after download
	}

	ctx := context.Background()

	log.Println("Starting IAM data collection...")

	// Download data
	err := iamcollect.DownloadData(ctx, downloadOptions)
	if err != nil {
		log.Printf("Download completed with some errors: %v", err)
		// You might want to continue with indexing even if some downloads failed
	} else {
		log.Println("Data download completed successfully")
	}

	log.Println("IAM data collection completed")
}