package main

// Disable scheduler
// event:
// { "instanceId": "i-00e92a5a9cb7eeb4d" }

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
	ScheduleTag string `env:"SCHEDULE_TAG" envDefault:"Schedule"`
}

type ec2ClientAPI interface {
	DescribeInstances(ctx context.Context, params *ec2.DescribeInstancesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeInstancesOutput, error)
	CreateTags(ctx context.Context, params *ec2.CreateTagsInput, optFns ...func(*ec2.Options)) (*ec2.CreateTagsOutput, error)
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

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return "", err
	}

	client := ec2.NewFromConfig(cfg)

	resp, err := describeInstances(ctx, client, event.InstanceID)
	if err != nil {
		return "", err
	}
	if len(resp.Reservations) < 1 {
		log.Printf("[%s] no instance found", event.InstanceID)
		return "", nil
	}

	for _, tag := range resp.Reservations[0].Instances[0].Tags {
		if *tag.Key == conf.ScheduleTag {
			if strings.Contains(*tag.Value, "#") {
				log.Printf("[%s] instance scheduler already disabled", event.InstanceID)
				return fmt.Sprintf("instance scheduler for %s already disabled", event.InstanceID), nil
			}

			// disable scheduler
			err := createTags(ctx, client, event.InstanceID, []types.Tag{
				{
					Key:   aws.String(conf.ScheduleTag),
					Value: aws.String(fmt.Sprintf("#%s", conf.ScheduleTag)),
				},
			})
			if err != nil {
				log.Printf("[%s] error disabling scheduler: %s", event.InstanceID, err)
				return "", err
			}
		}
	}

	log.Printf("[%s] instance scheduler disabled", event.InstanceID)
	return fmt.Sprintf("instance scheduler for %s disabled", event.InstanceID), nil
}

func createTags(ctx context.Context, client ec2ClientAPI, instanceID string, tags []types.Tag) error {
	_, err := client.CreateTags(ctx, &ec2.CreateTagsInput{
		Resources: []string{instanceID},
		Tags:      tags,
	})
	if err != nil {
		return err
	}

	return nil
}

func describeInstances(ctx context.Context, client ec2ClientAPI, instanceID string) (*ec2.DescribeInstancesOutput, error) {
	resp, err := client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
		InstanceIds: []string{instanceID},
	})
	if err != nil {
		return nil, err
	}

	return resp, nil
}
