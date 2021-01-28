package main

import (
	"context"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/stretchr/testify/assert"
)

var _ ec2ClientAPI = (*mockEC2client)(nil)

type mockEC2client struct {
	err error
}

const instanceID = "i-07d023c826d243165"

func (m *mockEC2client) DescribeInstances(ctx context.Context, params *ec2.DescribeInstancesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeInstancesOutput, error) {
	return &ec2.DescribeInstancesOutput{
		Reservations: []types.Reservation{},
	}, m.err
}

func (m *mockEC2client) CreateTags(ctx context.Context, params *ec2.CreateTagsInput, optFns ...func(*ec2.Options)) (*ec2.CreateTagsOutput, error) {
	return &ec2.CreateTagsOutput{}, m.err
}

func TestDisableScheduler(t *testing.T) {
	tests := []struct {
		name        string
		client      *mockEC2client
		instanceID  string
		scheduleTag string
		err         bool
	}{
		{
			name:        "disable scheduler",
			client:      &mockEC2client{},
			instanceID:  instanceID,
			scheduleTag: "#13:00-14:00",
		},
		{
			name: "disable scheduler error",
			client: &mockEC2client{
				err: fmt.Errorf("error creating tags"),
			},
			instanceID:  instanceID,
			scheduleTag: "#13:00-14:00",
			err:         true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := createTags(context.Background(), test.client, test.instanceID, []types.Tag{{Key: aws.String("Schedule"), Value: aws.String(test.scheduleTag)}})

			if test.err {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
		})
	}
}
