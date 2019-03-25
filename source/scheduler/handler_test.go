package main

import (
	"fmt"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/ec2iface"
	"github.com/stretchr/testify/assert"
)

type mockAWSClient struct {
	ec2iface.EC2API
	startInstancesResponse *ec2.StartInstancesOutput
	startInstancesError    error
	stopInstancesResponse  *ec2.StopInstancesOutput
	stopInstancesError     error
}

func init() {
	// disable logger
	// log.SetOutput(ioutil.Discard)
}

func (m *mockAWSClient) StartInstancesRequest(input *ec2.StartInstancesInput) ec2.StartInstancesRequest {
	return ec2.StartInstancesRequest{
		Request: &aws.Request{
			Data:  m.startInstancesResponse,
			Error: m.startInstancesError,
		},
	}
}

func (m *mockAWSClient) StopInstancesRequest(input *ec2.StopInstancesInput) ec2.StopInstancesRequest {
	return ec2.StopInstancesRequest{
		Request: &aws.Request{
			Data:  m.stopInstancesResponse,
			Error: m.stopInstancesError,
		},
	}
}

func TestShouldRunDay(t *testing.T) {
	tests := []struct {
		name    string
		s       *scheduler
		weekday time.Weekday
		want    bool
	}{
		{
			name: "weekdays not defined - Mon-Fri",
			s: &scheduler{
				instanceID: "i-07d023c826d243165",
			},
			weekday: time.Monday,
			want:    true,
		},
		{
			name: "weekdays not defined - Saturday",
			s: &scheduler{
				instanceID: "i-07d023c826d243165",
			},
			weekday: time.Saturday,
			want:    false,
		},

		{
			name: "weekdays defined - Mon,Wed,Thu",
			s: &scheduler{
				instanceID: "i-07d023c826d243165",
				weekdays:   []time.Weekday{1, 3, 5},
			},
			weekday: time.Monday,
			want:    true,
		},
		{
			name: "weekdays defined - wrong day",
			s: &scheduler{
				instanceID: "i-07d023c826d243165",
				weekdays:   []time.Weekday{1, 3, 5},
			},
			weekday: time.Tuesday,
			want:    false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := test.s.shouldRunDay(test.weekday)

			assert.Equal(t, test.want, got)
		})
	}
}

func TestShouldRun(t *testing.T) {
	tests := []struct {
		name    string
		s       *scheduler
		dateNow time.Time
		timeNow time.Time
		want    ec2.InstanceStateName
	}{
		{
			name: "weekend",
			s: &scheduler{
				instanceID: "i-07d023c826d243165",
				startTime:  time.Date(0000, 01, 01, 8, 00, 00, 00, time.UTC),
				stopTime:   time.Date(0000, 01, 01, 19, 00, 00, 00, time.UTC),
			},
			dateNow: time.Date(2019, 01, 06, 00, 00, 00, 00, time.UTC), // Sunday
			timeNow: time.Date(0000, 01, 01, 00, 00, 00, 00, time.UTC), // Sunday
			want:    ec2.InstanceStateNameStopped,
		},
		{
			name: "startTime:stopTime same day",
			s: &scheduler{
				instanceID: "i-07d023c826d243165",
				startTime:  time.Date(0000, 01, 0, 8, 00, 00, 00, time.UTC),
				stopTime:   time.Date(0000, 01, 03, 19, 00, 00, 00, time.UTC),
			},
			timeNow: time.Date(0000, 01, 01, 10, 00, 00, 00, time.UTC),
			want:    ec2.InstanceStateNameRunning,
		},
		{
			name: "startTime:stopTime same day - out of range",
			s: &scheduler{
				instanceID: "i-07d023c826d243165",
				startTime:  time.Date(0000, 01, 01, 8, 00, 00, 00, time.UTC),
				stopTime:   time.Date(0000, 01, 01, 19, 00, 00, 00, time.UTC),
			},
			timeNow: time.Date(0000, 01, 01, 20, 00, 00, 00, time.UTC),
			want:    ec2.InstanceStateNameStopped,
		},
		{
			name: "startTime:stopTime between days - before midnight",
			s: &scheduler{
				instanceID: "i-07d023c826d243165",
				startTime:  time.Date(0000, 01, 01, 19, 00, 00, 00, time.UTC),
				stopTime:   time.Date(0000, 01, 01, 7, 30, 00, 00, time.UTC),
			},
			timeNow: time.Date(0000, 01, 01, 23, 00, 00, 00, time.UTC),
			want:    ec2.InstanceStateNameRunning,
		},
		{
			name: "startTime:stopTime between days - after midnight",
			s: &scheduler{
				instanceID: "i-07d023c826d243165",
				startTime:  time.Date(0000, 01, 01, 19, 00, 00, 00, time.UTC),
				stopTime:   time.Date(0000, 01, 01, 7, 30, 00, 00, time.UTC),
			},
			timeNow: time.Date(0000, 01, 01, 3, 00, 00, 00, time.UTC),
			want:    ec2.InstanceStateNameRunning,
		},
		{
			name: "startTime:stopTime between days - out of range",
			s: &scheduler{
				instanceID: "i-07d023c826d243165",
				startTime:  time.Date(0000, 01, 01, 19, 00, 00, 00, time.UTC),
				stopTime:   time.Date(0000, 01, 01, 7, 30, 00, 00, time.UTC),
			},
			timeNow: time.Date(0000, 01, 01, 8, 00, 00, 00, time.UTC),
			want:    ec2.InstanceStateNameStopped,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := test.s.shouldRun(test.dateNow, test.timeNow)

			fmt.Printf("%s\n", got)
			assert.Equal(t, test.want, got)
		})
	}
}

