package awscwl

import (
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/op"
	"github.com/elastic/beats/libbeat/outputs"
)

func init() {
	outputs.RegisterOutputPlugin("awscwl", New)
}

type awsCloudWatchStream struct {
	nextToken *string
	svc       *cloudwatchlogs.CloudWatchLogs
}

type awscwlOutput struct {
	beat                common.BeatInfo
	codec               outputs.Codec
	logGroupName        string
	logStreamNamePrefix string
	session             *session.Session
	stream              *awsCloudWatchStream
}

// New instantiates a new awscwlOutput instance.
func New(beat common.BeatInfo, cfg *common.Config) (outputs.Outputer, error) {
	config := defaultConfig
	if err := cfg.Unpack(&config); err != nil {
		return nil, err
	}

	// bulk_max_size is the numner of events in a bulk message request.
	// default: 1024
	if !cfg.HasField("bulk_max_size") {
		cfg.SetInt("bulk_max_size", -1, defaultBulkSize)
	}

	// flush_interval is the number of seconds to wait in between bulk
	// messages.
	// default: 300 (seconds)
	if !cfg.HasField("flush_interval") {
		cfg.SetInt("flush_interval", -1, defaultFlushInterval)
	}

	output := &awscwlOutput{beat: beat, session: nil,
		stream: &awsCloudWatchStream{nextToken: nil, svc: nil},
	}
	if err := output.init(config); err != nil {
		return nil, err
	}
	return output, nil
}

func (out *awscwlOutput) init(config awscwlConfig) error {
	var err error

	// configure the codec passed in or create a default
	codec, err := outputs.CreateEncoder(config.Codec)
	if err != nil {
		return err
	}
	out.codec = codec

	// we need these variables ongoing to copy them to our output config
	out.logStreamNamePrefix = config.LogStreamNamePrefix
	out.logGroupName = config.LogGroupName

	// create an aws session based off the credentials in the config. If the
	// credentials are not specified then the aws SDK automagically checks
	// the environment. if you are running this on an aws instance you probably
	// wont need to specifiy credentials... maybe :)

	if config.AccessKeyId != "" && config.SecretAccessKey != "" {
		out.session, err = session.NewSession(&aws.Config{
			Region: aws.String(config.Region),
			Credentials: credentials.NewStaticCredentials(config.AccessKeyId,
				config.SecretAccessKey,
				config.SessionToken),
		})
	} else {
		out.session, err = session.NewSession(&aws.Config{
			Region: aws.String(config.Region),
		})
	}
	// return the err from either of the two above scenarios
	if err != nil {
		return err
	}

	// create a new instance of cloudwatchlogs
	out.stream.svc = cloudwatchlogs.New(out.session, aws.NewConfig())

	// see if the stream is a part of the configured LogGroup
	params := &cloudwatchlogs.DescribeLogStreamsInput{
		LogGroupName:        aws.String(out.logGroupName),
		Limit:               aws.Int64(1),
		LogStreamNamePrefix: aws.String(out.logStreamNamePrefix),
	}
	resp, err := out.stream.svc.DescribeLogStreams(params)
	if err != nil {
		return err
	}

	// if the stream exists we need to get the uploadSequenceToken. if the
	// stream does not exist we need to create it.
	if len(resp.LogStreams) == 0 {
		params := &cloudwatchlogs.CreateLogStreamInput{
			LogGroupName:  aws.String(out.logGroupName),
			LogStreamName: aws.String(out.logStreamNamePrefix),
		}
		if _, err := out.stream.svc.CreateLogStream(params); err != nil {
			return err
		}
	} else {
		if resp.LogStreams[0].UploadSequenceToken != nil {
			out.stream.nextToken = resp.LogStreams[0].UploadSequenceToken
		}
	}

	// at this point we have located a logGroup that we can write to and we
	// had access to create a stream. we will now wait for publish events
	// and then write them.

	return nil
}

// Implement Outputer
func (out *awscwlOutput) Close() error {
	return nil
}

// PublishEvent is called for single events if the BulkPublish service is
// disabled. this is a less than optimal function to call for rest based
// aws calls but we still need to handle single events
func (out *awscwlOutput) PublishEvent(
	sig op.Signaler,
	opts outputs.Options,
	data outputs.Data,
) error {
	dataAr := make([]outputs.Data, 0)
	dataAr = append(dataAr, data)
	return out.BulkPublish(sig, opts, dataAr)
}

// BulkPublish is called whenever the bulk_max_size queues the proper number of
// events or whenever flush_interval has reached its limit.
func (out *awscwlOutput) BulkPublish(
	sig op.Signaler,
	opts outputs.Options,
	data []outputs.Data,
) error {
	logEvents := make([]*cloudwatchlogs.InputLogEvent, 0) // empty events array
	var serializedEvent []byte
	var err error

	// get the current month
	t := time.Now().UnixNano() / int64(time.Millisecond)

	for _, v := range data {
		serializedEvent, err = out.codec.Encode(v.Event)
		if err != nil {
			op.SigFailed(sig, err)
			return err
		}
		logEvents = append(logEvents, &cloudwatchlogs.InputLogEvent{
			Message:   aws.String(string(serializedEvent)),
			Timestamp: aws.Int64(t),
		})
	}

	// push logEvents() to aws
	params := &cloudwatchlogs.PutLogEventsInput{
		LogEvents:     logEvents,
		LogGroupName:  aws.String(out.logGroupName),
		LogStreamName: aws.String(out.logStreamNamePrefix),
		SequenceToken: out.stream.nextToken,
	}
	resp, err := out.stream.svc.PutLogEvents(params)
	if err != nil {
		op.SigFailed(sig, err)
		return err
	}

	// the NextSequenceToken must be passed to future calls of PutLogEvents()
	if resp.NextSequenceToken != nil {
		out.stream.nextToken = resp.NextSequenceToken
	}

	// instruct the prospector that messages have been successfully processed
	op.SigCompleted(sig)

	return nil
}
