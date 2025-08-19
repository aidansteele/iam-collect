// Package iamcollect provides functionality to collect IAM data from AWS accounts
// and create searchable indexes. This package is designed to be used in AWS Lambda
// functions and other Go applications.
//
// The main functions are:
//   - DownloadData: Downloads IAM data from AWS accounts
//   - Index: Creates search-friendly JSON indexes of collected data
//
// Example usage:
//
//	config := &iamcollect.Config{
//		Storage: &iamcollect.StorageConfig{
//			Type: "s3",
//			Bucket: "my-iam-data-bucket",
//			Prefix: "iam-data/",
//		},
//	}
//
//	options := &iamcollect.DownloadOptions{
//		Configs: []*iamcollect.Config{config},
//		Services: []string{"iam", "s3"},
//		Concurrency: 4,
//	}
//
//	err := iamcollect.DownloadData(context.Background(), options)
//	if err != nil {
//		log.Fatal(err)
//	}
package iamcollect