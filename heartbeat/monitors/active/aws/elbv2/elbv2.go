package elbv2

import (
	"fmt"
	"net/url"
	"strconv"

	"github.com/pkg/errors"

	"github.com/elastic/beats/heartbeat/monitors/active/http"

	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/elbv2"

	"github.com/elastic/beats/heartbeat/monitors"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

// The maximum number of describable ELBs in one call.
// See the source for elbv2.DescribeLoadBalancersInput for more info.
const pageSize = 20

func create(name string, commonCfg *common.Config) (jobs []monitors.Job, endpoints int, err error) {
	config := Config{}
	commonCfg.Unpack(config)

	cfg, err := external.LoadDefaultAWSConfig()
	cfg.Region = config.Region
	if err != nil {
		logp.Err("error loading AWS config for aws_elb autodiscover provider: %s", err)
	}

	client := elbv2.New(cfg)

	// The AWS API paginates, so let's chunk the ARNs
	var pageARNs []string
	for idx, arn := range config.ARNs {
		pageARNs = append(pageARNs, arn)

		atMaxPageSize := len(pageARNs) == pageSize
		atLastARN := idx+1 == len(config.ARNs)
		if atMaxPageSize || atLastARN {
			job, err := newELBv2Job(client, pageARNs)
			if err != nil {
				return nil, 0, err
			}
			jobs = append(jobs, job)
		}
	}

	return jobs, len(pageARNs), nil
}

func newELBv2Job(client *elbv2.ELBV2, arns []string) (monitors.Job, error) {
	psValue := int64(pageSize)
	describeInput := &elbv2.DescribeLoadBalancersInput{
		PageSize:         &psValue,
		LoadBalancerArns: arns,
	}

	return monitors.MakeJob(monitors.JobSettings{}, func() (common.MapStr, []monitors.TaskRunner, error) {
		lbResp, err := client.DescribeLoadBalancersRequest(describeInput).Send()

		if err != nil {
			return nil, nil, err
		}

		var listenersCont []monitors.TaskRunner
		for _, lb := range lbResp.LoadBalancers {
			listenersCont = append(listenersCont, newListenerTask(client, lb))
		}
	}), nil
}

func newListenerTask(client *elbv2.ELBV2, lb elbv2.LoadBalancer) monitors.TaskRunner {
	return monitors.MakeCont(func() (common.MapStr, []monitors.TaskRunner, error) {
		describeInput := &elbv2.DescribeListenersInput{LoadBalancerArn: lb.LoadBalancerArn}
		req := client.DescribeListenersRequest(describeInput).Paginate()

		var tasks []monitors.TaskRunner
		for req.Next() {
			for _, listener := range req.CurrentPage().Listeners {
				hostPort := fmt.Sprintf("%s:%s", *lb.DNSName, listener.Port)

				var task monitors.TaskRunner
				var err error

				switch listener.Protocol {
				case elbv2.ProtocolEnumHttps:
					url := &url.URL{Scheme: "https", Host: hostPort}
					task, err = newHttpCheck(url)
				case elbv2.ProtocolEnumHttp:
					url := &url.URL{Scheme: "http", Host: hostPort}
					task, err = newHttpCheck(url)
				case elbv2.ProtocolEnumTcp:
					panic("IMPLEMENT ME")
				}

				if err != nil {
					return nil, nil, err
				}

				tasks = append(tasks, task)
			}
		}

	})
}

func newHttpCheck(url *url.URL) (monitors.TaskRunner, error) {
	httpConfig := &http.Config{
		URLs: []string{url.String()},
	}

	for _, url := range httpConfig.URLs {
		job, err := http.NewHTTPMonitorIPsJob(httpConfig, url, nil, nil, nil, nil)
		if err != nil {
			return nil, errors.Wrap(err, "could not initialize HTTP job for ELB")
		}
		job.
		return job, nil
	}

	req, err := http.BuildRequest(url.String(), httpConfig, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "could not build request for ELB listener port @ '%s'", url)
	}
	port, err := strconv.Atoi(url.Port())
	if err != nil {
		return nil, errors.Wrapf(err, "could not convert ELB listener port '%s' to int", url.Port())
	}

	task := http.CreatePingFactory(
		httpConfig,
		url.Hostname(),
		uint16(port),
		nil,
		req,
		nil,
		nil,
	)

	return task(), nil
}
