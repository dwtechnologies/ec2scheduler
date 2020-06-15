package main

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/ec2iface"
	"github.com/stretchr/testify/assert"
)

type mockAWSClient struct {
	ec2iface.ClientAPI
	createTagsOutput *ec2.CreateTagsOutput
	createTagsError  error
}

const instanceID = "i-07d023c826d243165"

func (m *mockAWSClient) CreateTagsRequest(input *ec2.CreateTagsInput) ec2.CreateTagsRequest {
	return ec2.CreateTagsRequest{
		Request: &aws.Request{
			Data:  m.createTagsOutput,
			Error: m.createTagsError,

			HTTPRequest: &http.Request{},
			Retryer:     aws.NoOpRetryer{},
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
			instanceID:  instanceID,
			scheduleTag: "#13:00-14:00",
		},
		{
			name: "disable scheduler error",
			awsClient: &mockAWSClient{
				createTagsError: fmt.Errorf("error creating tags"),
			},
			instanceID:  instanceID,
			scheduleTag: "#13:00-14:00",
			err:         true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := createTags(context.Background(), test.awsClient, test.instanceID, []ec2.Tag{{Key: aws.String("Schedule"), Value: aws.String(test.scheduleTag)}})

			if test.err {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
		})
	}
}
