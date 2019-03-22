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
	"os/exec"
	"time"

	"github.com/lnashier/viper"
	"github.com/spatialmodel/inmap/cloud/cloudrpc"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	batch "k8s.io/api/batch/v1"
	core "k8s.io/api/core/v1"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

// NewFakeClient creates a client for testing.
// Jobs that are created using this client are run locally.
// The InMAP command must be compiled for it to work,
// e.g., `go install github.com/spatialmodel/inmap/cmd/inmap`.
// The checkConfig and checkRun functions, if not nil, will be run before
// and after executing the inmap command, respectively.
func NewFakeClient(checkConfig func([]string), checkRun func([]byte, error), bucket string, root *cobra.Command, config *viper.Viper, inputFileArgs, outputFileArgs []string) (*Client, error) {
	k8sClient := fake.NewSimpleClientset()
	jobs := make([]batch.Job, 0, 1000)
	k8sClient.Fake.PrependReactor("create", "jobs", fakeRun(checkConfig, checkRun, &jobs))
	k8sClient.Fake.PrependReactor("list", "jobs", fakeList(&jobs))
	return NewClient(k8sClient, root, config, bucket, inputFileArgs, outputFileArgs)
}

// fakeRun runs the InMAP simulation specified by the job.
// The InMAP command must be compiled for it to work,
// e.g., `go install github.com/spatialmodel/inmap/cmd/inmap`.
func fakeRun(checkConfig func([]string), checkRun func([]byte, error), jobs *[]batch.Job) func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
	return func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
		job := action.(k8stesting.CreateAction).GetObject().(*batch.Job)
		cmd := job.Spec.Template.Spec.Containers[0].Command
		args := job.Spec.Template.Spec.Containers[0].Args
		for i := 0; i < len(args); i += 2 {
			cmd = append(cmd, fmt.Sprintf("%s=%s", args[i], args[i+1]))
		}

		if checkConfig != nil {
			checkConfig(cmd)
		}

		xcmd := exec.Command(cmd[0], cmd[1:]...)
		o, err := xcmd.CombinedOutput()
		if checkRun != nil {
			checkRun(o, err)
		}

		// Set status.
		job.Status.Conditions = []batch.JobCondition{{
			Type:   batch.JobComplete,
			Status: core.ConditionTrue,
		}}
		start, err := time.Parse("2006-Jan-02", "2013-Feb-03")
		if err != nil {
			panic(err)
		}
		end, err := time.Parse("2006-Jan-02", "2013-Feb-04")
		if err != nil {
			panic(err)
		}
		s := meta.NewTime(start)
		c := meta.NewTime(end)
		job.Status.StartTime = &s
		job.Status.CompletionTime = &c
		job.Status.Succeeded = 1

		*jobs = append(*jobs, *job)
		return false, job, nil
	}
}

// fakeList returns the job that was most recently run, if any.
func fakeList(jobs *[]batch.Job) func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
	return func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
		return true, &batch.JobList{Items: *jobs}, nil
	}
}

// FakeRPCClient is a local RPC client for testing.
type FakeRPCClient struct {
	Client *Client
}

func (c FakeRPCClient) RunJob(ctx context.Context, job *cloudrpc.JobSpec, op ...grpc.CallOption) (*cloudrpc.JobStatus, error) {
	return c.Client.RunJob(ctx, job)
}

func (c FakeRPCClient) Status(ctx context.Context, job *cloudrpc.JobName, op ...grpc.CallOption) (*cloudrpc.JobStatus, error) {
	return c.Client.Status(ctx, job)
}

func (c FakeRPCClient) Output(ctx context.Context, job *cloudrpc.JobName, op ...grpc.CallOption) (*cloudrpc.JobOutput, error) {
	return c.Client.Output(ctx, job)
}

func (c FakeRPCClient) Delete(ctx context.Context, job *cloudrpc.JobName, op ...grpc.CallOption) (*cloudrpc.JobName, error) {
	return c.Client.Delete(ctx, job)
}
