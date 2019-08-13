package main

// Disable scheduler
// event:
// { "instanceId": "i-00e92a5a9cb7eeb4d" }

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/ec2iface"
)

type inputEvent struct {
	InstanceID string `json:"instanceId"`
}

var scheduleTag = os.Getenv("SCHEDULE_TAG")

func main() {
	lambda.Start(handler)
}

func handler(ctx context.Context, event inputEvent) (string, error) {
	// CN regions don't support env variables
	if scheduleTag == "" {
		scheduleTag = "Schedule"
	}

	cfg, err := external.LoadDefaultAWSConfig()
	if err != nil {
		return "", err
	}

	client := ec2.New(cfg)

	resp, err := client.DescribeInstancesRequest(&ec2.DescribeInstancesInput{
		InstanceIds: []string{event.InstanceID},
	}).Send(ctx)
	if err != nil {
		return "", err
	}

	if len(resp.Reservations) < 1 {
		log.Printf("[%s] no instance found", event.InstanceID)
		return "", nil
	}

	for _, tag := range resp.Reservations[0].Instances[0].Tags {
		if *tag.Key == scheduleTag {
			if strings.Contains(*tag.Value, "#") {
				log.Printf("[%s] instance scheduler already disabled", event.InstanceID)
				return fmt.Sprintf("instance scheduler for %s already disabled", event.InstanceID), nil
			}

			// disable scheduler
			err := createTags(ctx, client, event.InstanceID, []ec2.Tag{
				{
					Key:   aws.String(scheduleTag),
					Value: aws.String(fmt.Sprintf("#%s", scheduleTag)),
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
