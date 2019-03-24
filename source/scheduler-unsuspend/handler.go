package main

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
	InstanceID string `json:"instanceId"`
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
		// remove suspend tag (scheduleTagSuspend)
		if *tag.Key == scheduleTagSuspend {
			err := deleteSuspendTag(client, event.InstanceID)
			if err != nil {
				log.Printf("unable to remove tag %s", scheduleTagSuspend)
				return fmt.Sprintf("unable to remove tag %s", scheduleTagSuspend), err
			}
		}

		// uncomment scheduleTag
		if *tag.Key == scheduleTag {
			err := enableScheduleTag(client, event.InstanceID, ec2.Tag{
				Key:   aws.String(scheduleTag),
				Value: aws.String(strings.Replace(*tag.Value, "#", "", -1)),
			})
			if err != nil {
				log.Printf("unable to uncomment tag %s", scheduleTag)
				return fmt.Sprintf("unable to uncomment tag %s", scheduleTag), err
			}
		}
	}

	log.Printf("instance %s scheduler unsuspended", event.InstanceID)
	return fmt.Sprintf("instance %s scheduler unsuspended", event.InstanceID), nil
}

func deleteSuspendTag(client *ec2.EC2, instanceID string) error {
	_, err := client.DeleteTagsRequest(&ec2.DeleteTagsInput{
		Resources: []string{instanceID},
		Tags: []ec2.Tag{
			{
				Key: aws.String(scheduleTagSuspend),
			},
		},
	}).Send()
	if err != nil {
		return err
	}

	return nil
}

func enableScheduleTag(client *ec2.EC2, instanceID string, scheduleTag ec2.Tag) error {
	_, err := client.CreateTagsRequest(&ec2.CreateTagsInput{
		Resources: []string{instanceID},
		Tags: []ec2.Tag{
			scheduleTag,
		},
	}).Send()
	if err != nil {
		return err
	}

	return nil
}
