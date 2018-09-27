package aws

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	lambdaApi "github.com/aws/aws-sdk-go-v2/service/lambda"

	"github.com/elastic/beats/libbeat/logp"
)

type opUpdateAlias struct {
	svc *lambdaApi.Lambda
	log *logp.Logger
}

func (o *opUpdateAlias) Execute(ctx *executerContext) error {
	o.log.Debugf("updating new alias for function with name: %s", ctx.Name)
	req := &lambdaApi.UpdateAliasInput{
		Description:     aws.String("alias for " + ctx.Name),
		FunctionVersion: aws.String("$LATEST"),
		FunctionName:    aws.String(ctx.Name),
		Name:            aws.String(aliasSuffix),
	}

	api := o.svc.UpdateAliasRequest(req)
	resp, err := api.Send()
	if err != nil {
		o.log.Debugf("could not update alias, error: %s, response: %s", err, resp)
		return err
	}

	ctx.AliasArn = *resp.AliasArn
	o.log.Debug("alias created successfully")
	return nil
}

func newOpUpdateAlias(log *logp.Logger, awsCfg aws.Config) *opUpdateAlias {
	if log == nil {
		log = logp.NewLogger("opUpdateLambda")
	}

	return &opUpdateAlias{log: log, svc: lambdaApi.New(awsCfg)}
}
