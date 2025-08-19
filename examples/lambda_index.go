package main

import (
	"github.com/aidansteele/iam-collect"
)

// Lambda function for creating indexes
func main() {
	// Start the Lambda runtime with direct handler
	// This handles events directly without API Gateway integration
	iamcollect.StartDirectIndexLambda()
}

// Alternatively, you can use the API Gateway compatible handler:
// func main() {
//     iamcollect.StartIndexLambda()
// }