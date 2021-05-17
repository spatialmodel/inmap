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

package inmaputil

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/spatialmodel/inmap"
	"github.com/spatialmodel/inmap/cloud"
	"github.com/spatialmodel/inmap/cloud/cloudrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// NewCloudClient creates a new RPC client based on the information in cfg.
func NewCloudClient(cfg *Cfg) (cloudrpc.CloudRPCClient, error) {
	conn, err := grpc.Dial(cfg.GetString("addr"),
		grpc.WithTransportCredentials(credentials.NewClientTLSFromCert(nil, "")),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(4.295e+9), // 4 gib max message size
			grpc.MaxCallSendMsgSize(4.295e+9), // 4 gib max message size
		),
	)
	if err != nil {
		return nil, err
	}
	return cloudrpc.NewCloudRPCClient(conn), nil
}

// CloudJobStart starts a new cloud job based on the information in cfg.
func CloudJobStart(ctx context.Context, c cloudrpc.CloudRPCClient, cfg *Cfg) error {
	in, err := cloud.JobSpec(
		cfg.Root, cfg.Viper,
		cfg.GetString("version"),
		cfg.GetString("job_name"),
		cfg.GetStringSlice("cmds"),
		cfg.InputFiles(),
		int32(cfg.GetInt("memory_gb")),
	)
	if err != nil {
		return err
	}
	return backoff.RetryNotify(
		func() error {
			_, err = c.RunJob(ctx, in)
			return err
		},
		backoff.NewExponentialBackOff(),
		func(err error, d time.Duration) {
			log.Printf("%v: retrying in %v", err, d)
		},
	)
}

// CloudJobStatus checks the status of a cloud job
// based on the information in cfg.
func CloudJobStatus(ctx context.Context, c cloudrpc.CloudRPCClient, cfg *Cfg) (*cloudrpc.JobStatus, error) {
	in := &cloudrpc.JobName{
		Version: inmap.Version,
		Name:    cfg.GetString("job_name"),
	}
	return c.Status(ctx, in)
}

// CloudJobOutput retrieves and saves the output of a cloud job
// based on the information in cfg. The files will be saved
// in `current_dir/job_name`, where current_dir is the directory
// the command is run in.
func CloudJobOutput(ctx context.Context, c cloudrpc.CloudRPCClient, cfg *Cfg) error {
	name := cfg.GetString("job_name")
	in := &cloudrpc.JobName{
		Version: inmap.Version,
		Name:    name,
	}
	output, err := c.Output(ctx, in)
	if err != nil {
		return err
	}
	os.Mkdir(name, os.ModePerm)
	for fname, data := range output.Files {
		w, err := os.Create(filepath.Join(name, fname))
		if err != nil {
			return err
		}
		_, err = w.Write(data)
		if err != nil {
			return err
		}
	}
	return nil
}

// CloudJobDelete deletes the specified cloud job.
func CloudJobDelete(ctx context.Context, name string, c cloudrpc.CloudRPCClient) error {
	in := &cloudrpc.JobName{
		Version: inmap.Version,
		Name:    name,
	}
	_, err := c.Delete(ctx, in)
	return err
}
