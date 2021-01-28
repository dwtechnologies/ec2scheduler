package main

import (
	"context"
	"fmt"
	"log"
	"regexp"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/caarlos0/env/v6"
)

type inputEvent struct {
	InstanceID    string `json:"instanceId"`
	RangeTime     string `json:"rangeTime"`
	RangeWeekdays string `json:"rangeWeekdays"`
}

type lambdaConfig struct {
	ScheduleTag    string `env:"SCHEDULE_TAG" envDefault:"Schedule"`
	ScheduleTagDay string `env:"SCHEDULE_TAG_DAY" envDefault:"ScheduleDay"`
}

const rangeTimeRegexp = `#?\d{2}:\d{2}-\d{2}:\d{2}`

func main() {
	lambda.Start(handler)
}

func handler(ctx context.Context, event inputEvent) (string, error) {
	// parse env variables
	conf := &lambdaConfig{}
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
	tags := []types.Tag{}
	tags = append(tags, types.Tag{
		Key:   aws.String(conf.ScheduleTag),
		Value: aws.String(event.RangeTime),
	})
	if event.RangeWeekdays != "" {
		tags = append(tags, types.Tag{
			Key:   aws.String(conf.ScheduleTagDay),
			Value: aws.String(event.RangeWeekdays),
		})
	}

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return "", err
	}
	client := ec2.NewFromConfig(cfg)

	// set tags
	err = createTags(ctx, client, event.InstanceID, tags)
	if err != nil {
		return "", err
	}

	log.Printf("scheduler set for instance %s. rangeTime: %s, rangeWeekdays: %s", event.InstanceID, event.RangeTime, event.RangeWeekdays)
	return fmt.Sprintf("scheduler set for instance %s: %s", event.InstanceID, event.RangeTime), nil
}

func createTags(ctx context.Context, client *ec2.Client, instanceID string, tags []types.Tag) error {
	_, err := client.CreateTags(ctx, &ec2.CreateTagsInput{
		Resources: []string{instanceID},
		Tags:      tags,
	})
	if err != nil {
		return err
	}

	return nil
}
