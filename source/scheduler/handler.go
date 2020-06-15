package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/ec2iface"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/aws-sdk-go-v2/service/sns/snsiface"
	"github.com/caarlos0/env/v6"
)

type scheduler struct {
	instanceID    string
	instanceName  string
	instanceState ec2.InstanceStateName

	snsTopicArn string

	startTime time.Time
	stopTime  time.Time
	weekdays  []time.Weekday
}

type config struct {
	ScheduleTag    string `env:"SCHEDULE_TAG" envDefault:"Schedule"`
	ScheduleTagDay string `env:"SCHEDULE_TAG_DAY" envDefault:"ScheduleDay"`
	ScheduleTagSNS string `env:"SCHEDULE_TAG_SNS" envDefault:"ScheduleSNS"`
}

func main() {
	lambda.Start(handler)
}

func handler(ctx context.Context) error {
	// parse env variables
	conf := &config{}
	if err := env.Parse(conf); err != nil {
		log.Printf("%s", err)
		return err
	}

	cfg, err := external.LoadDefaultAWSConfig()
	if err != nil {
		return err
	}
	client := ec2.New(cfg)

	resp, err := client.DescribeInstancesRequest(&ec2.DescribeInstancesInput{
		Filters: []ec2.Filter{
			{
				Name: aws.String("instance-state-name"),
				Values: []string{
					"running",
					"stopped",
				},
			},
			{
				Name: aws.String("tag-key"),
				Values: []string{
					conf.ScheduleTag,
				},
			},
		},
	}).Send(ctx)
	if err != nil {
		return err
	}

	if len(resp.Reservations) < 1 {
		log.Printf("no scheduled instance found")
		return nil
	}

	// outer loop Reservations (instances)
	// inner loop instance.Tags
	// resp.Reservations[i].Instances[0]
	// ec2.DescribeInstancesOutput{Reservations: []ec2.RunInstancesOutput{Instances: []ec2.Instance{}}}
	for _, reservation := range resp.Reservations {
		instance := reservation.Instances[0]
		s := &scheduler{
			instanceID:    *instance.InstanceId,
			instanceState: instance.State.Name,
		}

		for _, tag := range instance.Tags {
			// scheduler disabled
			if *tag.Key == conf.ScheduleTag && strings.Contains(*tag.Value, "#") {
				log.Printf("[%s] scheduler is disabled", s.instanceID)
				break
			}

			// instance name
			if *tag.Key == "Name" {
				s.instanceName = *tag.Value
			}

			// SNS topic Arn
			if *tag.Key == conf.ScheduleTagSNS {
				s.snsTopicArn = *tag.Value
			}

			// get start and stop time from scheduleTag
			if *tag.Key == conf.ScheduleTag {
				startStopTime := strings.Split(*tag.Value, "-")
				s.startTime, err = time.Parse("15:04", startStopTime[0])
				if err != nil {
					log.Printf("[%s] scheduler start time in wrong format %s: %s", s.instanceID, startStopTime[0], err)
					break
				}
				s.stopTime, err = time.Parse("15:04", startStopTime[1])
				if err != nil {
					log.Printf("[%s] scheduler stop time in wrong format %s: %s", s.instanceID, startStopTime[1], err)
					break
				}
			}

			// get week days from scheduleTagDay
			if *tag.Key == conf.ScheduleTagDay {
				err := json.Unmarshal([]byte(fmt.Sprintf("[%s]", *tag.Value)), &s.weekdays)
				if err != nil {
					log.Printf("[%s] unable to unmarshal %s: %s", s.instanceID, conf.ScheduleTagDay, *tag.Value)
				}
			}
		}

		// get instance expected state (running, stopped)
		expectedState := s.shouldRun(time.Now(), time.Date(0000, 01, 01, time.Now().Hour(), time.Now().Minute(), 00, 00, time.UTC))
		stateChange, err := s.fixInstanceState(ctx, client, expectedState)
		if err != nil {
			log.Printf("[%s] unable to change state", s.instanceID)
			continue
		}

		// publish state changes to SNS topic
		if s.snsTopicArn != "" && stateChange != "" {
			client := sns.New(cfg)

			err := s.publishStateChange(client, stateChange)
			if err != nil {
				log.Printf("[%s] unable to notify %s of state change: %s", s.instanceID, s.snsTopicArn, err)
			}

			log.Printf("[%s] notify %s of state change", s.instanceID, s.snsTopicArn)
		}

		log.Printf("\n")
	}

	return nil
}

