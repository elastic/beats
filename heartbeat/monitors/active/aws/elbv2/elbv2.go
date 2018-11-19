package elbv2

import (
	"fmt"
	"net/url"

	"github.com/elastic/beats/libbeat/beat"

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

func init() {
	monitors.RegisterActive("aws_elbv2", create)
}

func create(name string, commonCfg *common.Config) (jobs []monitors.Job, endpoints int, err error) {
	config := &Config{}
	err = commonCfg.Unpack(config)
	if err != nil {
		return nil, 0, err
	}

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

	return monitors.WrapAll(jobs, monitors.WithErrAsField), len(pageARNs), nil
}

func newELBv2Job(client *elbv2.ELBV2, arns []string) (monitors.Job, error) {
	job := monitors.CreateNamedJob(
		fmt.Sprintf("aws_elbv2/%v", arns),
		func() (*beat.Event, []monitors.Job, error) {
			describeInput := &elbv2.DescribeLoadBalancersInput{
				LoadBalancerArns: arns,
			}

			lbResp, err := client.DescribeLoadBalancersRequest(describeInput).Send()

			if err != nil {
				return nil, nil, err
			}

			var jobs []monitors.Job
			for _, lb := range lbResp.LoadBalancers {
				jobs = append(jobs, monitors.WithJobId(*lb.LoadBalancerArn, newLbJob(lb)))
				jobs = append(jobs, monitors.WithJobId(*lb.LoadBalancerArn, newListenerJob(client, lb)))
			}

			return nil, jobs, nil
		})

	return job, nil
}

func newLbJob(lb elbv2.LoadBalancer) monitors.Job {
	return monitors.TimeAndCheckJob(monitors.AnonJob(func() (*beat.Event, []monitors.Job, error) {
		var status string
		if lb.State.Code == elbv2.LoadBalancerStateEnumActive {
			status = "up"
		} else {
			status = "down"
		}

		event := &beat.Event{
			Fields: common.MapStr{
				"monitor": common.MapStr{
					"status":    status,
					"task_type": "elbv2_state",
					"task_id":   lb.DNSName,
				},
				"aws": common.MapStr{
					"arn": lb.LoadBalancerArn,
					"elbv2": common.MapStr{
						"status": lb.State,
						"arn":    lb.LoadBalancerArn,
					},
				},
			},
		}

		return event, []monitors.Job{}, nil
	}))
}

func newListenerJob(client *elbv2.ELBV2, lb elbv2.LoadBalancer) monitors.Job {
	return monitors.TimeAndCheckJob(monitors.AnonJob(func() (*beat.Event, []monitors.Job, error) {
		describeInput := &elbv2.DescribeListenersInput{LoadBalancerArn: lb.LoadBalancerArn}
		// Pagination not supported when LB is specified
		resp, err := client.DescribeListenersRequest(describeInput).Send()
		if err != nil {
			return nil, nil, err
		}

		var jobs []monitors.Job
		for _, listener := range resp.Listeners {
			hostPort := fmt.Sprintf("%s:%d", *lb.DNSName, *listener.Port)

			var job monitors.Job
			var err error

			var listenerURL url.URL
			switch listener.Protocol {
			case elbv2.ProtocolEnumHttps:
				listenerURL = url.URL{Scheme: "https", Host: hostPort}
				job, err = newHttpCheck(listener, &listenerURL)
			case elbv2.ProtocolEnumHttp:
				listenerURL = url.URL{Scheme: "http", Host: hostPort}
				job, err = newHttpCheck(listener, &listenerURL)
			case elbv2.ProtocolEnumTcp:
				panic("IMPLEMENT ME")
			}

			if err != nil {
				return nil, nil, err
			}

			jobs = append(jobs, job)
		}

		return nil, jobs, nil
	}))
}

func newHttpCheck(listener elbv2.Listener, url *url.URL) (monitors.Job, error) {
	httpConfig := &http.Config{
		URLs: []string{url.String()},
		Mode: monitors.DefaultIPSettings,
	}

	job, err := http.NewHTTPMonitorIPsJob(httpConfig, url.String(), nil, nil, nil, nil)
	if err != nil {
		return nil, err
	}

	overlay := common.MapStr{
		"monitor": common.MapStr{
			"task_type": "elbv2_check_port_http",
			"task_id":   url.String(),
		},
		"aws": common.MapStr{
			"arn": listener.ListenerArn,
			"elbv2": common.MapStr{
				"arn": listener.LoadBalancerArn,
				"listener": common.MapStr{
					"arn": listener.ListenerArn,
				},
			},
		},
	}

	runner := monitors.AnonJob(func() (*beat.Event, []monitors.Job, error) {
		event, jobs, err := job.Run()

		if event != nil {
			monitors.MergeEventFields(event, overlay)
		}

		return event, jobs, err
	})
	return runner, nil
}
