package iamcollect

import (
	"context"
	"fmt"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

// CredentialsProvider provides AWS credentials for accessing accounts
type CredentialsProvider struct {
	configs []*Config
	cache   map[string]aws.Config
	mu      sync.RWMutex
}

// NewCredentialsProvider creates a new credentials provider
func NewCredentialsProvider(configs []*Config) *CredentialsProvider {
	return &CredentialsProvider{
		configs: configs,
		cache:   make(map[string]aws.Config),
	}
}

// GetCredentials returns AWS credentials for the specified account
func (cp *CredentialsProvider) GetCredentials(ctx context.Context, accountID string) (aws.Config, error) {
	cp.mu.RLock()
	if cfg, exists := cp.cache[accountID]; exists {
		cp.mu.RUnlock()
		return cfg, nil
	}
	cp.mu.RUnlock()

	cp.mu.Lock()
	defer cp.mu.Unlock()

	// Double-check after acquiring write lock
	if cfg, exists := cp.cache[accountID]; exists {
		return cfg, nil
	}

	authConfig := cp.getAuthConfig(accountID)
	cfg, err := cp.buildConfig(ctx, accountID, authConfig)
	if err != nil {
		return aws.Config{}, fmt.Errorf("failed to build config for account %s: %w", accountID, err)
	}

	cp.cache[accountID] = cfg
	return cfg, nil
}

// getAuthConfig finds the auth configuration for an account
func (cp *CredentialsProvider) getAuthConfig(accountID string) *AuthConfig {
	var authConfig *AuthConfig

	// Start with global auth config
	for _, config := range cp.configs {
		if config.Auth != nil {
			authConfig = mergeAuthConfig(authConfig, config.Auth)
		}
	}

	// Apply account-specific auth config
	for _, config := range cp.configs {
		if config.AccountConfigs != nil {
			if accountConfig, exists := config.AccountConfigs[accountID]; exists && accountConfig.Auth != nil {
				authConfig = mergeAuthConfig(authConfig, accountConfig.Auth)
			}
		}
	}

	return authConfig
}

// buildConfig builds AWS config for the account
func (cp *CredentialsProvider) buildConfig(ctx context.Context, accountID string, authConfig *AuthConfig) (aws.Config, error) {
	var cfg aws.Config
	var err error

	// Build base config
	configOptions := []func(*config.LoadOptions) error{
		config.WithRegion("us-east-1"), // Default region
	}

	if authConfig != nil && authConfig.Profile != "" {
		configOptions = append(configOptions, config.WithSharedConfigProfile(authConfig.Profile))
	}

	cfg, err = config.LoadDefaultConfig(ctx, configOptions...)
	if err != nil {
		return aws.Config{}, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Handle initial role assumption
	if authConfig != nil && authConfig.InitialRole != nil {
		cfg, err = cp.assumeInitialRole(ctx, cfg, authConfig.InitialRole)
		if err != nil {
			return aws.Config{}, fmt.Errorf("failed to assume initial role: %w", err)
		}
	}

	// Handle target account role assumption
	if authConfig != nil && authConfig.Role != nil {
		cfg, err = cp.assumeAccountRole(ctx, cfg, accountID, authConfig.Role)
		if err != nil {
			return aws.Config{}, fmt.Errorf("failed to assume role in account %s: %w", accountID, err)
		}
	}

	// Verify the credentials are for the correct account
	if err := cp.verifyAccount(ctx, cfg, accountID); err != nil {
		return aws.Config{}, fmt.Errorf("account verification failed: %w", err)
	}

	return cfg, nil
}

// assumeInitialRole assumes the initial role if specified
func (cp *CredentialsProvider) assumeInitialRole(ctx context.Context, cfg aws.Config, roleRef *RoleRef) (aws.Config, error) {
	roleARN := roleRef.ARN
	if roleARN == "" && roleRef.PathAndName != "" {
		// Get current account ID to build ARN
		stsClient := sts.NewFromConfig(cfg)
		callerID, err := stsClient.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
		if err != nil {
			return aws.Config{}, fmt.Errorf("failed to get caller identity: %w", err)
		}
		roleARN = fmt.Sprintf("arn:aws:iam::%s:role/%s", *callerID.Account, roleRef.PathAndName)
	}

	return cp.assumeRole(ctx, cfg, roleARN, roleRef.SessionName, roleRef.ExternalID)
}

// assumeAccountRole assumes a role in the target account
func (cp *CredentialsProvider) assumeAccountRole(ctx context.Context, cfg aws.Config, accountID string, roleInfo *RoleInfo) (aws.Config, error) {
	roleARN := fmt.Sprintf("arn:aws:iam::%s:role/%s", accountID, roleInfo.PathAndName)
	return cp.assumeRole(ctx, cfg, roleARN, roleInfo.SessionName, roleInfo.ExternalID)
}

// assumeRole performs the actual role assumption
func (cp *CredentialsProvider) assumeRole(ctx context.Context, cfg aws.Config, roleARN, sessionName, externalID string) (aws.Config, error) {
	if sessionName == "" {
		sessionName = "iam-collect-session"
	}

	stsClient := sts.NewFromConfig(cfg)
	
	input := &sts.AssumeRoleInput{
		RoleArn:         aws.String(roleARN),
		RoleSessionName: aws.String(sessionName),
	}
	
	if externalID != "" {
		input.ExternalId = aws.String(externalID)
	}

	result, err := stsClient.AssumeRole(ctx, input)
	if err != nil {
		return aws.Config{}, fmt.Errorf("failed to assume role %s: %w", roleARN, err)
	}

	// Create new config with assumed role credentials
	newCfg := cfg.Copy()
	newCfg.Credentials = credentials.NewStaticCredentialsProvider(
		*result.Credentials.AccessKeyId,
		*result.Credentials.SecretAccessKey,
		*result.Credentials.SessionToken,
	)

	return newCfg, nil
}

// verifyAccount verifies that the credentials are for the expected account
func (cp *CredentialsProvider) verifyAccount(ctx context.Context, cfg aws.Config, expectedAccountID string) error {
	stsClient := sts.NewFromConfig(cfg)
	callerID, err := stsClient.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return fmt.Errorf("failed to get caller identity: %w", err)
	}

	if *callerID.Account != expectedAccountID {
		return fmt.Errorf("credentials are for account %s, expected %s", *callerID.Account, expectedAccountID)
	}

	return nil
}

// mergeAuthConfig merges two auth configurations
func mergeAuthConfig(base, override *AuthConfig) *AuthConfig {
	if base == nil {
		if override == nil {
			return nil
		}
		return &AuthConfig{
			Profile:     override.Profile,
			InitialRole: override.InitialRole,
			Role:        override.Role,
		}
	}

	if override == nil {
		return &AuthConfig{
			Profile:     base.Profile,
			InitialRole: base.InitialRole,
			Role:        base.Role,
		}
	}

	result := &AuthConfig{
		Profile:     base.Profile,
		InitialRole: base.InitialRole,
		Role:        base.Role,
	}

	if override.Profile != "" {
		result.Profile = override.Profile
	}
	if override.InitialRole != nil {
		result.InitialRole = override.InitialRole
	}
	if override.Role != nil {
		result.Role = override.Role
	}

	return result
}