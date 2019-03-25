package main

import (
	"fmt"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/stretchr/testify/assert"
)

func init() {
	// disable logger
	// log.SetOutput(ioutil.Discard)
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
		name string
		s    *scheduler
		now  time.Time
		want ec2.InstanceStateName
	}{
		{
			name: "startTime:stopTime same day",
			s: &scheduler{
				instanceID: "i-07d023c826d243165",
				startTime:  time.Date(0000, 01, 01, 8, 00, 00, 00, time.UTC),
				stopTime:   time.Date(0000, 01, 01, 19, 00, 00, 00, time.UTC),
			},
			now:  time.Date(0000, 01, 01, 10, 00, 00, 00, time.UTC),
			want: ec2.InstanceStateNameRunning,
		},
		{
			name: "startTime:stopTime same day - out of range",
			s: &scheduler{
				instanceID: "i-07d023c826d243165",
				startTime:  time.Date(0000, 01, 01, 8, 00, 00, 00, time.UTC),
				stopTime:   time.Date(0000, 01, 01, 19, 00, 00, 00, time.UTC),
			},
			now:  time.Date(0000, 01, 01, 20, 00, 00, 00, time.UTC),
			want: ec2.InstanceStateNameStopped,
		},
		{
			name: "startTime:stopTime between days",
			s: &scheduler{
				instanceID: "i-07d023c826d243165",
				startTime:  time.Date(0000, 01, 01, 19, 00, 00, 00, time.UTC),
				stopTime:   time.Date(0000, 01, 01, 7, 30, 00, 00, time.UTC),
			},
			now:  time.Date(0000, 01, 01, 3, 00, 00, 00, time.UTC),
			want: ec2.InstanceStateNameRunning,
		},
		{
			name: "startTime:stopTime between days - out of range",
			s: &scheduler{
				instanceID: "i-07d023c826d243165",
				startTime:  time.Date(0000, 01, 01, 19, 00, 00, 00, time.UTC),
				stopTime:   time.Date(0000, 01, 01, 7, 30, 00, 00, time.UTC),
			},
			now:  time.Date(0000, 01, 01, 8, 00, 00, 00, time.UTC),
			want: ec2.InstanceStateNameStopped,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// now, _ := time.Parse("15:04", test.now)
			// fmt.Printf("NOW: %s\n", now)
			got := test.s.shouldRun(test.now)

			fmt.Printf("%s\n", got)
			assert.Equalf(t, test.want, got, "instance state received: %s\n", got)
		})
	}
}