// splitting time and date logic
// dateNow contains information regarding current date and time
// timeNow contains information regarding current time (null value for YYYY, mm, dd)
func (s *scheduler) shouldRun(dateNow, timeNow time.Time) ec2.InstanceStateName {
	// logging
	log.Printf("[%s] time now: %d:%d", s.instanceID, timeNow.Hour(), timeNow.Minute())
	log.Printf("[%s] weekday: %s", s.instanceID, dateNow.Weekday())
	log.Printf("[%s] start time: %d:%d", s.instanceID, s.startTime.Hour(), s.startTime.Minute())
	log.Printf("[%s] stop time: %d:%d", s.instanceID, s.stopTime.Hour(), s.stopTime.Minute())

	// should not run today
	if !s.shouldRunDay(dateNow.Weekday()) {
		log.Printf("[%s] should not run on %s", s.instanceID, dateNow.Weekday())
		return ec2.InstanceStateNameStopped
	}

	// startTime-stopTime same day (07:00-19:30)
	if s.startTime.Before(s.stopTime) {
		if timeNow.After(s.startTime) && timeNow.Before(s.stopTime) {
			return ec2.InstanceStateNameRunning
		}

		return ec2.InstanceStateNameStopped
	}

	// startTime-stopTime between days (22:00-03:00 = 22:00-23:59,00:00-03:00)
	// startTime-midnight
	if timeNow.After(s.startTime) && timeNow.Before(time.Date(0000, 01, 01, 23, 59, 00, 00, time.UTC)) {
		return ec2.InstanceStateNameRunning
	}
	// midnight-stopTime
	if timeNow.After(time.Date(0000, 01, 01, 00, 00, 00, 00, time.UTC)) && timeNow.Before(s.stopTime) {
		return ec2.InstanceStateNameRunning
	}

	return ec2.InstanceStateNameStopped
}

// check if instance should run based on day of the week
func (s *scheduler) shouldRunDay(weekday time.Weekday) bool {
	// by default run weekdays (1,2,3,4,5)
	if len(s.weekdays) == 0 {
		if weekday != 0 && weekday != 6 {
			return true
		}
	}

	for _, w := range s.weekdays {
		if w == weekday {
			return true
		}
	}

	return false
}

// fix instance state - start or stop
// return instance state and a possible error
func (s *scheduler) fixInstanceState(ctx context.Context, client ec2iface.ClientAPI, expectedState ec2.InstanceStateName) (ec2.InstanceStateName, error) {
	if s.instanceState == expectedState {
		log.Printf("[%s] instance %s. Nothing to do", s.instanceID, s.instanceState)
		return "", nil
	}

	if expectedState == ec2.InstanceStateNameRunning {
		if _, err := client.StartInstancesRequest(&ec2.StartInstancesInput{
			InstanceIds: []string{s.instanceID},
		}).Send(ctx); err != nil {
			return "", err
		}

		log.Printf("[%s] state changed to %s", s.instanceID, ec2.InstanceStateNameRunning)
		return ec2.InstanceStateNameRunning, nil
	}

	if expectedState == ec2.InstanceStateNameStopped {
		if _, err := client.StopInstancesRequest(&ec2.StopInstancesInput{
			InstanceIds: []string{s.instanceID},
		}).Send(ctx); err != nil {
			return "", err
		}

		log.Printf("[%s] state changed to %s", s.instanceID, ec2.InstanceStateNameStopped)
		return ec2.InstanceStateNameStopped, nil
	}

	return "", nil
}

func (s *scheduler) publishStateChange(client snsiface.ClientAPI, stateChange ec2.InstanceStateName) error {
	_, err := client.PublishRequest(&sns.PublishInput{
		Message:  aws.String(fmt.Sprintf("%s (%s) state changed to %s", s.instanceID, s.instanceName, stateChange)),
		TopicArn: aws.String(s.snsTopicArn),
	}).Send(context.Background())
	if err != nil {
		return err
	}

	return nil
}
