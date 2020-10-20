package main

import (
	"context"
	"log"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/caarlos0/env/v6"
)

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

func handler(ctx context.Context) error {
	// parse env variables
	conf := &lambdaConfig{}
	if err := env.Parse(conf); err != nil {
		log.Printf("%s", err)
		return err
	}

	cfg, err := config.LoadDefaultConfig()
	if err != nil {
		return err
	}
	client := ec2.NewFromConfig(cfg)

	resp, err := client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
		Filters: []*types.Filter{
			{
				Name:   aws.String("instance-state-name"),
				Values: []*string{aws.String("running"), aws.String("stopped")},
			},
			{
				Name:   aws.String("tag-key"),
				Values: []*string{aws.String(conf.ScheduleTagSuspend)},
			},
		},
	})
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
		if _, ok := scheduleTagSuspendLayouts[len(tags[conf.ScheduleTagSuspend])]; !ok {
			log.Printf("[%s] layout doesn't match any supported one %s", *instance.InstanceId, tags[conf.ScheduleTagSuspend])
			continue
		}
		suspendTime, err := time.Parse(scheduleTagSuspendLayouts[len(tags[conf.ScheduleTagSuspend])], tags[conf.ScheduleTagSuspend])
		if err != nil {
			log.Printf("[%s] can't parse date %s", *instance.InstanceId, tags[conf.ScheduleTagSuspend])
			continue
		}

		// check if suspend time is expired
		if time.Now().After(suspendTime) {
			log.Printf("[%s] suspension tag [%s] expired. unsuspending...", *instance.InstanceId, tags[conf.ScheduleTagSuspend])

			// delete suspend tag
			err := deleteSuspendTag(ctx, client, conf.ScheduleTagSuspend, *instance.InstanceId)
			if err != nil {
				log.Printf("[%s] unable to remove tag %s. Error: %s", *instance.InstanceId, conf.ScheduleTagSuspend, err)
				continue
			}

			// uncomment scheduleTag
			err = createTags(ctx, client, *instance.InstanceId, []*types.Tag{
				{
					Key:   aws.String(conf.ScheduleTag),
					Value: aws.String(strings.Replace(tags[conf.ScheduleTag], "#", "", -1)),
				},
			})
			if err != nil {
				log.Printf("[%s] unable to uncomment tag %s. Error: %s", *instance.InstanceId, conf.ScheduleTag, err)
			}
		}
	}

	log.Printf("done and dusted")
	return nil
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
