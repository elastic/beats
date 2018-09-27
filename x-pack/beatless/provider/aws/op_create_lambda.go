package aws

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	lambdaApi "github.com/aws/aws-sdk-go-v2/service/lambda"

	"github.com/elastic/beats/libbeat/logp"
)

var handlerName = "beatless"

type opCreateLambda struct {
	svc *lambdaApi.Lambda
	log *logp.Logger
}

func (o *opCreateLambda) Execute(ctx *executerContext) error {
	// Setup the environment to known which function to execute.
	envVariables := map[string]string{
		"BEAT_STRICT_PERMS": "false",
		"ENABLED_FUNCTIONS": ctx.Name,
	}

	req := &lambdaApi.CreateFunctionInput{
		Code:         &lambdaApi.FunctionCode{ZipFile: ctx.Content},
		FunctionName: aws.String(ctx.Name),
		Handler:      aws.String(ctx.HandleName),
		Role:         aws.String(ctx.Role),
		Runtime:      ctx.Runtime,
		Description:  aws.String(ctx.Description),
		Publish:      aws.Bool(false), // TODO check that.
		Environment:  &lambdaApi.Environment{Variables: envVariables},
	}

	api := o.svc.CreateFunctionRequest(req)
	resp, err := api.Send()
	if err != nil {
		o.log.Debugf("could not create function, error: %s, response: %s", err, resp)
		return err
	}

	// retrieve the function arn for future calls.
	ctx.FunctionArn = *resp.FunctionArn

	return nil
}

func (o *opCreateLambda) Rollback(ctx *executerContext) error {
	req := &lambdaApi.DeleteFunctionInput{FunctionName: aws.String(ctx.Name)}

	api := o.svc.DeleteFunctionRequest(req)
	resp, err := api.Send()
	if err != nil {
		o.log.Debugf("could not remove function, error: %s, response: %s", err, resp)
		return err
	}
	return nil
}

func newOpCreateLambda(log *logp.Logger, awsCfg aws.Config) *opCreateLambda {
	if log == nil {
		log = logp.NewLogger("opCreateLambda")
	}
	return &opCreateLambda{log: log, svc: lambdaApi.New(awsCfg)}
}
