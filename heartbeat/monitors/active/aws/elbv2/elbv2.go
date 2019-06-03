package elbv2

import (
	"context"
	"fmt"
	"net/url"

	"github.com/elastic/beats/heartbeat/eventext"

	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"

	"github.com/aws/aws-sdk-go-v2/aws/external"

	"github.com/elastic/beats/heartbeat/monitors"
	"github.com/elastic/beats/heartbeat/monitors/active/http"
	"github.com/elastic/beats/heartbeat/monitors/jobs"
	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

// The maximum number of describable ELBs in one call.
// See the source for elbv2.DescribeLoadBalancersInput for more info.
const pageSize = 20

func init() {
	monitors.RegisterActive("aws_elbv2", create)
}

func create(name string, commonCfg *common.Config) (jobs []jobs.Job, endpoints int, err error) {
	config := &Config{}
	err = commonCfg.Unpack(config)
	if err != nil {
		return nil, 0, err
	}

	cfg, _ := external.LoadDefaultAWSConfig()
	cfg.Region = config.Region
	if err != nil {
		logp.Err("error loading AWS config for aws_elb autodiscover provider: %s", err)
	}

	client := elasticloadbalancingv2.New(cfg)

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

func newELBv2Job(client *elasticloadbalancingv2.Client, arns []string) (jobs.Job, error) {
	job := func(event *beat.Event) ([]jobs.Job, error) {
		describeInput := &elasticloadbalancingv2.DescribeLoadBalancersInput{
			LoadBalancerArns: arns,
		}

		lbResp, err := client.DescribeLoadBalancersRequest(describeInput).Send(context.TODO())

		if err != nil {
			return nil, err
		}

		var jobs []jobs.Job
		for _, lb := range lbResp.LoadBalancers {
			jobs = append(jobs, newLbJob(lb))
			jobs = append(jobs, newListenerJob(client, lb))
		}

		return jobs, nil
	}

	return job, nil
}

func newLbJob(lb elasticloadbalancingv2.LoadBalancer) jobs.Job {
	return func(event *beat.Event) ([]jobs.Job, error) {
		var status string
		if lb.State.Code == elasticloadbalancingv2.LoadBalancerStateEnumActive {
			status = "up"
		} else {
			status = "down"
		}

		eventext.MergeEventFields(event, common.MapStr{
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
		})

		return []jobs.Job{}, nil
	}
}

func newListenerJob(client *elasticloadbalancingv2.Client, lb elasticloadbalancingv2.LoadBalancer) jobs.Job {
	return func(event *beat.Event) ([]jobs.Job, error) {
		describeInput := &elasticloadbalancingv2.DescribeListenersInput{LoadBalancerArn: lb.LoadBalancerArn}
		// Pagination not supported when LB is specified
		resp, err := client.DescribeListenersRequest(describeInput).Send(context.TODO())
		if err != nil {
			return nil, err
		}

		var conts []jobs.Job
		for _, listener := range resp.Listeners {
			hostPort := fmt.Sprintf("%s:%d", *lb.DNSName, *listener.Port)

			var job jobs.Job
			var err error

			var listenerURL url.URL
			switch listener.Protocol {
			case elasticloadbalancingv2.ProtocolEnumHttps:
				listenerURL = url.URL{Scheme: "https", Host: hostPort}
				job, err = newHttpCheck(listener, &listenerURL)
			case elasticloadbalancingv2.ProtocolEnumHttp:
				listenerURL = url.URL{Scheme: "http", Host: hostPort}
				job, err = newHttpCheck(listener, &listenerURL)
			case elasticloadbalancingv2.ProtocolEnumTcp:
				panic("IMPLEMENT ME")
			}

			if err != nil {
				return nil, err
			}

			conts = append(conts, job)
		}

		return conts, nil
	}
}

func newHttpCheck(listener elasticloadbalancingv2.Listener, url *url.URL) (jobs.Job, error) {
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

	runner := func(event *beat.Event) ([]jobs.Job, error) {
		jobs, err := job(event)

		if event != nil {
			eventext.MergeEventFields(event, overlay)
		}

		return jobs, err
	}

	return runner, nil
}
