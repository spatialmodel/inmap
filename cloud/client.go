/*
Copyright © 2018 the InMAP authors.
This file is part of InMAP.

InMAP is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

InMAP is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with InMAP.  If not, see <http://www.gnu.org/licenses/>.
*/

package cloud

import (
	"context"
	"fmt"

	"github.com/improbable-eng/grpc-web/go/grpcweb"
	"github.com/spatialmodel/inmap"
	"github.com/spatialmodel/inmap/cloud/cloudrpc"
	"google.golang.org/grpc"
	batch "k8s.io/api/batch/v1"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	batchclient "k8s.io/client-go/kubernetes/typed/batch/v1"
)

// Client is a Kubernetes client for InMAP.
type Client struct {
	*grpcweb.WrappedGrpcServer

	kubernetes.Interface
	jobControl batchclient.JobInterface

	bucketName string

	// Image holds the container image to be used.
	// The default is "inmap/inmap:latest".
	Image string
}

// NewClient creates a new distributed InMAP Kubernetes client.
// storage is the name of a blob storage bucket for storing output files
// in the format gs://bucketname.
func NewClient(k kubernetes.Interface, bucketName string) (*Client, error) {
	batchClient := k.BatchV1()
	jobControl := batchClient.Jobs("inmap-distributed")

	c := &Client{
		Interface:  k,
		jobControl: jobControl,
		bucketName: bucketName,
		Image:      "inmap/inmap:latest",
	}

	grpcServer := grpc.NewServer()
	cloudrpc.RegisterCloudRPCServer(grpcServer, c)
	c.WrappedGrpcServer = grpcweb.WrapServer(grpcServer, grpcweb.WithWebsockets(true))

	return c, nil
}

// Create creates (and queues) a Kubernetes job with the given name that executes
// the given command with the given command-line arguments on the given container
// image. resources specifies the minimum required resources for execution.
func (c *Client) RunJob(ctx context.Context, job *cloudrpc.JobSpec) (*cloudrpc.JobStatus, error) {
	if job.Version != inmap.Version {
		return nil, fmt.Errorf("incorrect InMAP version: %s != %s", job.Version, inmap.Version)
	}
	if err := c.stageInputs(ctx, job); err != nil {
		return nil, err
	}
	if err := c.setOutputPaths(ctx, job); err != nil {
		return nil, err
	}
	user, err := getUser(ctx)
	if err != nil {
		return nil, err
	}
	k8sJob := createJob(userJobName(user, job.Name), job.Cmd, job.Args, c.Image, core.ResourceList{
		core.ResourceMemory:  resource.MustParse(fmt.Sprintf("%dGi", job.MemoryGB)),
		core.ResourceStorage: resource.MustParse(fmt.Sprintf("%dGi", job.StorageGB)),
	})
	k8sJobResult, err := c.jobControl.Create(k8sJob)
	if err != nil {
		return nil, err
	}
	return c.jobStatus(k8sJobResult)
}

// Status returns the status of the given job.
func (c *Client) Status(ctx context.Context, job *cloudrpc.JobName) (*cloudrpc.JobStatus, error) {
	k8sJob, err := c.getk8sJob(ctx, job)
	if err != nil {
		return nil, err
	}
	return c.jobStatus(k8sJob)
}

func (c *Client) getk8sJob(ctx context.Context, job *cloudrpc.JobName) (*batch.Job, error) {
	if job.Version != inmap.Version {
		return nil, fmt.Errorf("incorrect InMAP version: %s != %s", job.Version, inmap.Version)
	}
	user, err := getUser(ctx)
	if err != nil {
		return nil, err
	}
	jobName := userJobName(user, job.Name)
	jobList, err := c.jobControl.List(meta.ListOptions{})
	if err != nil {
		return nil, err
	}
	for _, k8sJob := range jobList.Items {
		if k8sJob.GetName() == jobName {
			return &k8sJob, nil
		}
	}
	return nil, fmt.Errorf("cannot find job %s", jobName)
}

// getUser returns the "user" value of ctx.
func getUser(ctx context.Context) (string, error) {
	u := ctx.Value("user")
	if _, ok := u.(string); !ok {
		return "", fmt.Errorf("inmap/cloud: invalid user '%v'", u)
	}
	return ctx.Value("user").(string), nil
}

// userJobName returns a combination of the user and job name.
func userJobName(user, name string) string {
	return user + "_" + name
}

func (c *Client) jobStatus(j *batch.Job) (*cloudrpc.JobStatus, error) {
	return &cloudrpc.JobStatus{
		Status: j.Status.String(),
	}, nil
}

// createJob creates a Kubernetes job specification with the given name that executes the
// given command with the given command-line arguments on the given container
// image. resources specifies the minimum required resources for execution.
func createJob(name string, command, args []string, image string, resources core.ResourceList) *batch.Job {
	return &batch.Job{
		TypeMeta: meta.TypeMeta{
			Kind:       "Job",
			APIVersion: "batch/v1",
		},
		ObjectMeta: meta.ObjectMeta{
			Name: name,
		},
		Spec: batch.JobSpec{
			Template: core.PodTemplateSpec{
				ObjectMeta: meta.ObjectMeta{
					Name:   name + "_pod",
					Labels: map[string]string{"app": "inmap-distributed"},
				},
				Spec: core.PodSpec{
					Containers: []core.Container{
						{
							Name:    "inmap-container",
							Image:   image,
							Command: command,
							Args:    args,
							Resources: core.ResourceRequirements{
								Requests: resources,
							},
						},
					},
					RestartPolicy: core.RestartPolicyOnFailure,
				},
			},
		},
	}
}
