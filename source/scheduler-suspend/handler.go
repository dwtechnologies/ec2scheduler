package main

// Suspend
// event:
// { "instanceId": "i-00e92a5a9cb7eeb4d", "unsuspendDatetime": "20171117" }

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/caarlos0/env/v6"
)

type inputEvent struct {
	InstanceID        string `json:"instanceId"`
	UnsuspendDatetime string `json:"unsuspendDatetime"`
}

type lambdaConfig struct {
	ScheduleTag        string `env:"SCHEDULE_TAG" envDefault:"Schedule"`
	ScheduleTagSuspend string `env:"SCHEDULE_TAG_SUSPEND" envDefault:"ScheduleSuspendUntil"`
}

var scheduleTagSuspendLayouts = map[int]string{
	4:  "2006",
	6:  "200601",
	8:  "20060102",
	11: "20060102T15",
	14: "20060102T15:04",
}

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

	// parse suspend time
	if _, ok := scheduleTagSuspendLayouts[len(event.UnsuspendDatetime)]; !ok {
		log.Printf("[%s] layout doesn't match any supported one %s", event.InstanceID, event.UnsuspendDatetime)
		return fmt.Sprintf("unable to parse date: %s", event.UnsuspendDatetime), nil
	}
	_, err := time.Parse(scheduleTagSuspendLayouts[len(event.UnsuspendDatetime)], event.UnsuspendDatetime)
	if err != nil {
		log.Printf("[%s] can't parse date %s", event.InstanceID, event.UnsuspendDatetime)
		return fmt.Sprintf("unable to parse date: %s", event.UnsuspendDatetime), nil
	}

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return "", err
	}
	client := ec2.NewFromConfig(cfg)

	resp, err := client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
		InstanceIds: []string{event.InstanceID},
	})
	if err != nil {
		return "", err
	}

	if len(resp.Reservations) < 1 {
		log.Printf("[%s] no instance found", event.InstanceID)
		return fmt.Sprintf("no instance found with ID %s", event.InstanceID), nil
	}

	for _, tag := range resp.Reservations[0].Instances[0].Tags {
		if *tag.Key == conf.ScheduleTag {
			err = createTags(ctx, client, event.InstanceID, []types.Tag{
				{
					Key:   aws.String(conf.ScheduleTagSuspend),
					Value: aws.String(event.UnsuspendDatetime),
				},
				{
					Key:   aws.String(conf.ScheduleTag),
					Value: aws.String(fmt.Sprintf("#%s", *tag.Value)),
				},
			})
			if err != nil {
				return "", err
			}

			log.Printf("[%s] scheduler suspended until %s", event.InstanceID, event.UnsuspendDatetime)
			return fmt.Sprintf("instance %s scheduler suspended until %s", event.InstanceID, event.UnsuspendDatetime), nil
		}
	}

	log.Printf("[%s] unable to find %s tag", event.InstanceID, conf.ScheduleTag)
	return fmt.Sprintf("unable to find %s tag for instance %s", conf.ScheduleTag, event.InstanceID), nil
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