func TestFixInstanceState(t *testing.T) {
	tests := []struct {
		name          string
		awsClient     *mockAWSClient
		s             *scheduler
		expectedState ec2.InstanceStateName
		err           bool
	}{
		{
			name: "running-running",
			awsClient: &mockAWSClient{
				startInstancesResponse: &ec2.StartInstancesOutput{},
			},
			s: &scheduler{
				instanceID:    "i-07d023c826d243165",
				instanceState: ec2.InstanceStateNameRunning,
			},
			expectedState: ec2.InstanceStateNameRunning,
			err:           false,
		},
		{
			name: "stopped-running",
			awsClient: &mockAWSClient{
				startInstancesResponse: &ec2.StartInstancesOutput{},
			},
			s: &scheduler{
				instanceID:    "i-07d023c826d243165",
				instanceState: ec2.InstanceStateNameStopped,
			},
			expectedState: ec2.InstanceStateNameRunning,
			err:           false,
		},
		{
			name: "stopped-running-error",
			awsClient: &mockAWSClient{
				startInstancesResponse: &ec2.StartInstancesOutput{},
				startInstancesError:    fmt.Errorf("error starting instance"),
			},
			s: &scheduler{
				instanceID:    "i-07d023c826d243165",
				instanceState: ec2.InstanceStateNameStopped,
			},
			expectedState: ec2.InstanceStateNameRunning,
			err:           true,
		},
		{
			name: "running-stopped",
			awsClient: &mockAWSClient{
				stopInstancesResponse: &ec2.StopInstancesOutput{},
			},
			s: &scheduler{
				instanceID:    "i-07d023c826d243165",
				instanceState: ec2.InstanceStateNameRunning,
			},
			expectedState: ec2.InstanceStateNameStopped,
			err:           false,
		},
		{
			name: "running-stopped-error",
			awsClient: &mockAWSClient{
				stopInstancesResponse: &ec2.StopInstancesOutput{},
				stopInstancesError:    fmt.Errorf("error stopping instance"),
			},
			s: &scheduler{
				instanceID:    "i-07d023c826d243165",
				instanceState: ec2.InstanceStateNameRunning,
			},
			expectedState: ec2.InstanceStateNameStopped,
			err:           true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := test.s.fixInstanceState(test.awsClient, test.expectedState)
			if test.err {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
		})
	}
}
