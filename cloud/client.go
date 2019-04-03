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
	"strings"

	"github.com/improbable-eng/grpc-web/go/grpcweb"
	"github.com/lnashier/viper"
	"github.com/spatialmodel/inmap"
	"github.com/spatialmodel/inmap/cloud/cloudrpc"
	"github.com/spf13/cobra"
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

	root   *cobra.Command
	config *viper.Viper

	// inputFileArgs and outputFileArgs list the names of the
	// configuration arguments that represent input and output files.
	inputFileArgs, outputFileArgs []string

	// Image holds the container image to be used.
	// The default is "inmap/inmap:latest".
	Image string

	// Volumes specifies any Kubernetes volumes that are to be
	// mounted in the containers that are created.
	// Each volume will be mounted at /data/volumeName
	// with read-only access.
	Volumes []core.Volume
}

// NewClient creates a new distributed InMAP Kubernetes client.
// root is the root command to be run, config holds simulation configuration
// information, and
// bucketName is the name of a blob storage bucket for storing output files
// in the format gs://bucketname.
// inputFileArgs and outputFileArgs list the names of the
// configuration arguments that represent input and output files.
func NewClient(k kubernetes.Interface, root *cobra.Command, config *viper.Viper, bucketName string, inputFileArgs, outputFileArgs []string) (*Client, error) {
	batchClient := k.BatchV1()
	jobControl := batchClient.Jobs("inmap-distributed")

	c := &Client{
		Interface:      k,
		jobControl:     jobControl,
		bucketName:     bucketName,
		root:           root,
		config:         config,
		inputFileArgs:  inputFileArgs,
		outputFileArgs: outputFileArgs,
		Image:          "inmap/inmap:latest",
	}

	grpcServer := grpc.NewServer(grpc.MaxMsgSize(4.295e+9)) // 4 gib max message size.
	cloudrpc.RegisterCloudRPCServer(grpcServer, c)
	c.WrappedGrpcServer = grpcweb.WrapServer(grpcServer, grpcweb.WithWebsockets(true))

	return c, nil
}

// RunJob creates (and queues) a Kubernetes job with the given name that executes
// the given command with the given command-line arguments on the given container
// image. resources specifies the minimum required resources for execution.
func (c *Client) RunJob(ctx context.Context, job *cloudrpc.JobSpec) (*cloudrpc.JobStatus, error) {
	if job.Version != inmap.Version {
		return nil, fmt.Errorf("incorrect InMAP version: %s != %s", job.Version, inmap.Version)
	}

	status, err := c.Status(ctx, &cloudrpc.JobName{Name: job.Name, Version: job.Version})
	if status.Status != cloudrpc.Status_Missing && err != nil {
		return nil, err
	}
	if status.Status != cloudrpc.Status_Failed && status.Status != cloudrpc.Status_Missing {
		// Only create the job if it is missing or failed.
		return status, nil
	}
	// TODO: Is this necessary?
	if status.Status != cloudrpc.Status_Missing {
		c.Delete(ctx, &cloudrpc.JobName{Name: job.Name, Version: job.Version})
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
		core.ResourceMemory: resource.MustParse(fmt.Sprintf("%dGi", job.MemoryGB)),
	}, c.Volumes)
	_, err = c.jobControl.Create(k8sJob)
	if err != nil {
		return nil, err
	}
	return c.Status(ctx, &cloudrpc.JobName{Name: job.Name, Version: job.Version})
}

// Delete deletes the given job.
func (c *Client) Delete(ctx context.Context, job *cloudrpc.JobName) (*cloudrpc.JobName, error) {
	user, err := getUser(ctx)
	if err != nil {
		return nil, err
	}
	if err = deleteBlobDir(ctx, c.bucketName, user, job.Name); err != nil {
		return nil, err
	}
	p := meta.DeletePropagationForeground
	return job, c.jobControl.Delete(userJobName(user, job.Name), &meta.DeleteOptions{
		PropagationPolicy: &p,
	})
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
	return strings.Replace(user, "_", "-", -1) + "-" + strings.Replace(name, "_", "-", -1)
}

// Status returns the status of the given job.
func (c *Client) Status(ctx context.Context, job *cloudrpc.JobName) (*cloudrpc.JobStatus, error) {
	s := new(cloudrpc.JobStatus)
	k8sJob, err := c.getk8sJob(ctx, job)
	if err != nil {
		return &cloudrpc.JobStatus{
			Status:  cloudrpc.Status_Missing,
			Message: err.Error(),
		}, nil
	}
	for i, cond := range k8sJob.Status.Conditions {
		if i != len(k8sJob.Status.Conditions)-1 {
			continue
		}
		if cond.Type == batch.JobComplete && cond.Status == core.ConditionTrue {
			s.Status = cloudrpc.Status_Complete
			s.StartTime = k8sJob.Status.StartTime.Time.Unix()
			s.CompletionTime = k8sJob.Status.CompletionTime.Time.Unix()
			err := c.checkOutputs(ctx, job.Name, k8sJob.Spec.Template.Spec.Containers[0].Command)
			if err != nil {
				s.Status = cloudrpc.Status_Failed
				s.Message = fmt.Sprintf("job completed but the following error occurred when checking outputs: %s", err)
				return s, nil
			}
		} else if cond.Type == batch.JobFailed && cond.Status == core.ConditionTrue {
			s.Status = cloudrpc.Status_Failed
			s.Message = cond.Message
		}
	}
	if len(k8sJob.Status.Conditions) == 0 {
		if k8sJob.Status.Active > 0 {
			s.Status = cloudrpc.Status_Running
			s.StartTime = k8sJob.Status.StartTime.Time.Unix()
		} else {
			s.Status = cloudrpc.Status_Waiting
		}
	}
	return s, nil
}

// createJob creates a Kubernetes job specification with the given name that executes the
// given command with the given command-line arguments on the given container
// image. resources specifies the minimum required resources for execution.
// volumes holds the list of k8s volumes to mount, with all volumes assumed to
// be read-only.
func createJob(name string, command, args []string, image string, resources core.ResourceList, volumes []core.Volume) *batch.Job {
	volumeMounts := make([]core.VolumeMount, len(volumes))
	for i, v := range volumes {
		volumeMounts[i] = core.VolumeMount{
			Name:      v.Name,
			ReadOnly:  true,
			MountPath: "/data/" + v.Name,
		}
	}

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
							VolumeMounts: volumeMounts,
						},
					},
					Volumes:       volumes,
					RestartPolicy: core.RestartPolicyOnFailure,
				},
			},
		},
	}
}
