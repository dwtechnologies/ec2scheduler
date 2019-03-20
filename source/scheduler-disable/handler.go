package main

// Disable scheduler
// event:
// { "instanceId": "i-00e92a5a9cb7eeb4d" }

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
)

type inputEvent struct {
	InstanceID string
}

var scheduleTag = os.Getenv("SCHEDULE_TAG")

func main() {
	lambda.Start(handler)
}

func handler(event inputEvent) (string, error) {
	cfg, err := external.LoadDefaultAWSConfig()
	if err != nil {
		return "", err
	}

	client := ec2.New(cfg)

	resp, err := client.DescribeInstancesRequest(&ec2.DescribeInstancesInput{
		InstanceIds: []string{event.InstanceID},
	}).Send()
	if err != nil {
		return "", err
	}

	if len(resp.Reservations) < 1 {
		log.Printf("no instance with ID %s", event.InstanceID)
		return "", nil
	}

	for _, tag := range resp.Reservations[0].Instances[0].Tags {
		fmt.Printf("%v", tag)

		if *tag.Key == scheduleTag && strings.Contains(*tag.Value, "#") {
			log.Printf("instance scheduler for %s already disabled", event.InstanceID)
			return fmt.Sprintf("instance scheduler for %s already disabled", event.InstanceID), nil
		}

		// disable scheduler
		_, err = client.CreateTagsRequest(&ec2.CreateTagsInput{
			Resources: []string{event.InstanceID},
			Tags: []ec2.Tag{
				{
					Key:   aws.String(scheduleTag),
					Value: aws.String(fmt.Sprintf("#%s", *tag.Value)),
				},
			},
		}).Send()
	}

	return fmt.Sprintf("instance scheduler for %s enabled", event.InstanceID), nil
}
