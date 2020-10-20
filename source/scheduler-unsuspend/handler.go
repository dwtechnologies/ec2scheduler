package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/caarlos0/env/v6"
)

type inputEvent struct {
	InstanceID string `json:"instanceId"`
}
type lambdaConfig struct {
	ScheduleTag        string `env:"SCHEDULE_TAG" envDefault:"Schedule"`
	ScheduleTagSuspend string `env:"SCHEDULE_TAG_SUSPEND" envDefault:"ScheduleSuspendUntil"`
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

	cfg, err := config.LoadDefaultConfig()
	if err != nil {
		return "", err
	}
	client := ec2.NewFromConfig(cfg)

	resp, err := client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
		InstanceIds: []*string{aws.String(event.InstanceID)},
	})
	if err != nil {
		return "", err
	}

	if len(resp.Reservations) < 1 {
		log.Printf("no instance found %s", event.InstanceID)
		return "", nil
	}

	for _, tag := range resp.Reservations[0].Instances[0].Tags {
		// remove suspend tag (scheduleTagSuspend)
		if *tag.Key == conf.ScheduleTagSuspend {
			err := deleteSuspendTag(ctx, client, conf.ScheduleTagSuspend, event.InstanceID)
			if err != nil {
				log.Printf("unable to remove tag %s", conf.ScheduleTagSuspend)
				return fmt.Sprintf("unable to remove tag %s", conf.ScheduleTagSuspend), err
			}
		}

		// uncomment scheduleTag
		if *tag.Key == conf.ScheduleTag {
			err := createTags(ctx, client, event.InstanceID, []*types.Tag{
				{
					Key:   aws.String(conf.ScheduleTag),
					Value: aws.String(strings.Replace(*tag.Value, "#", "", -1)),
				},
			})
			if err != nil {
				log.Printf("unable to uncomment tag %s", conf.ScheduleTag)
				return fmt.Sprintf("unable to uncomment tag %s", conf.ScheduleTag), err
			}
		}
	}

	log.Printf("instance %s scheduler unsuspended", event.InstanceID)
	return fmt.Sprintf("instance %s scheduler unsuspended", event.InstanceID), nil
}

func deleteSuspendTag(ctx context.Context, client *ec2.Client, tag, instanceID string) error {
	_, err := client.DeleteTags(ctx, &ec2.DeleteTagsInput{
		Resources: []*string{aws.String(instanceID)},
		Tags: []*types.Tag{
			{
				Key: aws.String(tag),
			},
		},
	})
	if err != nil {
		return err
	}

	return nil
}

func createTags(ctx context.Context, client *ec2.Client, instanceID string, tags []*types.Tag) error {
	_, err := client.CreateTags(ctx, &ec2.CreateTagsInput{
		Resources: []*string{aws.String(instanceID)},
		Tags:      tags,
	})
	if err != nil {
		return err
	}

	return nil
}
