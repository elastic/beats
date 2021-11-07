
# Sample Function

The following is a sample class and Lambda function that receives AWS AppSync event message data as input, writes some of the message data to CloudWatch Logs, and responds with a 200 status and the same body as the request. (Note that by default anything written to Console will be logged as CloudWatch Logs events.)

```go
package main

import (
	"context"
	"fmt"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

func handler(ctx context.Context, event events.AppSyncResolverTemplate) error {

  fmt.Printf("Version: %s\n", event.Version)
  fmt.Printf("Operation: %s\n", event.Operation)
  fmt.Printf("Payload: %s\n", string(event.Payload))

	return nil
}

func main() {
	lambda.Start(handler)
}

```
