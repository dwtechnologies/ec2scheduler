package main

import (
	"bytes"
	"fmt"
	"html/template"
	"log"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
)

type instanceData struct {
	InstanceID      string
	InstanceName    string
	State           string
	Schedule        string
	ScheduleDay     string
	ScheduleSuspend string
	ScheduleSNS     string
}

var scheduleTag = os.Getenv("SCHEDULE_TAG")
var scheduleTagDay = os.Getenv("SCHEDULE_TAG_DAY")
var scheduleTagSuspend = os.Getenv("SCHEDULE_TAG_SUSPEND")
var scheduleTagSNS = os.Getenv("SCHEDULE_TAG_SNS")
var teamsOutputTmpl = `{{ range . -}}
○ EC2 instance [{{ .InstanceID }} - {{ .InstanceName }}]
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
{{ end -}}
{{ end }}`

func main() {
	lambda.Start(handler)
}

func handler() (string, error) {
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
				Values: []string{scheduleTag},
			},
		},
	}).Send()

	if len(resp.Reservations) < 1 {
		log.Printf("no scheduled instances")
		return "", nil
	}

	instancesData := []instanceData{}
	for _, instance := range resp.Reservations[0].Instances {
		d := &instanceData{}
		d.InstanceID = *instance.InstanceId
		d.State = fmt.Sprintf("%s", instance.State.Name)

		for _, tag := range instance.Tags {
			if *tag.Key == "Name" {
				d.InstanceName = *tag.Value
			}

			if *tag.Key == scheduleTag {
				d.Schedule = *tag.Value
			}

			if *tag.Key == scheduleTagDay {
				d.ScheduleDay = *tag.Value
			}

			if *tag.Key == scheduleTagSuspend {
				d.ScheduleSuspend = *tag.Value
			}

			if *tag.Key == scheduleTagSNS {
				d.ScheduleSNS = *tag.Value
			}
		}

		instancesData = append(instancesData, *d)
	}

	log.Printf("%+v", instancesData)

	if os.Getenv("TEAMS_INTEGRATION") != "" {
		return parseTeamsResponse(instancesData)
	}

	return fmt.Sprintf("%v", instancesData), nil
}

// parse template response
func parseTeamsResponse(response []instanceData) (string, error) {
	t, _ := template.New("output").Parse(teamsOutputTmpl)
	var pp bytes.Buffer
	if err := t.Execute(&pp, response); err != nil {
		log.Printf("template - execute error: %s", err)

		return "error getting dw-info", nil
	}

	return pp.String(), nil
}
