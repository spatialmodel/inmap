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
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"gocloud.dev/blob"
	"gocloud.dev/blob/fileblob"
)

func helperLog(t *testing.T) chan string {
	outChan := make(chan string)
	go func() {
		for {
			msg := <-outChan
			t.Logf(msg)
		}
	}()
	return outChan
}

func TestMaybeDownloadLocal(t *testing.T) {
	if k := maybeDownload(context.Background(), "/dev/null", helperLog(t)); k != "/dev/null" {
		t.Error("Expected /dev/null, got ", k)
	}
}

func TestMaybeDownloadLocal2(t *testing.T) {
	if k := maybeDownload(context.Background(), "/blah/test/", helperLog(t)); k != "/blah/test/" {
		t.Error("Expected /blah/test/, got ", k)
	}
}

func TestMaybeDownloadRemoteFail(t *testing.T) {
	if k := maybeDownload(context.Background(), "httpz://blah/test/", helperLog(t)); k != "httpz://blah/test/" {
		t.Error("Expected httpz://blah/test/, got ", k)
	}
}

func TestMaybeDownloadRemote(t *testing.T) {
	srv := httptest.NewServer(http.FileServer(http.Dir("../cmd/inmap/testdata/")))
	defer srv.Close()
	if k := maybeDownload(context.Background(), srv.URL+"/testEmis.shp", helperLog(t)); !strings.HasSuffix(k, "testEmis.shp") {
		t.Error("Expected tempDir/testEmis.shp, got ", k)
	}
}

func TestMaybeDownload_bucket(t *testing.T) {
	dir := "download_test"
	if err := os.Mkdir(dir, os.ModePerm); err != nil {
		t.Error(err)
	}
	defer os.RemoveAll(dir)
	bucket, err := fileblob.OpenBucket(dir, nil)
	if err != nil {
		t.Fatal(err)
	}

	// Create a test file in the bucket.
	ctx := context.Background()
	w, err := bucket.NewWriter(ctx, "test.txt", &blob.WriterOptions{})
	if err != nil {
		t.Fatal(err)
	}
	fmt.Fprint(w, "This is a test file.")
	w.Close()

	path := maybeDownload(ctx, "file://download_test/test.txt", helperLog(t))
	if !strings.HasSuffix(path, "test.txt") || strings.HasPrefix(path, "file://") {
		t.Errorf("inproperly downloaded: %s", path)
	}
}
