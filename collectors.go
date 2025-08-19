package iamcollect

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
)

// ServiceCollector defines the interface for collecting resources from AWS services
type ServiceCollector interface {
	CollectResources(ctx context.Context, region string) ([]*ResourceMetadata, error)
}

// IAMCollector collects IAM resources
type IAMCollector struct {
	config aws.Config
	accountID string
}

// NewIAMCollector creates a new IAM collector
func NewIAMCollector(config aws.Config, accountID string) *IAMCollector {
	return &IAMCollector{
		config:    config,
		accountID: accountID,
	}
}

// CollectResources collects IAM resources (global service, region is ignored)
func (c *IAMCollector) CollectResources(ctx context.Context, region string) ([]*ResourceMetadata, error) {
	client := iam.NewFromConfig(c.config)
	var resources []*ResourceMetadata

	// Collect users
	userResources, err := c.collectUsers(ctx, client)
	if err != nil {
		return nil, err
	}
	resources = append(resources, userResources...)

	// Collect roles
	roleResources, err := c.collectRoles(ctx, client)
	if err != nil {
		return nil, err
	}
	resources = append(resources, roleResources...)

	// Collect policies
	policyResources, err := c.collectPolicies(ctx, client)
	if err != nil {
		return nil, err
	}
	resources = append(resources, policyResources...)

	return resources, nil
}

// collectUsers collects IAM users
func (c *IAMCollector) collectUsers(ctx context.Context, client *iam.Client) ([]*ResourceMetadata, error) {
	var resources []*ResourceMetadata

	paginator := iam.NewListUsersPaginator(client, &iam.ListUsersInput{})
	for paginator.HasMorePages() {
		output, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, user := range output.Users {
			resource := &ResourceMetadata{
				ARN:          *user.Arn,
				Account:      c.accountID,
				Service:      "iam",
				ResourceType: "user",
				ResourceID:   *user.UserName,
				Data: map[string]interface{}{
					"UserName":   *user.UserName,
					"UserId":     *user.UserId,
					"Path":       *user.Path,
					"CreateDate": user.CreateDate,
				},
			}

			// Get user tags
			tagsOutput, err := client.ListUserTags(ctx, &iam.ListUserTagsInput{
				UserName: user.UserName,
			})
			if err == nil && len(tagsOutput.Tags) > 0 {
				tags := make(map[string]string)
				for _, tag := range tagsOutput.Tags {
					tags[*tag.Key] = *tag.Value
				}
				resource.Tags = tags
			}

			resources = append(resources, resource)
		}
	}

	return resources, nil
}

// collectRoles collects IAM roles
func (c *IAMCollector) collectRoles(ctx context.Context, client *iam.Client) ([]*ResourceMetadata, error) {
	var resources []*ResourceMetadata

	paginator := iam.NewListRolesPaginator(client, &iam.ListRolesInput{})
	for paginator.HasMorePages() {
		output, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, role := range output.Roles {
			resource := &ResourceMetadata{
				ARN:          *role.Arn,
				Account:      c.accountID,
				Service:      "iam",
				ResourceType: "role",
				ResourceID:   *role.RoleName,
				Data: map[string]interface{}{
					"RoleName":                 *role.RoleName,
					"RoleId":                   *role.RoleId,
					"Path":                     *role.Path,
					"CreateDate":               role.CreateDate,
					"AssumeRolePolicyDocument": role.AssumeRolePolicyDocument,
				},
			}

			// Get role tags
			tagsOutput, err := client.ListRoleTags(ctx, &iam.ListRoleTagsInput{
				RoleName: role.RoleName,
			})
			if err == nil && len(tagsOutput.Tags) > 0 {
				tags := make(map[string]string)
				for _, tag := range tagsOutput.Tags {
					tags[*tag.Key] = *tag.Value
				}
				resource.Tags = tags
			}

			resources = append(resources, resource)
		}
	}

	return resources, nil
}

// collectPolicies collects IAM policies
func (c *IAMCollector) collectPolicies(ctx context.Context, client *iam.Client) ([]*ResourceMetadata, error) {
	var resources []*ResourceMetadata

	// Collect customer managed policies
	paginator := iam.NewListPoliciesPaginator(client, &iam.ListPoliciesInput{
		Scope: "Local", // Only customer managed policies
	})
	for paginator.HasMorePages() {
		output, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, policy := range output.Policies {
			resource := &ResourceMetadata{
				ARN:          *policy.Arn,
				Account:      c.accountID,
				Service:      "iam",
				ResourceType: "policy",
				ResourceID:   *policy.PolicyName,
				Data: map[string]interface{}{
					"PolicyName":   *policy.PolicyName,
					"PolicyId":     *policy.PolicyId,
					"Path":         *policy.Path,
					"CreateDate":   policy.CreateDate,
					"UpdateDate":   policy.UpdateDate,
					"Description":  policy.Description,
				},
			}

			// Get policy tags
			tagsOutput, err := client.ListPolicyTags(ctx, &iam.ListPolicyTagsInput{
				PolicyArn: policy.Arn,
			})
			if err == nil && len(tagsOutput.Tags) > 0 {
				tags := make(map[string]string)
				for _, tag := range tagsOutput.Tags {
					tags[*tag.Key] = *tag.Value
				}
				resource.Tags = tags
			}

			resources = append(resources, resource)
		}
	}

	return resources, nil
}

