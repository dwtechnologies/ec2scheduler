package main

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/ec2iface"
	"github.com/stretchr/testify/assert"
)

type mockAWSClient struct {
	ec2iface.EC2API
	createTagsResponse *ec2.CreateTagsOutput
	createTagsError    error
}

func (m *mockAWSClient) CreateTagsRequest(input *ec2.CreateTagsInput) ec2.CreateTagsRequest {
	return ec2.CreateTagsRequest{
		Request: &aws.Request{
			Data:  m.createTagsResponse,
			Error: m.createTagsError,
		},
	}
}

func TestDisableScheduler(t *testing.T) {
	tests := []struct {
		name        string
		awsClient   *mockAWSClient
		instanceID  string
		scheduleTag string
		err         bool
	}{
		{
			name:        "disable scheduler",
			awsClient:   &mockAWSClient{},
			instanceID:  "i-07d023c826d243165",
			scheduleTag: "#13:00-14:00",
		},
		{
			name: "disable scheduler error",
			awsClient: &mockAWSClient{
				createTagsError: fmt.Errorf("error creating tags"),
			},
			instanceID:  "i-07d023c826d243165",
			scheduleTag: "#13:00-14:00",
			err:         true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := disableScheduler(test.awsClient, test.instanceID, test.scheduleTag)

			if test.err {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
		})
	}
}