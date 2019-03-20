package main

// Suspend
// event:
// { "instanceId": "i-00e92a5a9cb7eeb4d", "unsuspendDatetime": "20171117" }

import (
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
)

type inputEvent struct {
	InstanceID        string `json:"instanceId"`
	UnsuspendDatetime string `json:"unsuspendDatetime"`
}

var scheduleTag = os.Getenv("SCHEDULE_TAG")
var scheduleTagSuspend = os.Getenv("SCHEDULE_TAG_SUSPEND")

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
		log.Printf("no instance found %s", event.InstanceID)
		return "", nil
	}

	for _, tag := range resp.Reservations[0].Instances[0].Tags {
		if *tag.Key == scheduleTag {
			err = suspendScheduler(client, event.InstanceID, []ec2.Tag{
				{
					Key:   aws.String(scheduleTagSuspend),
					Value: aws.String(event.UnsuspendDatetime),
				},
				{
					Key:   aws.String(scheduleTag),
					Value: aws.String(fmt.Sprintf("#%s", *tag.Value)),
				},
			})
			if err != nil {
				return "", err
			}

			break
		}

	}

	log.Printf("instance %s scheduler suspended until %s", event.InstanceID, event.UnsuspendDatetime)
	return fmt.Sprintf("instance %s scheduler suspended until %s", event.InstanceID, event.UnsuspendDatetime), nil
}

func suspendScheduler(client *ec2.EC2, instanceID string, tags []ec2.Tag) error {
	_, err := client.CreateTagsRequest(&ec2.CreateTagsInput{
		Resources: []string{instanceID},
		Tags:      tags,
	}).Send()
	if err != nil {
		return err
	}

	return nil
}
