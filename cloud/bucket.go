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
	"net/url"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"gocloud.dev/blob"
	"gocloud.dev/blob/fileblob"
	"gocloud.dev/blob/gcsblob"
	"gocloud.dev/blob/s3blob"
	"gocloud.dev/gcp"
)

// OpenBucket returns the blob storage bucket specified by bucketName,
// where bucketName must be in the format 'provider://name' where provider
// is the name of the storage provider and name is the name of the bucket.
// Even if name contains subdirectories, only the base directory name will be
// used when opening the bucket.
// The currently accepted storage providers are "file" for the local filesystem
// (e.g., for testing), "gs" for Google Cloud Storage, and "s3" for AWS S3.
func OpenBucket(ctx context.Context, bucketName string) (*blob.Bucket, error) {
	url, err := url.Parse(bucketName)
	if err != nil {
		return nil, fmt.Errorf("cloud.OpenBucket: %v", err)
	}
	switch url.Scheme {
	case "file":
		return fileblob.OpenBucket(url.Hostname(), nil)
	case "gs":
		return gsBucket(ctx, url.Hostname())
	case "s3":
		return s3Bucket(ctx, url.Hostname())
	default:
		return nil, fmt.Errorf("cloud.OpenBucket: invalid provider %s", url.Scheme)
	}
}

func gsBucket(ctx context.Context, name string) (*blob.Bucket, error) {
	// See here for information on credentials:
	// https://cloud.google.com/docs/authentication/getting-started
	creds, err := gcp.DefaultCredentials(ctx)
	if err != nil {
		return nil, err
	}
	c, err := gcp.NewHTTPClient(gcp.DefaultTransport(), gcp.CredentialsTokenSource(creds))
	if err != nil {
		return nil, err
	}
	return gcsblob.OpenBucket(ctx, c, name, nil)
}

// s3Bucket opens an s3 storage bucket. It assumes the following
// environment variables are set: AWS_REGION, AWS_ACCESS_KEY_ID, and
// AWS_SECRET_ACCESS_KEY.
func s3Bucket(ctx context.Context, name string) (*blob.Bucket, error) {
	region := os.ExpandEnv("AWS_REGION")
	if region == "" {
		region = "us-east-2"
	}
	c := &aws.Config{
		Region:      aws.String(region),
		Credentials: credentials.NewEnvCredentials(),
	}
	s := session.Must(session.NewSession(c))
	return s3blob.OpenBucket(ctx, s, name, nil)
}
