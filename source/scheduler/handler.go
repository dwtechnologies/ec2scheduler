package main

import (
	"encoding/json"
	"fmt"
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
var scheduleTagDay = os.Getenv("SCHEDULE_TAG_DAY")

type scheduler struct {
	instanceID    string
	instanceState ec2.InstanceStateName
	startTime     time.Time
	stopTime      time.Time
	weekdays      []time.Weekday
}

func main() {
	lambda.Start(handler)
}

func handler() error {
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
					scheduleTag,
				},
			},
		},
	}).Send()
	if err != nil {
		return err
	}

	if len(resp.Reservations) < 1 {
		log.Printf("no scheduled instance found")
		return nil
	}

	for _, instance := range resp.Reservations[0].Instances {
		s := &scheduler{}
		s.instanceID = *instance.InstanceId
		s.instanceState = instance.State.Name

		for _, tag := range instance.Tags {
			// scheduler disabled
			if *tag.Key == scheduleTag && strings.Contains(*tag.Value, "#") {
				log.Printf("scheduler is disabled for instanceId %s", s.instanceID)
				break
			}

			// get start and stop time from scheduleTag
			if *tag.Key == scheduleTag {
				startStopTime := strings.Split(*tag.Value, "-")
				s.startTime, err = time.Parse("15:04", startStopTime[0])
				if err != nil {
					log.Printf("instance %s scheduler start time in wrong format %s: %s", s.instanceID, startStopTime[0], err)
					break
				}
				s.stopTime, err = time.Parse("15:04", startStopTime[1])
				if err != nil {
					log.Printf("instance %s scheduler stop time in wrong format %s: %s", s.instanceID, startStopTime[1], err)
					break
				}
			}

			// get week days from scheduleTagDay
			if *tag.Key == scheduleTagDay {
				err := json.Unmarshal([]byte(fmt.Sprintf("[%s]", *tag.Value)), &s.weekdays)
				if err != nil {
					log.Printf("unable to unmarshal %s: %s", scheduleTagDay, *tag.Value)
				}
			}
		}

		// get instance expected state (running, stopped)
		expectedState := s.shouldRun(time.Now(), time.Date(0000, 01, 01, time.Now().Hour(), time.Now().Minute(), 00, 00, time.UTC))
		err := s.fixInstanceState(client, expectedState)
		if err != nil {
			log.Printf("unable to change state for instance %s", s.instanceID)
		}
	}

	return nil
}

// splitting time and date logic
// dateNow contains information regarding current date and time
// timeNow contains information regarding current time (null value for YYYY, mm, dd)
func (s *scheduler) shouldRun(dateNow, timeNow time.Time) ec2.InstanceStateName {
	// should not run today
	log.Printf("weekday: %s", dateNow.Weekday())
	if !s.shouldRunDay(dateNow.Weekday()) {
		log.Printf("%s should run today: false", s.instanceID)
		return ec2.InstanceStateNameStopped
	}

	// logging
	log.Printf("time now: %d:%d", timeNow.Hour(), timeNow.Minute())
	log.Printf("%s start time: %s", s.instanceID, s.startTime)
	log.Printf("%s stop time: %s", s.instanceID, s.stopTime)

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
func (s *scheduler) fixInstanceState(client ec2iface.EC2API, expectedState ec2.InstanceStateName) error {
	if s.instanceState == expectedState {
		return nil
	}

	if expectedState == ec2.InstanceStateNameRunning {
		_, err := client.StartInstancesRequest(&ec2.StartInstancesInput{
			InstanceIds: []string{s.instanceID},
		}).Send()
		if err != nil {
			return err
		}

		log.Printf("instance %s state changed to %s", s.instanceID, ec2.InstanceStateNameRunning)
	}

	if expectedState == ec2.InstanceStateNameStopped {
		_, err := client.StopInstancesRequest(&ec2.StopInstancesInput{
			InstanceIds: []string{s.instanceID},
		}).Send()
		if err != nil {
			return err
		}

		log.Printf("instance %s state changed to %s", s.instanceID, ec2.InstanceStateNameStopped)
	}

	return nil
}
