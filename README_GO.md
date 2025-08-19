# IAM Collect Go Package

This Go package provides functionality to collect IAM data from AWS accounts and create searchable indexes. It's designed to be used in AWS Lambda functions and other Go applications.

## Features

- **Download IAM Data**: Collect IAM resources (users, roles, policies) and other AWS resources from multiple accounts
- **Create Indexes**: Build search-friendly JSON indexes for fast lookups
- **Multiple Storage Backends**: Support for file system and S3 storage
- **AWS Lambda Ready**: Built-in Lambda handlers for serverless deployment
- **Concurrent Processing**: Configurable concurrency for improved performance
- **Flexible Authentication**: Support for profiles, role assumption, and cross-account access

## Installation

```bash
go get github.com/aidansteele/iam-collect
```

## Quick Start

### Basic Usage

```go
package main

import (
    "context"
    "log"
    
    "github.com/aidansteele/iam-collect"
)

func main() {
    // Configure storage (S3 example)
    config := &iamcollect.Config{
        Storage: &iamcollect.StorageConfig{
            Type:   "s3",
            Bucket: "my-iam-data-bucket",
            Prefix: "iam-data/",
            Region: "us-east-1",
        },
    }

    // Download IAM data
    downloadOptions := &iamcollect.DownloadOptions{
        Configs:     []*iamcollect.Config{config},
        Services:    []string{"iam", "s3"},
        Concurrency: 4,
    }

    err := iamcollect.DownloadData(context.Background(), downloadOptions)
    if err != nil {
        log.Fatal("Download failed:", err)
    }

    // Create indexes
    indexOptions := &iamcollect.IndexOptions{
        Configs:     []*iamcollect.Config{config},
        Partition:   "aws",
        Services:    []string{"iam", "s3"},
        Concurrency: 4,
    }

    err = iamcollect.Index(context.Background(), indexOptions)
    if err != nil {
        log.Fatal("Indexing failed:", err)
    }

    log.Println("IAM data collection and indexing completed successfully")
}
```

### AWS Lambda Usage

```go
package main

import (
    "github.com/aidansteele/iam-collect"
)

func main() {
    // Start Lambda function for downloading data
    iamcollect.StartDirectDownloadLambda()
}
```

## Configuration

The package supports flexible configuration through the `Config` struct:

```go
config := &iamcollect.Config{
    Name:              "my-config",
    IamCollectVersion: "0.1.0",
    Storage: &iamcollect.StorageConfig{
        Type:   "s3",        // "s3" or "file"
        Bucket: "my-bucket", // for S3
        Prefix: "data/",     // for S3
        Path:   "/tmp/data", // for file storage
        Region: "us-east-1",
    },
    Auth: &iamcollect.AuthConfig{
        Profile: "my-profile",
        Role: &iamcollect.RoleInfo{
            PathAndName: "IAMCollectRole",
            SessionName: "iam-collect-session",
        },
    },
    Accounts: &iamcollect.AccountsConfig{
        Included: []string{"123456789012", "987654321098"},
    },
}
```

## Supported AWS Services

The package supports collecting data from the following AWS services:

- IAM (Identity and Access Management)
- S3 (Simple Storage Service)
- EC2 (Elastic Compute Cloud)
- Lambda
- DynamoDB
- KMS (Key Management Service)
- SNS (Simple Notification Service)
- SQS (Simple Queue Service)
- And more...

## Storage Backends

### File System Storage

```go
config := &iamcollect.Config{
    Storage: &iamcollect.StorageConfig{
        Type: "file",
        Path: "/path/to/iam-data",
    },
}
```

### S3 Storage

```go
config := &iamcollect.Config{
    Storage: &iamcollect.StorageConfig{
        Type:   "s3",
        Bucket: "my-iam-data-bucket",
        Prefix: "iam-data/",
        Region: "us-east-1",
    },
}
```

## Authentication

The package supports various authentication methods:

### Default Credentials

```go
config := &iamcollect.Config{
    // Uses default credential chain
}
```

### Profile-based Authentication

```go
config := &iamcollect.Config{
    Auth: &iamcollect.AuthConfig{
        Profile: "my-aws-profile",
    },
}
```

### Cross-Account Role Assumption

```go
config := &iamcollect.Config{
    Auth: &iamcollect.AuthConfig{
        InitialRole: &iamcollect.RoleRef{
            ARN: "arn:aws:iam::123456789012:role/InitialRole",
        },
        Role: &iamcollect.RoleInfo{
            PathAndName: "TargetAccountRole",
            ExternalID:  "unique-external-id",
        },
    },
}
```

## AWS Lambda Deployment

The package provides multiple Lambda handler options:

### API Gateway Handler

```go
func main() {
    iamcollect.StartDownloadLambda() // For API Gateway integration
}
```

### Direct Event Handler

```go
func main() {
    iamcollect.StartDirectDownloadLambda() // For direct Lambda invocation
}
```

### Custom Handler

```go
import (
    "github.com/aws/aws-lambda-go/lambda"
    "github.com/aidansteele/iam-collect"
)

func main() {
    lambda.Start(iamcollect.DirectDownloadHandler)
}
```

## Example Lambda Payload

```json
{
    "config": {
        "storage": {
            "type": "s3",
            "bucket": "my-iam-data-bucket",
            "prefix": "iam-data/"
        }
    },
    "services": ["iam", "s3"],
    "concurrency": 4,
    "skipIndex": false
}
```

## Data Organization

The collected data is organized in a hierarchical structure:

```
aws/
└── aws/
    ├── accounts/
    │   └── 123456789012/
    │       ├── iam/
    │       │   ├── user/
    │       │   │   └── username/
    │       │   │       └── metadata.json
    │       │   └── role/
    │       │       └── rolename/
    │       │           └── metadata.json
    │       └── s3/
    │           └── us-east-1/
    │               └── bucket/
    │                   └── bucketname/
    │                       └── metadata.json
    └── indexes/
        ├── global/
        │   └── index.json
        ├── accounts/
        │   └── 123456789012/
        │       └── index.json
        └── services/
            └── iam/
                └── index.json
```

## Testing

```bash
go test ./...
```

Note: Some tests require valid AWS credentials and permissions to run successfully.

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

## License

This project is licensed under the same license as the original iam-collect project.