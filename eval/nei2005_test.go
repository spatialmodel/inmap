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

	dynamic := true
	createGrid := false // this isn't used for the dynamic grid
	os.Setenv("InMAPRunType", "dynamic")
	inmaputil.Cfg.SetConfigFile("nei2005Config.toml")
	cfg, err := inmaputil.LoadConfigFile()
	if err != nil {
		t.Fatal(err)
	}
	if err := inmaputil.Run(cfg, dynamic, createGrid, inmaputil.DefaultScienceFuncs, nil, nil, nil); err != nil {
		t.Fatal(err)
	}

	if err := obsCompare(cfg.OutputFile, cfg.InMAPData, filepath.Join(evalData, "annual_all_2005.csv"),
		filepath.Join(evalData, "states.shp"), filepath.Dir(cfg.OutputFile), "dynamic"); err != nil {
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

	dynamic := false
	createGrid := false
	os.Setenv("InMAPRunType", "static")
	inmaputil.Cfg.SetConfigFile("nei2005Config.toml")
	cfg, err := inmaputil.LoadConfigFile()
	if err != nil {
		t.Fatal(err)
	}
	if err := inmaputil.Run(cfg, dynamic, createGrid, inmaputil.DefaultScienceFuncs, nil, nil, nil); err != nil {
		t.Fatal(err)
	}

	if err := obsCompare(cfg.OutputFile, cfg.InMAPData, filepath.Join(evalData, "annual_all_2005.csv"),
		filepath.Join(evalData, "states.shp"), filepath.Dir(cfg.OutputFile), "static"); err != nil {
		t.Error(err)
	}
}
