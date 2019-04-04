package main

// Suspend
// event:
// { "instanceId": "i-00e92a5a9cb7eeb4d", "unsuspendDatetime": "20171117" }

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/ec2iface"
)

type inputEvent struct {
	InstanceID        string `json:"instanceId"`
	UnsuspendDatetime string `json:"unsuspendDatetime"`
}

var scheduleTag = os.Getenv("SCHEDULE_TAG")
var scheduleTagSuspend = os.Getenv("SCHEDULE_TAG_SUSPEND")
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

func handler(event inputEvent) (string, error) {
	// CN regions don't support env variables
	if scheduleTag == "" {
		scheduleTag = "Schedule"
	}
	if scheduleTagSuspend == "" {
		scheduleTagSuspend = "ScheduleSuspendUntil"
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
		log.Printf("[%s] no instance found", event.InstanceID)
		return fmt.Sprintf("no instance found with ID %s", event.InstanceID), nil
	}

	for _, tag := range resp.Reservations[0].Instances[0].Tags {
		if *tag.Key == scheduleTag {
			err = createTags(client, event.InstanceID, []ec2.Tag{
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

			log.Printf("[%s] scheduler suspended until %s", event.InstanceID, event.UnsuspendDatetime)
			return fmt.Sprintf("instance %s scheduler suspended until %s", event.InstanceID, event.UnsuspendDatetime), nil
		}
	}

	log.Printf("[%s] unable to find %s tag", event.InstanceID, scheduleTag)
	return fmt.Sprintf("unable to find %s tag for instance %s", scheduleTag, event.InstanceID), nil
}

func createTags(client ec2iface.EC2API, instanceID string, tags []ec2.Tag) error {
	_, err := client.CreateTagsRequest(&ec2.CreateTagsInput{
		Resources: []string{instanceID},
		Tags:      tags,
	}).Send()
	if err != nil {
		return err
	}

	return nil
}
