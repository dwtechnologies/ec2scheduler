package main

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"log"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/caarlos0/env/v6"
)

type inputEvent struct {
	Format string `json:"format"`
	Filter string `json:"filter"`
}
type instanceData struct {
	InstanceID      string
	InstanceName    string
	State           string
	Schedule        string
	ScheduleDay     string
	ScheduleSuspend string
	ScheduleSNS     string
}

type config struct {
	ScheduleTag        string `env:"SCHEDULE_TAG" envDefault:"Schedule"`
	ScheduleTagDay     string `env:"SCHEDULE_TAG_DAY" envDefault:"ScheduleDay"`
	ScheduleTagSuspend string `env:"SCHEDULE_TAG_SUSPEND" envDefault:"ScheduleSuspendUntil"`
	ScheduleTagSNS     string `env:"SCHEDULE_TAG_SNS" envDefault:"ScheduleSNS"`
}

var teamsOutputTmpl = `{{ range . -}}
▸ **{{ .InstanceID }}** {{ if ne .InstanceName "" }}[{{ .InstanceName }}]{{ end }}
State: {{ .State }}
Schedule: {{ .Schedule }}
{{ if ne .ScheduleDay "" -}}
ScheduleDay: {{ .ScheduleDay }}
{{ end -}}
{{ if ne .ScheduleSuspend "" -}}
ScheduleSuspend: {{ .ScheduleSuspend }}
{{ end -}}
{{ if ne .ScheduleSNS "" -}}
ScheduleSNS: {{ .ScheduleSNS }}
{{ end }}
{{ end }}`

func main() {
	lambda.Start(handler)
}

func handler(ctx context.Context, event inputEvent) (string, error) {
	// parse env variables
	conf := &config{}
	if err := env.Parse(conf); err != nil {
		log.Printf("%s", err)
		return "", err
	}

	cfg, err := external.LoadDefaultAWSConfig()
	if err != nil {
		return "", err
	}
	client := ec2.New(cfg)

	resp, err := client.DescribeInstancesRequest(&ec2.DescribeInstancesInput{
		Filters: []ec2.Filter{
			{
				Name:   aws.String("instance-state-name"),
				Values: []string{"running", "stopped"},
			},
			{
				Name:   aws.String("tag-key"),
				Values: []string{conf.ScheduleTag},
			},
			{
				Name:   aws.String("tag:Name"),
				Values: []string{fmt.Sprintf("*%s*", event.Filter)},
			},
		},
	}).Send(ctx)

	if len(resp.Reservations) < 1 {
		log.Printf("no scheduled instances")
		return "", nil
	}

	instancesData := []instanceData{}
	for _, reservation := range resp.Reservations {
		instance := reservation.Instances[0]

		d := &instanceData{}
		d.InstanceID = *instance.InstanceId
		d.State = fmt.Sprintf("%s", instance.State.Name)

		for _, tag := range instance.Tags {
			if *tag.Key == "Name" {
				d.InstanceName = *tag.Value
			}

			if *tag.Key == conf.ScheduleTag {
				d.Schedule = *tag.Value
			}

			if *tag.Key == conf.ScheduleTagDay {
				d.ScheduleDay = *tag.Value
			}

			if *tag.Key == conf.ScheduleTagSuspend {
				d.ScheduleSuspend = *tag.Value
			}

			if *tag.Key == conf.ScheduleTagSNS {
				d.ScheduleSNS = *tag.Value
			}
		}

		instancesData = append(instancesData, *d)
	}

	log.Printf("%+v", instancesData)

	switch event.Format {
	case "teams":
		return teamsResponse(instancesData)
	}

	// event.Format: text
	return fmt.Sprintf("%+v", instancesData), nil
}

// parse Teams response
func teamsResponse(response []instanceData) (string, error) {
	t, _ := template.New("output").Parse(teamsOutputTmpl)
	var pp bytes.Buffer
	if err := t.Execute(&pp, response); err != nil {
		log.Printf("template - execute error: %s", err)

		return "error getting dw-info", nil
	}

	return pp.String(), nil
}
