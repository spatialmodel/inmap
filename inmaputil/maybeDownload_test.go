package inmaputil

import (
	"context"
	"net/http"
	"strings"
	"testing"
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

func fileServer(t *testing.T) *http.Server {
	srv := &http.Server{
		Addr:    ":7777",
		Handler: http.FileServer(http.Dir("../inmap/testdata/")),
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil {
			t.Logf("Httpserver: ListenAndServe error: %s", err)
		}
	}()

	return srv
}

func TestMaybeDownloadLocal(t *testing.T) {
	if k := maybeDownload("/dev/null", helperLog(t)); k != "/dev/null" {
		t.Error("Expected /dev/null, got ", k)
	}
}

func TestMaybeDownloadLocal2(t *testing.T) {
	if k := maybeDownload("/blah/test/", helperLog(t)); k != "/blah/test/" {
		t.Error("Expected /blah/test/, got ", k)
	}
}

func TestMaybeDownloadRemoteFail(t *testing.T) {
	if k := maybeDownload("http://blah/test/", helperLog(t)); k != "http://blah/test/" {
		t.Error("Expected http://blah/test/, got ", k)
	}
}

func TestMaybeDownloadRemote(t *testing.T) {
	srv := fileServer(t)
	if k := maybeDownload("http://localhost:7777/testEmis.shp", helperLog(t)); !strings.HasSuffix(k, "testEmis.shp") {
		t.Error("Expected tempDir/testEmis.shp, got ", k)
	}
	if err := srv.Shutdown(context.Background()); err != nil {
		t.Error("Failed shutting down testing server")
	}
}

func TestMaybeDownloadRemoteDecompress(t *testing.T) {
	srv := fileServer(t)
	k := ""
	if k = maybeDownload("http://localhost:7777/testEmis.zip", helperLog(t)); !strings.HasSuffix(k, "testEmis/testEmis.shp") {
		t.Error("Expected tempDir/testEmis/testEmis.shp, got ", k)
	}
	t.Logf(k)

	if err := srv.Shutdown(context.Background()); err != nil {
		t.Error("Failed shutting down testing server")
	}
}

func TestMaybeDownloadRemoteNestedDecompress(t *testing.T) {
	srv := fileServer(t)
	k := ""
	if k = maybeDownload("http://localhost:7777/nestedTestEmis.zip", helperLog(t)); !strings.HasSuffix(k, "nestedTestEmis/inside/testEmis.shp") {
		t.Error("Expected tempDir/nestedTestEmis/inside/testEmis.shp, got ", k)
	}
	t.Logf(k)

	if err := srv.Shutdown(context.Background()); err != nil {
		t.Error("Failed shutting down testing server")
	}
}
