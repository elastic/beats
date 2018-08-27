package aws

import (
	"github.com/aws/aws-sdk-go-v2/aws"

	"github.com/aws/aws-sdk-go-v2/service/lambda"

	"github.com/elastic/beats/libbeat/logp"
)

type opCreateFunction struct {
	svc     *lambda.Lambda
	content []byte
}

func createFunction(lambda *lambda.Lambda, content []byte) operation {
	return &opCreateFunction{svc: lambda, content: content}
}

func (c *opCreateFunction) Commit(log *logp.Logger, info *txInfo) error {
	req := &lambda.CreateFunctionInput{
		Code: &lambda.FunctionCode{
			ZipFile: c.content,
		},
		FunctionName: aws.String(info.FunctionName),
		Handler:      aws.String("main"),
		Role:         aws.String("arn:aws:iam::357250095780:role/serverlessbeat"),
		Runtime:      runtime,
		Description:  aws.String("something meaningful"),
		Publish:      aws.Bool(false), // function is not published.
	}

	api := c.svc.CreateFunctionRequest(req)
	resp, err := api.Send()
	if err != nil {
		log.Debugf("could not create function, error: %s, response:", err, resp)
		return err
	}

	info.FunctionArn = *resp.FunctionArn
	log.Debug("function successfully created")
	return nil
}

func (c *opCreateFunction) Rollback(log *logp.Logger, info *txInfo) error {
	return nil
}

type opAddPermission struct {
	req *lambda.AddPermissionInput
	svc *lambda.Lambda
}

func addPermission(lambda *lambda.Lambda, req *lambda.AddPermissionInput) operation {
	return &opAddPermission{svc: lambda, req: req}
}

func (a *opAddPermission) Commit(log *logp.Logger, info *txInfo) error {
	a.req.FunctionName = aws.String(info.FunctionName)

	api := a.svc.AddPermissionRequest(a.req)
	resp, err := api.Send()
	if err != nil {
		log.Debugf("could not add permission to function, error: %s, response:", err, resp)
		return err
	}
	log.Debug("added permissions to function successfully")
	return nil
}

func (a *opAddPermission) Rollback(log *logp.Logger, info *txInfo) error {
	return nil
}
