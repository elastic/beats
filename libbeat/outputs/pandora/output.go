package pandora

import (
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"qiniu.com/pandora/pipeline"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/op"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/outputs"
)

type pandoraOutput struct {
	beatName string
	hostName string
	repo     string
	retries  int
	batch    int
	pointBuf []pipeline.Point
	client   pipeline.PipelineAPI
}

func init() {
	outputs.RegisterOutputPlugin("pandora", New)
}

var (
	debugf = logp.MakeDebug("pandora")
)

// New instantiates a new output plugin instance publishing to pandora.
func New(beatName string, cfg *common.Config, topologyExpire int) (outputs.Outputer, error) {

	config := defaultConfig
	if err := cfg.Unpack(&config); err != nil {
		return nil, err
	}
	pconfig := pipeline.NewConfig().WithEndpoint(config.Endpoint).WithAccessKeySecretKey(config.AK, config.SK)
	pclient, err := pipeline.New(pconfig)
	if err != nil {
		logp.Err("create pandora client failed, err[%s]", err)
		return nil, err
	}

	hostname, err := os.Hostname()
	if err != nil {
		logp.Err("failed to get hostname, err[%s]", err)
		return nil, err
	}
	repo := fmt.Sprintf("dora_%s", config.Region)
	output := &pandoraOutput{
		beatName: beatName,
		hostName: hostname,
		repo:     repo,
		retries:  config.MaxRetries,
		batch:    config.Batch,
		pointBuf: make([]pipeline.Point, 0, config.Batch),
		client:   pclient,
	}
	return output, nil
}

func (p *pandoraOutput) sendPoints() (err error) {
	points := &pipeline.PostDataInput{
		RepoName: p.repo,
		Points:   pipeline.Points(p.pointBuf),
	}
	for i := 0; i < p.retries; i++ {
		if err = p.client.PostData(points); err != nil {
			logp.Err("post data failed at try %d, err[%s]", i, err)

		} else {
			logp.Info("published %d points", len(p.pointBuf))
			break
		}
	}
	p.pointBuf = p.pointBuf[:0]
	return
}

func (p *pandoraOutput) Close() error {
	if len(p.pointBuf) > 0 {
		return p.sendPoints()
	}
	return nil
}

func (p *pandoraOutput) PublishEvent(
	sig op.Signaler,
	opts outputs.Options,
	data outputs.Data,
) error {
	if len(p.pointBuf) >= p.batch {
		p.sendPoints()
	}
	p.pointBuf = append(p.pointBuf, convertToPoint(p.hostName, data.Event))
	op.Sig(sig, nil)
	return nil
}

func parseProxyURL(raw string) (*url.URL, error) {
	url, err := url.Parse(raw)
	if err == nil && strings.HasPrefix(url.Scheme, "http") {
		return url, err
	}

	// Proxy was bogus. Try prepending "http://" to it and
	// see if that parses correctly.
	return url.Parse("http://" + raw)
}
func escapeString(str string) string {
	newStr := strings.Replace(str, `\`, `\\`, -1)
	newStr = strings.Replace(newStr, "\t", `\\t`, -1)
	newStr = strings.Replace(newStr, "\r", `\\r`, -1)
	newStr = strings.Replace(newStr, "\n", `\\n`, -1)
	return newStr
}

func mapStrToSlice(hostname string, event common.MapStr) []pipeline.PointField {
	fields := []pipeline.PointField{}

	fields = append(fields, pipeline.PointField{Key: "hostname", Value: hostname})

	message := event["message"].(string)
	fields = append(fields, pipeline.PointField{Key: "message", Value: escapeString(message)})

	ts := event["@timestamp"].(common.Time)
	fields = append(fields, pipeline.PointField{Key: "timestamp", Value: time.Time(ts).Format(time.RFC3339)})

	logType := event["type"].(string)
	if logType == "stdout" ||
		logType == "stderr" ||
		logType == "sandbox" {
		path := event["source"].(string)
		parts := strings.Split(path, "/")
		var i int
		var p string
		for i, p = range parts {
			if p == "executors" {
				executorId := parts[i+1]
				executorIdParts := strings.Split(parts[i+1], ".")
				fields = append(fields, pipeline.PointField{Key: "instance_id", Value: executorId})
				fields = append(fields, pipeline.PointField{Key: "app", Value: executorIdParts[0]})
				fields = append(fields, pipeline.PointField{Key: "launch_id", Value: executorIdParts[1]})
			}
		}

		if logType == "sandbox" {
			logType = parts[i-1]
		}
		fields = append(fields, pipeline.PointField{Key: "source", Value: logType})

	}

	return fields
}

func convertToPoint(hostName string, event common.MapStr) pipeline.Point {
	fields := mapStrToSlice(hostName, event)
	return pipeline.Point{Fields: fields}
}
