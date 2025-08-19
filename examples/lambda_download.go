package main

import (
	"github.com/aidansteele/iam-collect"
)

// Lambda function for downloading IAM data
func main() {
	// Start the Lambda runtime with direct handler
	// This handles events directly without API Gateway integration
	iamcollect.StartDirectDownloadLambda()
}

// Alternatively, you can use the API Gateway compatible handler:
// func main() {
//     iamcollect.StartDownloadLambda()
// }