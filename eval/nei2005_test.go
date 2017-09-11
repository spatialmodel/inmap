package eval

import (
	"log"
	"net/http"
	_ "net/http/pprof" // pprof serves a performance profiler.
	"os"
	"path/filepath"
	"testing"

	"github.com/spatialmodel/inmap/inmaputil"
)

const evalDataEnv = "evaldata"

func init() {
	go func() {
		// Start a web server for performance profiling.
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()
}

func TestNEI2005Dynamic(t *testing.T) {
	if testing.Short() {
		return
	}

	evalData := os.Getenv(evalDataEnv)
	if evalData == "" {
		t.Fatalf("please set the '%s' environment variable to the location of the "+
			"downloaded evaluation data and try again", evalDataEnv)
	}

	os.MkdirAll("nei2005", os.ModePerm)

	os.Setenv("InMAPRunType", "dynamic")
	inmaputil.Cfg.Set("config", "nei2005Config.toml")
	inmaputil.Root.SetArgs([]string{"run", "steady"})
	if err := inmaputil.Root.Execute(); err != nil {
		t.Fatal(err)
	}

	outputFile := inmaputil.Cfg.GetString("OutputFile")
	inmapData := inmaputil.Cfg.GetString("InMAPData")
	if err := obsCompare(outputFile, inmapData, filepath.Join(evalData, "annual_all_2005.csv"),
		filepath.Join(evalData, "states.shp"), filepath.Dir(outputFile), "dynamic"); err != nil {
		t.Error(err)
	}
}

func TestNEI2005Static(t *testing.T) {
	if testing.Short() {
		return
	}

	evalData := os.Getenv(evalDataEnv)
	if evalData == "" {
		t.Fatalf("please set the '%s' environment variable to the location of the "+
			"downloaded evaluation data and try again", evalDataEnv)
	}

	os.MkdirAll("nei2005", os.ModePerm)

	os.Setenv("InMAPRunType", "static")
	inmaputil.Cfg.Set("config", "nei2005Config.toml")
	inmaputil.Cfg.Set("static", true)
	inmaputil.Root.SetArgs([]string{"run", "steady"})
	if err := inmaputil.Root.Execute(); err != nil {
		t.Fatal(err)
	}

	if err := obsCompare(inmaputil.Cfg.GetString("OutputFile"), inmaputil.Cfg.GetString("InMAPData"), filepath.Join(evalData, "annual_all_2005.csv"),
		filepath.Join(evalData, "states.shp"), filepath.Dir(inmaputil.Cfg.GetString("OutputFile")), "static"); err != nil {
		t.Error(err)
	}
}
