package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"regexp"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/ec2iface"
)

type inputEvent struct {
	InstanceID    string `json:"instanceId"`
	RangeTime     string `json:"rangeTime"`
	RangeWeekdays string `json:"rangeWeekdays"`
}

var scheduleTag = os.Getenv("SCHEDULE_TAG")
var scheduleTagDay = os.Getenv("SCHEDULE_TAG_DAY")
var scheduleTagSuspend = os.Getenv("SCHEDULE_TAG_SUSPEND")

const rangeTimeRegexp = `#?\d{2}:\d{2}-\d{2}:\d{2}`

func main() {
	lambda.Start(handler)
}

func handler(ctx context.Context, event inputEvent) (string, error) {
	// CN regions don't support env variables
	if scheduleTag == "" {
		scheduleTag = "Schedule"
	}
	if scheduleTagDay == "" {
		scheduleTagDay = "ScheduleDay"
	}

	matched, err := regexp.Match(rangeTimeRegexp, []byte(event.RangeTime))
	if err != nil {
		return "", err
	}

	if !matched {
		log.Printf("invalid time range: %s", event.RangeTime)
		return fmt.Sprintf("invalid time range: %s", event.RangeTime), nil
	}

	// tags
	tags := []ec2.Tag{}
	tags = append(tags, ec2.Tag{
		Key:   aws.String(scheduleTag),
		Value: aws.String(event.RangeTime),
	})
	if event.RangeWeekdays != "" {
		tags = append(tags, ec2.Tag{
			Key:   aws.String(scheduleTagDay),
			Value: aws.String(event.RangeWeekdays),
		})
	}

	cfg, err := external.LoadDefaultAWSConfig()
	if err != nil {
		return "", err
	}
	client := ec2.New(cfg)

	// set tags
	err = createTags(ctx, client, event.InstanceID, tags)
	if err != nil {
		return "", err
	}

	log.Printf("scheduler set for instance %s. rangeTime: %s, rangeWeekdays: %s", event.InstanceID, event.RangeTime, event.RangeWeekdays)
	return fmt.Sprintf("scheduler set for instance %s: %s", event.InstanceID, event.RangeTime), nil
}

func createTags(ctx context.Context, client ec2iface.ClientAPI, instanceID string, tags []ec2.Tag) error {
	_, err := client.CreateTagsRequest(&ec2.CreateTagsInput{
		Resources: []string{instanceID},
		Tags:      tags,
	}).Send(ctx)
	if err != nil {
		return err
	}

	return nil
}
