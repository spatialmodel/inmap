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
	"testing"

	"github.com/spatialmodel/inmap/cloud/cloudrpc"
	"google.golang.org/grpc"
	batch "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

// NewFakeClient creates a client for testing.
// Jobs that are created using this client are run locally.
// The InMAP command must be compiled for it to work,
// e.g., `go install github.com/spatialmodel/inmap/cmd/inmap`.
func NewFakeClient(t *testing.T, checkConfig bool, bucket string) (*Client, error) {
	k8sClient := fake.NewSimpleClientset()
	k8sClient.Fake.PrependReactor("create", "jobs", fakeRun(t, checkConfig))
	return NewClient(k8sClient, bucket)
}

// fakeRun runs the InMAP simulation specified by the job.
// The InMAP command must be compiled for it to work,
// e.g., `go install github.com/spatialmodel/inmap/cmd/inmap`.
func fakeRun(t *testing.T, checkConfig bool) func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
	return func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
		job := action.(k8stesting.CreateAction).GetObject().(*batch.Job)
		cmd := job.Spec.Template.Spec.Containers[0].Command
		args := job.Spec.Template.Spec.Containers[0].Args
		for i := 0; i < len(args); i += 2 {
			cmd = append(cmd, fmt.Sprintf("%s=%s", args[i], args[i+1]))
		}
		if checkConfig {
			wantCmd := []string{"inmap", "run", "steady",
				"--EmissionUnits=tons/year",
				"--EmissionsShapefiles=file://test/test_user/test_job/258bbcefe8c0073d6f323351463be9e9685e74bb92e367ca769b9536ed247213.shp",
				"--InMAPData=file://test/test_user/test_job/434bf26e3fda1ef9cef7e1fa6cc6b5174d11a22b19cbe10d256adc83b2a97d44.ncf",
				"--LogFile=file://test/test_user/test_job/log.txt",
				"--NumIterations=0", "--OutputAllLayers=false",
				"--OutputFile=file://test/test_user/test_job/OutputFile.shp",
				"--OutputVariables={\"TotalPM25\":\"PrimaryPM25 + pNH4 + pSO4 + pNO3 + SOA\",\"TotalPopD\":\"(exp(log(1.078)/10 * TotalPM25) - 1) * TotalPop * AllCause / 100000\"}\n",
				"--VarGrid.CensusFile=file://test/test_user/test_job/72f6717ef5f6f9600378fe5b192776ba142b3e93311c3dfd0b67bfecbe399990.shp",
				"--VarGrid.CensusPopColumns=TotalPop,WhiteNoLat,Black,Native,Asian,Latino",
				"--VarGrid.GridProj=+proj=lcc +lat_1=33.000000 +lat_2=45.000000 +lat_0=40.000000 +lon_0=-97.000000 +x_0=0 +y_0=0 +a=6370997.000000 +b=6370997.000000 +to_meter=1",
				"--VarGrid.HiResLayers=1",
				"--VarGrid.MortalityRateColumns={\"AllCause\":\"TotalPop\",\"AsianMort\":\"Asian\",\"BlackMort\":\"Black\",\"LatinoMort\":\"Latino\",\"NativeMort\":\"Native\",\"WhNoLMort\":\"WhiteNoLat\"}\n",
				"--VarGrid.MortalityRateFile=file://test/test_user/test_job/764874ad5081665459c67d40607f68df6fc689aa695b4822e012aef84cba5394.shp",
				"--VarGrid.PopConcThreshold=1e-09", "--VarGrid.PopDensityThreshold=0.0055",
				"--VarGrid.PopGridColumn=TotalPop", "--VarGrid.PopThreshold=40000", "--VarGrid.VariableGridDx=4000",
				"--VarGrid.VariableGridDy=4000", "--VarGrid.VariableGridXo=-4000", "--VarGrid.VariableGridYo=-4000",
				"--VarGrid.Xnests=2,2,2", "--VarGrid.Ynests=2,2,2",
				"--VariableGridData=file://test/test_user/test_job/6cd7b21b88adfaac1ac16cf4d5a746d6818b17eaa1cbc629899020a0ef2e9ece.gob",
				"--creategrid=false", "--static=false",
			}
			if len(cmd) != len(wantCmd) {
				t.Errorf("wrong command length: %d != %d", len(cmd), len(wantCmd))
			}
			for i, a := range cmd {
				if a != wantCmd[i] {
					t.Errorf("command element %d: '%s' != '%s'", i, a, wantCmd[i])
				}
			}
		}

		xcmd := exec.Command(cmd[0], cmd[1:]...)
		o, err := xcmd.CombinedOutput()
		if err != nil {
			t.Error(err)
		}
		t.Logf("%s", o)
		return false, job, nil
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
