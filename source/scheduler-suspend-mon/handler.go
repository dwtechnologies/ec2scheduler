package main

import (
	"log"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/ec2iface"
)

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

func handler() error {
	// CN regions don't support env variables
	if scheduleTag == "" {
		scheduleTag = "Schedule"
	}
	if scheduleTagSuspend == "" {
		scheduleTagSuspend = "ScheduleSuspendUntil"
	}

	cfg, err := external.LoadDefaultAWSConfig()
	if err != nil {
		return err
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
		return err
	}

	if len(resp.Reservations) < 1 {
		log.Printf("no instance found")
		return nil
	}

	for _, reservation := range resp.Reservations {
		instance := reservation.Instances[0]
		tags := map[string]string{}
		for _, tag := range instance.Tags {
			tags[*tag.Key] = *tag.Value
		}

		// parse suspend time
		if _, ok := scheduleTagSuspendLayouts[len(tags[scheduleTagSuspend])]; !ok {
			log.Printf("layout doesn't match any supported one %s", tags[scheduleTagSuspend])
			break
		}
		suspendTime, err := time.Parse(scheduleTagSuspendLayouts[len(tags[scheduleTagSuspend])], tags[scheduleTagSuspend])
		if err != nil {
			log.Printf("can't parse date %s", tags[scheduleTagSuspend])
			break
		}

		// check if suspend time is expired
		if time.Now().After(suspendTime) {
			log.Printf("suspension tag [%s] expired on instance %s. unsuspending...", tags[scheduleTagSuspend], *instance.InstanceId)

			// delete suspend tag
			err := deleteSuspendTag(client, *instance.InstanceId)
			if err != nil {
				log.Printf("unable to remove tag %s", scheduleTagSuspend)
				return err
			}

			// uncomment scheduleTag
			err = createTags(client, *instance.InstanceId, []ec2.Tag{
				{
					Key:   aws.String(scheduleTag),
					Value: aws.String(strings.Replace(tags[scheduleTag], "#", "", -1)),
				},
			})
			if err != nil {
				log.Printf("unable to uncomment tag %s", scheduleTag)
				return err
			}
		}
	}

	log.Printf("done and dusted")
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