// S3Collector collects S3 resources
type S3Collector struct {
	config aws.Config
	accountID string
}

// NewS3Collector creates a new S3 collector
func NewS3Collector(config aws.Config, accountID string) *S3Collector {
	return &S3Collector{
		config:    config,
		accountID: accountID,
	}
}

// CollectResources collects S3 buckets (global service, region is ignored)
func (c *S3Collector) CollectResources(ctx context.Context, region string) ([]*ResourceMetadata, error) {
	client := s3.NewFromConfig(c.config)
	var resources []*ResourceMetadata

	// List buckets
	output, err := client.ListBuckets(ctx, &s3.ListBucketsInput{})
	if err != nil {
		return nil, err
	}

	for _, bucket := range output.Buckets {
		// Get bucket region
		locationOutput, err := client.GetBucketLocation(ctx, &s3.GetBucketLocationInput{
			Bucket: bucket.Name,
		})
		if err != nil {
			continue // Skip buckets we can't access
		}

		bucketRegion := "us-east-1" // Default for us-east-1
		if locationOutput.LocationConstraint != "" {
			bucketRegion = string(locationOutput.LocationConstraint)
		}

		bucketARN := "arn:aws:s3:::" + *bucket.Name

		resource := &ResourceMetadata{
			ARN:          bucketARN,
			Account:      c.accountID,
			Service:      "s3",
			Region:       bucketRegion,
			ResourceType: "bucket",
			ResourceID:   *bucket.Name,
			Data: map[string]interface{}{
				"Name":               *bucket.Name,
				"CreationDate":       bucket.CreationDate,
				"LocationConstraint": locationOutput.LocationConstraint,
			},
		}

		// Get bucket tags
		tagsOutput, err := client.GetBucketTagging(ctx, &s3.GetBucketTaggingInput{
			Bucket: bucket.Name,
		})
		if err == nil && len(tagsOutput.TagSet) > 0 {
			tags := make(map[string]string)
			for _, tag := range tagsOutput.TagSet {
				tags[*tag.Key] = *tag.Value
			}
			resource.Tags = tags
		}

		resources = append(resources, resource)
	}

	return resources, nil
}

// createCollectors creates service collectors for the given AWS config and account
func createCollectors(awsConfig aws.Config, accountID string) map[string]ServiceCollector {
	collectors := make(map[string]ServiceCollector)

	collectors["iam"] = NewIAMCollector(awsConfig, accountID)
	collectors["s3"] = NewS3Collector(awsConfig, accountID)
	collectors["ec2"] = NewEC2Collector(awsConfig, accountID)
	
	// Add more collectors for other services as needed
	// collectors["lambda"] = NewLambdaCollector(awsConfig, accountID)
	// collectors["dynamodb"] = NewDynamoDBCollector(awsConfig, accountID)
	// etc.

	return collectors
}

// Additional collectors can be implemented here following the same pattern
// For example:

// EC2Collector collects EC2 resources
type EC2Collector struct {
	config aws.Config
	accountID string
}

// NewEC2Collector creates a new EC2 collector
func NewEC2Collector(config aws.Config, accountID string) *EC2Collector {
	return &EC2Collector{
		config:    config,
		accountID: accountID,
	}
}

// CollectResources collects EC2 instances in the specified region
func (c *EC2Collector) CollectResources(ctx context.Context, region string) ([]*ResourceMetadata, error) {
	// Set region in config
	regionConfig := c.config.Copy()
	regionConfig.Region = region
	
	client := ec2.NewFromConfig(regionConfig)
	var resources []*ResourceMetadata

	// Describe instances
	paginator := ec2.NewDescribeInstancesPaginator(client, &ec2.DescribeInstancesInput{})
	for paginator.HasMorePages() {
		output, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, reservation := range output.Reservations {
			for _, instance := range reservation.Instances {
				instanceARN := "arn:aws:ec2:" + region + ":" + c.accountID + ":instance/" + *instance.InstanceId

				resource := &ResourceMetadata{
					ARN:          instanceARN,
					Account:      c.accountID,
					Service:      "ec2",
					Region:       region,
					ResourceType: "instance",
					ResourceID:   *instance.InstanceId,
					Data: map[string]interface{}{
						"InstanceId":     *instance.InstanceId,
						"InstanceType":   instance.InstanceType,
						"State":          instance.State,
						"LaunchTime":     instance.LaunchTime,
						"ImageId":        instance.ImageId,
						"SubnetId":       instance.SubnetId,
						"VpcId":          instance.VpcId,
					},
				}

				// Extract tags
				if len(instance.Tags) > 0 {
					tags := make(map[string]string)
					for _, tag := range instance.Tags {
						if tag.Key != nil && tag.Value != nil {
							tags[*tag.Key] = *tag.Value
						}
					}
					resource.Tags = tags
				}

				resources = append(resources, resource)
			}
		}
	}

	return resources, nil
}