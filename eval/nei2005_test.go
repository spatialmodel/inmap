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

	cfg := inmaputil.InitializeConfig()

	os.Setenv("InMAPRunType", "dynamic")
	cfg.Set("config", "nei2005Config.toml")
	cfg.Root.SetArgs([]string{"run", "steady"})
	if err := cfg.Root.Execute(); err != nil {
		t.Fatal(err)
	}

	outputFile := cfg.GetString("OutputFile")
	inmapData := cfg.GetString("InMAPData")
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

	cfg := inmaputil.InitializeConfig()

	os.Setenv("InMAPRunType", "static")
	cfg.Set("config", "nei2005Config.toml")
	cfg.Set("static", true)
	cfg.Root.SetArgs([]string{"run", "steady"})
	if err := cfg.Root.Execute(); err != nil {
		t.Fatal(err)
	}

	if err := obsCompare(cfg.GetString("OutputFile"), cfg.GetString("InMAPData"), filepath.Join(evalData, "annual_all_2005.csv"),
		filepath.Join(evalData, "states.shp"), filepath.Dir(cfg.GetString("OutputFile")), "static"); err != nil {
		t.Error(err)
	}
}
