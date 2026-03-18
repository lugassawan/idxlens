// Package main provides an AWS Lambda handler for IDXLens.
//
// This is scaffolding for a future Lambda deployment. The handler accepts
// a base64-encoded PDF in the request, runs extraction via the IDXLens
// pipeline, and returns structured JSON.
//
// To build and deploy, add github.com/aws/aws-lambda-go to go.mod and
// uncomment the implementation below.
//
//nolint:all
package main

// Example request/response types for the Lambda handler.
//
//	type Request struct {
//		PDF    string `json:"pdf"`    // base64-encoded PDF content
//		Format string `json:"format"` // output format: "json" (default) or "csv"
//	}
//
//	type Response struct {
//		StatusCode int    `json:"statusCode"`
//		Body       string `json:"body"`
//	}
//
// Example handler implementation:
//
//	func handler(ctx context.Context, req Request) (*Response, error) {
//		pdfBytes, err := base64.StdEncoding.DecodeString(req.PDF)
//		if err != nil {
//			return &Response{StatusCode: 400, Body: `{"error":"invalid base64"}`}, nil
//		}
//
//		// Run IDXLens extraction pipeline on pdfBytes.
//		// result, err := extract(pdfBytes)
//		// ...
//
//		return &Response{StatusCode: 200, Body: resultJSON}, nil
//	}
//
//	func main() {
//		lambda.Start(handler)
//	}

func main() {
	// Placeholder — see comments above for the intended implementation.
	// Requires github.com/aws/aws-lambda-go/lambda to be added to go.mod.
}
