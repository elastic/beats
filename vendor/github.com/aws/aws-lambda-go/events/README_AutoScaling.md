# Sample Function

The following is a sample class and Lambda function that receives Amazon S3 event record data as an input and writes some of the record data to CloudWatch Logs. (Note that by default anything written to Console will be logged as CloudWatch Logs events.)

```go

import (
    "strings"
    "github.com/aws/aws-lambda-go/events")

func handler(ctx context.Context, autoScalingEvent events.AutoScalingEvent) {
        fmt.Printf("Instance-Id available in event is %s \n",autoScalingEvent.Detail["EC2InstanceId"]) 
    }
}

```
