package inmaputil

import (
	"net/http"
	"net/http/httptest"
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
	srv := httptest.NewServer(http.FileServer(http.Dir("../cmd/inmap/testdata/")))
	defer srv.Close()
	if k := maybeDownload(srv.URL+"/testEmis.shp", helperLog(t)); !strings.HasSuffix(k, "testEmis.shp") {
		t.Error("Expected tempDir/testEmis.shp, got ", k)
	}
}

func TestMaybeDownloadRemoteDecompress(t *testing.T) {
	srv := httptest.NewServer(http.FileServer(http.Dir("../cmd/inmap/testdata/")))
	defer srv.Close()
	k := ""
	if k = maybeDownload(srv.URL+"/testEmis.zip", helperLog(t)); !strings.HasSuffix(k, "testEmis/testEmis.shp") {
		t.Error("Expected tempDir/testEmis/testEmis.shp, got ", k)
	}
	t.Logf(k)
}

func TestMaybeDownloadRemoteNestedDecompress(t *testing.T) {
	srv := httptest.NewServer(http.FileServer(http.Dir("../cmd/inmap/testdata/")))
	defer srv.Close()
	k := ""
	if k = maybeDownload(srv.URL+"/nestedTestEmis.zip", helperLog(t)); !strings.HasSuffix(k, "nestedTestEmis/inside/testEmis.shp") {
		t.Error("Expected tempDir/nestedTestEmis/inside/testEmis.shp, got ", k)
	}
	t.Logf(k)
}
