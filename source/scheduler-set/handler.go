package main

import (
	"context"
	"fmt"
	"log"
	"regexp"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/ec2iface"
	"github.com/caarlos0/env/v6"
)

type inputEvent struct {
	InstanceID    string `json:"instanceId"`
	RangeTime     string `json:"rangeTime"`
	RangeWeekdays string `json:"rangeWeekdays"`
}

type config struct {
	ScheduleTag    string `env:"SCHEDULE_TAG" envDefault:"Schedule"`
	ScheduleTagDay string `env:"SCHEDULE_TAG_DAY" envDefault:"ScheduleDay"`
}

const rangeTimeRegexp = `#?\d{2}:\d{2}-\d{2}:\d{2}`

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
		Key:   aws.String(conf.ScheduleTag),
		Value: aws.String(event.RangeTime),
	})
	if event.RangeWeekdays != "" {
		tags = append(tags, ec2.Tag{
			Key:   aws.String(conf.ScheduleTagDay),
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
