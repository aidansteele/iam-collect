package iamcollect

import (
	"time"
)

// Config represents the main configuration for iam-collect
type Config struct {
	Name              string                          `json:"name,omitempty"`
	IamCollectVersion string                          `json:"iamCollectVersion"`
	Storage           *StorageConfig                  `json:"storage,omitempty"`
	Auth              *AuthConfig                     `json:"auth,omitempty"`
	Accounts          *AccountsConfig                 `json:"accounts,omitempty"`
	AccountConfigs    map[string]*AccountConfig       `json:"accountConfigs,omitempty"`
	ServiceConfigs    map[string]*ServiceConfig       `json:"serviceConfigs,omitempty"`
}

// StorageConfig defines storage backend configuration
type StorageConfig struct {
	Type     string                 `json:"type"` // "file" or "s3"
	Path     string                 `json:"path,omitempty"`     // for file storage
	Bucket   string                 `json:"bucket,omitempty"`   // for s3 storage
	Prefix   string                 `json:"prefix,omitempty"`   // for s3 storage
	Region   string                 `json:"region,omitempty"`   // for s3 storage
	Endpoint string                 `json:"endpoint,omitempty"` // for s3 storage
	Auth     *AuthConfig            `json:"auth,omitempty"`
}

// AuthConfig defines authentication configuration
type AuthConfig struct {
	Profile     string    `json:"profile,omitempty"`
	InitialRole *RoleRef  `json:"initialRole,omitempty"`
	Role        *RoleInfo `json:"role,omitempty"`
}

// RoleRef represents a role reference (either ARN or path/name)
type RoleRef struct {
	ARN         string `json:"arn,omitempty"`
	PathAndName string `json:"pathAndName,omitempty"`
	ExternalID  string `json:"externalId,omitempty"`
	SessionName string `json:"sessionName,omitempty"`
}

// RoleInfo represents role information for assumption
type RoleInfo struct {
	PathAndName string `json:"pathAndName"`
	SessionName string `json:"sessionName,omitempty"`
	ExternalID  string `json:"externalId,omitempty"`
}

// AccountsConfig defines which accounts to include
type AccountsConfig struct {
	Included []string `json:"included,omitempty"`
}

// AccountConfig defines per-account configuration
type AccountConfig struct {
	Auth           *AuthConfig                    `json:"auth,omitempty"`
	ServiceConfigs map[string]*ServiceConfig      `json:"serviceConfigs,omitempty"`
}

// ServiceConfig defines per-service configuration
type ServiceConfig struct {
	Auth          *AuthConfig                   `json:"auth,omitempty"`
	Endpoint      string                        `json:"endpoint,omitempty"`
	RegionConfigs map[string]*RegionConfig      `json:"regionConfigs,omitempty"`
	SyncConfigs   map[string]*SyncConfig        `json:"syncConfigs,omitempty"`
}

// RegionConfig defines per-region configuration
type RegionConfig struct {
	Auth     *AuthConfig `json:"auth,omitempty"`
	Endpoint string      `json:"endpoint,omitempty"`
}

// SyncConfig defines sync-specific configuration
type SyncConfig struct {
	Custom  map[string]interface{} `json:"custom,omitempty"`
	Regions *RegionsFilter         `json:"regions,omitempty"`
	Auth    *AuthConfig            `json:"auth,omitempty"`
}

// RegionsFilter defines region filtering
type RegionsFilter struct {
	Included []string `json:"included,omitempty"`
	Excluded []string `json:"excluded,omitempty"`
}

// DownloadOptions contains options for the download operation
type DownloadOptions struct {
	Configs     []*Config
	AccountIDs  []string
	Regions     []string
	Services    []string
	Concurrency int
	SkipIndex   bool
}

// IndexOptions contains options for the index operation
type IndexOptions struct {
	Configs     []*Config
	Partition   string
	AccountIDs  []string
	Regions     []string
	Services    []string
	Concurrency int
}

// AwsService represents supported AWS services
type AwsService string

const (
	ServiceAPIGateway       AwsService = "apigateway"
	ServiceBackup           AwsService = "backup"
	ServiceDynamoDB         AwsService = "dynamodb"
	ServiceEC2              AwsService = "ec2"
	ServiceECR              AwsService = "ecr"
	ServiceEFS              AwsService = "elasticfilesystem"
	ServiceGlacier          AwsService = "glacier"
	ServiceGlue             AwsService = "glue"
	ServiceIAM              AwsService = "iam"
	ServiceKMS              AwsService = "kms"
	ServiceLambda           AwsService = "lambda"
	ServiceOrganizations    AwsService = "organizations"
	ServiceRAM              AwsService = "ram"
	ServiceS3               AwsService = "s3"
	ServiceS3Express        AwsService = "s3express"
	ServiceS3Outposts       AwsService = "s3outposts"
	ServiceS3Tables         AwsService = "s3tables"
	ServiceSecretsManager   AwsService = "secretsmanager"
	ServiceSNS              AwsService = "sns"
	ServiceSQS              AwsService = "sqs"
	ServiceSSO              AwsService = "sso"
)

// AllServices returns all supported AWS services
func AllServices() []AwsService {
	return []AwsService{
		ServiceAPIGateway,
		ServiceBackup,
		ServiceDynamoDB,
		ServiceEC2,
		ServiceECR,
		ServiceEFS,
		ServiceGlacier,
		ServiceGlue,
		ServiceIAM,
		ServiceKMS,
		ServiceLambda,
		ServiceOrganizations,
		ServiceRAM,
		ServiceS3,
		ServiceS3Express,
		ServiceS3Outposts,
		ServiceS3Tables,
		ServiceSecretsManager,
		ServiceSNS,
		ServiceSQS,
		ServiceSSO,
	}
}

// ResourceMetadata represents metadata about a collected resource
type ResourceMetadata struct {
	ARN           string                 `json:"arn"`
	Account       string                 `json:"account"`
	Service       string                 `json:"service"`
	Region        string                 `json:"region,omitempty"`
	ResourceType  string                 `json:"resourceType"`
	ResourceID    string                 `json:"resourceId"`
	Tags          map[string]string      `json:"tags,omitempty"`
	CollectedAt   time.Time              `json:"collectedAt"`
	Data          map[string]interface{} `json:"data"`
}

// IndexEntry represents an index entry for search
type IndexEntry struct {
	ARN          string            `json:"arn"`
	Account      string            `json:"account"`
	Service      string            `json:"service"`
	Region       string            `json:"region,omitempty"`
	ResourceType string            `json:"resourceType"`
	ResourceID   string            `json:"resourceId"`
	Tags         map[string]string `json:"tags,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}