# IAM Collect Go Implementation - Usage Guide

This document provides a comprehensive guide on how to use the Go implementation of iam-collect functionality in AWS Lambda functions and other Go applications.

## Overview

The Go package provides two main functions that mirror the TypeScript implementation:

1. **`DownloadData()`** - Downloads IAM data from AWS accounts
2. **`Index()`** - Creates search-friendly JSON indexes of collected data

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
    config := &iamcollect.Config{
        Storage: &iamcollect.StorageConfig{
            Type: "s3",
            Bucket: "my-iam-data-bucket",
            Prefix: "iam-data/",
        },
    }

    options := &iamcollect.DownloadOptions{
        Configs:     []*iamcollect.Config{config},
        Services:    []string{"iam", "s3"},
        Concurrency: 4,
    }

    err := iamcollect.DownloadData(context.Background(), options)
    if err != nil {
        log.Fatal(err)
    }
}
```

## AWS Lambda Deployment

### Option 1: Direct Lambda Handler

Create a `main.go` file for your Lambda function:

```go
package main

import (
    "github.com/aidansteele/iam-collect"
)

func main() {
    // For download functionality
    iamcollect.StartDirectDownloadLambda()
    
    // OR for indexing functionality  
    // iamcollect.StartDirectIndexLambda()
}
```

Build and deploy:

```bash
GOOS=linux GOARCH=amd64 go build -o bootstrap main.go
zip lambda-function.zip bootstrap
aws lambda create-function \
    --function-name iam-collect-download \
    --runtime provided.al2 \
    --role arn:aws:iam::123456789012:role/lambda-execution-role \
    --handler bootstrap \
    --zip-file fileb://lambda-function.zip
```

### Option 2: API Gateway Integration

```go
package main

import (
    "github.com/aidansteele/iam-collect"
)

func main() {
    // For API Gateway integration
    iamcollect.StartDownloadLambda()
}
```

## Lambda Event Payloads

### Download Event Payload

```json
{
    "config": {
        "storage": {
            "type": "s3",
            "bucket": "my-iam-data-bucket",
            "prefix": "iam-data/"
        },
        "auth": {
            "role": {
                "pathAndName": "IAMCollectRole"
            }
        }
    },
    "services": ["iam", "s3"],
    "regions": ["us-east-1", "us-west-2"],
    "concurrency": 4,
    "skipIndex": false
}
```

### Index Event Payload

```json
{
    "config": {
        "storage": {
            "type": "s3",
            "bucket": "my-iam-data-bucket",
            "prefix": "iam-data/"
        }
    },
    "partition": "aws",
    "services": ["iam", "s3"],
    "concurrency": 4
}
```

## Configuration Options

### Storage Configuration

#### File Storage
```go
storage := &iamcollect.StorageConfig{
    Type: "file",
    Path: "/tmp/iam-data",
}
```

#### S3 Storage
```go
storage := &iamcollect.StorageConfig{
    Type:   "s3",
    Bucket: "my-bucket",
    Prefix: "iam-data/",
    Region: "us-east-1",
}
```

### Authentication Configuration

#### Default Credentials
```go
// Uses default AWS credential chain
config := &iamcollect.Config{
    Storage: storageConfig,
}
```

#### Profile-based
```go
config := &iamcollect.Config{
    Auth: &iamcollect.AuthConfig{
        Profile: "my-aws-profile",
    },
    Storage: storageConfig,
}
```

#### Cross-account Role Assumption
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
    Storage: storageConfig,
}
```

### Multi-Account Configuration
```go
config := &iamcollect.Config{
    Accounts: &iamcollect.AccountsConfig{
        Included: []string{
            "123456789012",
            "987654321098",
        },
    },
    Storage: storageConfig,
}
```

## Supported AWS Services

The current implementation supports collecting data from:

- **IAM**: Users, roles, and policies
- **S3**: Buckets with metadata and tags
- **EC2**: Instances with state and configuration

Additional services can be easily added by implementing the `ServiceCollector` interface.

## Data Organization

Data is stored in a hierarchical structure:

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

## Lambda IAM Permissions

Your Lambda execution role needs the following permissions:

```json
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Action": [
                "iam:GetRole",
                "iam:GetUser",
                "iam:GetPolicy",
                "iam:ListRoles",
                "iam:ListUsers",
                "iam:ListPolicies",
                "iam:ListRoleTags",
                "iam:ListUserTags",
                "iam:ListPolicyTags",
                "s3:ListBucket",
                "s3:GetObject",
                "s3:PutObject",
                "s3:GetBucketLocation",
                "s3:GetBucketTagging",
                "sts:GetCallerIdentity",
                "sts:AssumeRole"
            ],
            "Resource": "*"
        }
    ]
}
```

## Error Handling

The package provides comprehensive error handling:

```go
err := iamcollect.DownloadData(ctx, options)
if err != nil {
    // Check if it's a partial failure
    if strings.Contains(err.Error(), "download failed for") {
        log.Printf("Some accounts failed: %v", err)
        // You might want to continue with indexing
    } else {
        log.Fatalf("Critical error: %v", err)
    }
}
```

## Performance Tuning

### Concurrency
```go
options := &iamcollect.DownloadOptions{
    Concurrency: 8, // Adjust based on your needs and AWS limits
    // ... other options
}
```

### Region Optimization
```go
options := &iamcollect.DownloadOptions{
    Regions: []string{
        "us-east-1",
        "us-west-2",
        // Only specify regions you need
    },
    // ... other options
}
```

### Service Selection
```go
options := &iamcollect.DownloadOptions{
    Services: []string{
        "iam", // Always useful
        "s3",  // Add only services you need
    },
    // ... other options
}
```

## Example: Complete Lambda Function

```go
package main

import (
    "context"
    "encoding/json"
    
    "github.com/aws/aws-lambda-go/events"
    "github.com/aws/aws-lambda-go/lambda"
    "github.com/aidansteele/iam-collect"
)

type CustomEvent struct {
    BucketName string   `json:"bucketName"`
    Accounts   []string `json:"accounts"`
    Services   []string `json:"services"`
}

func handleCustomEvent(ctx context.Context, event CustomEvent) error {
    config := &iamcollect.Config{
        Storage: &iamcollect.StorageConfig{
            Type:   "s3",
            Bucket: event.BucketName,
            Prefix: "iam-data/",
        },
        Accounts: &iamcollect.AccountsConfig{
            Included: event.Accounts,
        },
    }

    options := &iamcollect.DownloadOptions{
        Configs:     []*iamcollect.Config{config},
        Services:    event.Services,
        Concurrency: 4,
    }

    return iamcollect.DownloadData(ctx, options)
}

func main() {
    lambda.Start(handleCustomEvent)
}
```

This implementation provides a complete, production-ready Go package that can be easily integrated into AWS Lambda functions or any other Go application.