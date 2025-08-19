package iamcollect

import (
	"context"
	"encoding/json"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

// LambdaDownloadRequest represents the request payload for the download Lambda function
type LambdaDownloadRequest struct {
	Config      *Config  `json:"config"`
	AccountIDs  []string `json:"accountIds,omitempty"`
	Regions     []string `json:"regions,omitempty"`
	Services    []string `json:"services,omitempty"`
	Concurrency int      `json:"concurrency,omitempty"`
	SkipIndex   bool     `json:"skipIndex,omitempty"`
}

// LambdaIndexRequest represents the request payload for the index Lambda function
type LambdaIndexRequest struct {
	Config      *Config  `json:"config"`
	Partition   string   `json:"partition,omitempty"`
	AccountIDs  []string `json:"accountIds,omitempty"`
	Regions     []string `json:"regions,omitempty"`
	Services    []string `json:"services,omitempty"`
	Concurrency int      `json:"concurrency,omitempty"`
}

// LambdaResponse represents the response from Lambda functions
type LambdaResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
}

// DownloadLambdaHandler handles Lambda requests for downloading IAM data
func DownloadLambdaHandler(ctx context.Context, event events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	var request LambdaDownloadRequest
	if err := json.Unmarshal([]byte(event.Body), &request); err != nil {
		return createErrorResponse(400, "Invalid request body: "+err.Error()), nil
	}

	// Validate request
	if request.Config == nil {
		return createErrorResponse(400, "Configuration is required"), nil
	}

	options := &DownloadOptions{
		Configs:     []*Config{request.Config},
		AccountIDs:  request.AccountIDs,
		Regions:     request.Regions,
		Services:    request.Services,
		Concurrency: request.Concurrency,
		SkipIndex:   request.SkipIndex,
	}

	// Execute download
	if err := DownloadData(ctx, options); err != nil {
		return createErrorResponse(500, "Download failed: "+err.Error()), nil
	}

	response := LambdaResponse{
		Success: true,
		Message: "IAM data download completed successfully",
	}

	body, _ := json.Marshal(response)
	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Body: string(body),
	}, nil
}

// IndexLambdaHandler handles Lambda requests for indexing IAM data
func IndexLambdaHandler(ctx context.Context, event events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	var request LambdaIndexRequest
	if err := json.Unmarshal([]byte(event.Body), &request); err != nil {
		return createErrorResponse(400, "Invalid request body: "+err.Error()), nil
	}

	// Validate request
	if request.Config == nil {
		return createErrorResponse(400, "Configuration is required"), nil
	}

	options := &IndexOptions{
		Configs:     []*Config{request.Config},
		Partition:   request.Partition,
		AccountIDs:  request.AccountIDs,
		Regions:     request.Regions,
		Services:    request.Services,
		Concurrency: request.Concurrency,
	}

	// Execute indexing
	if err := Index(ctx, options); err != nil {
		return createErrorResponse(500, "Indexing failed: "+err.Error()), nil
	}

	response := LambdaResponse{
		Success: true,
		Message: "IAM data indexing completed successfully",
	}

	body, _ := json.Marshal(response)
	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Body: string(body),
	}, nil
}

// createErrorResponse creates an error response
func createErrorResponse(statusCode int, message string) events.APIGatewayProxyResponse {
	response := LambdaResponse{
		Success: false,
		Error:   message,
	}

	body, _ := json.Marshal(response)
	return events.APIGatewayProxyResponse{
		StatusCode: statusCode,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Body: string(body),
	}
}

// StartDownloadLambda starts the Lambda runtime for the download function
func StartDownloadLambda() {
	lambda.Start(DownloadLambdaHandler)
}

// StartIndexLambda starts the Lambda runtime for the index function
func StartIndexLambda() {
	lambda.Start(IndexLambdaHandler)
}

// DirectDownloadHandler provides a direct Lambda handler for simple event-based invocation
func DirectDownloadHandler(ctx context.Context, request LambdaDownloadRequest) (LambdaResponse, error) {
	if request.Config == nil {
		return LambdaResponse{
			Success: false,
			Error:   "Configuration is required",
		}, nil
	}

	options := &DownloadOptions{
		Configs:     []*Config{request.Config},
		AccountIDs:  request.AccountIDs,
		Regions:     request.Regions,
		Services:    request.Services,
		Concurrency: request.Concurrency,
		SkipIndex:   request.SkipIndex,
	}

	if err := DownloadData(ctx, options); err != nil {
		return LambdaResponse{
			Success: false,
			Error:   "Download failed: " + err.Error(),
		}, nil
	}

	return LambdaResponse{
		Success: true,
		Message: "IAM data download completed successfully",
	}, nil
}

// DirectIndexHandler provides a direct Lambda handler for simple event-based invocation
func DirectIndexHandler(ctx context.Context, request LambdaIndexRequest) (LambdaResponse, error) {
	if request.Config == nil {
		return LambdaResponse{
			Success: false,
			Error:   "Configuration is required",
		}, nil
	}

	options := &IndexOptions{
		Configs:     []*Config{request.Config},
		Partition:   request.Partition,
		AccountIDs:  request.AccountIDs,
		Regions:     request.Regions,
		Services:    request.Services,
		Concurrency: request.Concurrency,
	}

	if err := Index(ctx, options); err != nil {
		return LambdaResponse{
			Success: false,
			Error:   "Indexing failed: " + err.Error(),
		}, nil
	}

	return LambdaResponse{
		Success: true,
		Message: "IAM data indexing completed successfully",
	}, nil
}

// StartDirectDownloadLambda starts the Lambda runtime for direct download invocation
func StartDirectDownloadLambda() {
	lambda.Start(DirectDownloadHandler)
}

// StartDirectIndexLambda starts the Lambda runtime for direct index invocation
func StartDirectIndexLambda() {
	lambda.Start(DirectIndexHandler)
}