package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/external"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
)

var scheduleTag = os.Getenv("SCHEDULE_TAG")
var scheduleTagSuspend = os.Getenv("SCHEDULE_TAG_SUSPEND")

func main() {
	lambda.Start(handler)
}

func handler() error {
	cfg, err := external.LoadDefaultAWSConfig()
	if err != nil {
		return "", err
	}
	client := ec2.New(cfg)

	resp, err := client.DescribeInstancesRequest(&ec2.DescribeInstancesInput{
		Filters: []ec2.Filter{
			{
				Name:   aws.String("instance-state-name"),
				Values: []string{"running", "stopped"},
			},
			{
				Name:   aws.String("tag-key"),
				Values: []string{scheduleTagSuspend},
			},
		},
	}).Send()
	if err != nil {
		return "", err
	}

	if len(resp.Reservations) < 1 {
		log.Printf("no instance found")
		return "", nil
	}

	for _, instance := range resp.Reservations[0].Instances {
		tags := map[string]string{}
		for _, tag := range instace.Tags {
			tags[*tag.Key] = *tag.Value
		}

		// check if suspend time is expired
		if time.Now().After(tags[scheduleTagSuspend]) {
			// delete suspend tag
			err := deleteSuspendTag(client, instance.InstanceId)
			if err != nil {
				log.Printf("unable to remove tag %s", scheduleTagSuspend)
				return fmt.Sprintf("unable to remove tag %s", scheduleTagSuspend), err
			}

			// uncomment scheduleTag
			err := enableScheduleTag(client, event.InstanceID, ec2.Tag{
				Key:   aws.String(scheduleTag),
				Value: aws.String(strings.Replace(tags[scheduleTag], "#", "", -1)),
			})
			if err != nil {
				log.Printf("unable to uncomment tag %s", scheduleTag)
				return fmt.Sprintf("unable to uncomment tag %s", scheduleTag), err
			}
		}
	}

	return nil
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

func enableScheduleTag(client *ec2.EC2, instanceID string, tag ec2.Tag) error {
	_, err := client.CreateTagsRequest(&ec2.CreateTagsInput{
		Resources: []string{instanceID},
		Tags: []ec2.Tag{
			tag,
		},
	}).Send()
	if err != nil {
		return err
	}

	return nil
}
