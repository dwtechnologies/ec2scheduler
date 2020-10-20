package main

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/stretchr/testify/assert"
)

var _ ec2ClientAPI = (*mockEC2client)(nil)

type mockEC2client struct {
	err error
}

const instanceID = "i-07d023c826d243165"

func (m *mockEC2client) StartInstances(ctx context.Context, params *ec2.StartInstancesInput, optFns ...func(*ec2.Options)) (*ec2.StartInstancesOutput, error) {
	return &ec2.StartInstancesOutput{}, m.err
}

func (m *mockEC2client) StopInstances(ctx context.Context, params *ec2.StopInstancesInput, optFns ...func(*ec2.Options)) (*ec2.StopInstancesOutput, error) {
	return &ec2.StopInstancesOutput{}, m.err
}

func TestShouldRunDay(t *testing.T) {
	tests := []struct {
		name    string
		sch     *scheduler
		weekday time.Weekday
		want    bool
	}{
		{
			name: "weekdays not defined - Mon-Fri",
			sch: &scheduler{
				instanceID: instanceID,
			},
			weekday: time.Monday,
			want:    true,
		},
		{
			name: "weekdays not defined - Saturday",
			sch: &scheduler{
				instanceID: instanceID,
			},
			weekday: time.Saturday,
			want:    false,
		},

		{
			name: "weekdays defined - Mon,Wed,Thu",
			sch: &scheduler{
				instanceID: instanceID,
				weekdays:   []time.Weekday{1, 3, 5},
			},
			weekday: time.Monday,
			want:    true,
		},
		{
			name: "weekdays defined - wrong day",
			sch: &scheduler{
				instanceID: instanceID,
				weekdays:   []time.Weekday{1, 3, 5},
			},
			weekday: time.Tuesday,
			want:    false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := test.sch.shouldRunDay(test.weekday)

			assert.Equal(t, test.want, got)
		})
	}
}

func TestShouldRun(t *testing.T) {
	tests := []struct {
		name    string
		sch     *scheduler
		dateNow time.Time
		timeNow time.Time
		want    types.InstanceStateName
	}{
		{
			name: "weekend and scheduler suspended",
			sch: &scheduler{
				instanceID:    instanceID,
				instanceState: types.InstanceStateNameRunning,
				suspended:     true,
			},
			dateNow: time.Date(2019, 01, 06, 00, 00, 00, 00, time.UTC), // Sunday
			timeNow: time.Date(0000, 01, 01, 00, 00, 00, 00, time.UTC), // Sunday
			want:    types.InstanceStateNameRunning,
		},

		{
			name: "weekend",
			sch: &scheduler{
				instanceID: instanceID,
				startTime:  time.Date(0000, 01, 01, 8, 00, 00, 00, time.UTC),
				stopTime:   time.Date(0000, 01, 01, 19, 00, 00, 00, time.UTC),
			},
			dateNow: time.Date(2019, 01, 06, 00, 00, 00, 00, time.UTC), // Sunday
			timeNow: time.Date(0000, 01, 01, 00, 00, 00, 00, time.UTC), // Sunday
			want:    types.InstanceStateNameStopped,
		},
		{
			name: "startTime:stopTime same day",
			sch: &scheduler{
				instanceID: instanceID,
				startTime:  time.Date(0000, 01, 0, 8, 00, 00, 00, time.UTC),
				stopTime:   time.Date(0000, 01, 03, 19, 00, 00, 00, time.UTC),
			},
			timeNow: time.Date(0000, 01, 01, 10, 00, 00, 00, time.UTC),
			want:    types.InstanceStateNameRunning,
		},
		{
			name: "startTime:stopTime same day - out of range",
			sch: &scheduler{
				instanceID: instanceID,
				startTime:  time.Date(0000, 01, 01, 8, 00, 00, 00, time.UTC),
				stopTime:   time.Date(0000, 01, 01, 19, 00, 00, 00, time.UTC),
			},
			timeNow: time.Date(0000, 01, 01, 20, 00, 00, 00, time.UTC),
			want:    types.InstanceStateNameStopped,
		},
		{
			name: "startTime:stopTime between days - before midnight",
			sch: &scheduler{
				instanceID: instanceID,
				startTime:  time.Date(0000, 01, 01, 19, 00, 00, 00, time.UTC),
				stopTime:   time.Date(0000, 01, 01, 7, 30, 00, 00, time.UTC),
			},
			timeNow: time.Date(0000, 01, 01, 23, 00, 00, 00, time.UTC),
			want:    types.InstanceStateNameRunning,
		},
		{
			name: "startTime:stopTime between days - after midnight",
			sch: &scheduler{
				instanceID: instanceID,
				startTime:  time.Date(0000, 01, 01, 19, 00, 00, 00, time.UTC),
				stopTime:   time.Date(0000, 01, 01, 7, 30, 00, 00, time.UTC),
			},
			timeNow: time.Date(0000, 01, 01, 3, 00, 00, 00, time.UTC),
			want:    types.InstanceStateNameRunning,
		},
		{
			name: "startTime:stopTime between days - out of range",
			sch: &scheduler{
				instanceID: instanceID,
				startTime:  time.Date(0000, 01, 01, 19, 00, 00, 00, time.UTC),
				stopTime:   time.Date(0000, 01, 01, 7, 30, 00, 00, time.UTC),
			},
			timeNow: time.Date(0000, 01, 01, 8, 00, 00, 00, time.UTC),
			want:    types.InstanceStateNameStopped,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := test.sch.shouldRun(test.dateNow, test.timeNow)

			fmt.Printf("%s\n", got)
			assert.Equal(t, test.want, got)
		})
	}
}

func TestFixInstanceState(t *testing.T) {
	tests := []struct {
		name   string
		client *mockEC2client
		sch    *scheduler
		want   types.InstanceStateName
		err    bool
	}{
		{
			name:   "running to running",
			client: &mockEC2client{},
			sch: &scheduler{
				instanceID:    instanceID,
				instanceState: types.InstanceStateNameRunning,
			},
			want: types.InstanceStateNameRunning,
			err:  false,
		},
		{
			name:   "stopped to running",
			client: &mockEC2client{},
			sch: &scheduler{
				instanceID:    instanceID,
				instanceState: types.InstanceStateNameStopped,
			},
			want: types.InstanceStateNameRunning,
			err:  false,
		},
		{
			name: "stopped to running - error",
			client: &mockEC2client{
				err: fmt.Errorf("error starting instance"),
			},
			sch: &scheduler{
				instanceID:    instanceID,
				instanceState: types.InstanceStateNameStopped,
			},
			want: types.InstanceStateNameRunning,
			err:  true,
		},
		{
			name:   "running to stopped",
			client: &mockEC2client{},
			sch: &scheduler{
				instanceID:    instanceID,
				instanceState: types.InstanceStateNameRunning,
			},
			want: types.InstanceStateNameStopped,
			err:  false,
		},
		{
			name: "running to stopped - error",
			client: &mockEC2client{
				err: fmt.Errorf("error stopping instance"),
			},
			sch: &scheduler{
				instanceID:    instanceID,
				instanceState: types.InstanceStateNameRunning,
			},
			want: types.InstanceStateNameStopped,
			err:  true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := test.sch.fixInstanceState(context.Background(), test.client, test.want)
			if test.err {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
		})
	}
}
