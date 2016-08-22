package eval

import (
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof" // pprof serves a performance profiler.
	"os"
	"path/filepath"
	"testing"

	"github.com/spatialmodel/inmap/inmap/cmd"
)

const evalDataEnv = "evaldata"

var evalData string // the location of the downloaded evaluation data directory

func init() {

	evalData = os.Getenv(evalDataEnv)
	if evalData == "" {
		panic(fmt.Errorf("please set the '%s' environment variable to the location of the "+
			"downloaded evaluation data and try again", evalDataEnv))
	}

	go func() {
		// Start a web server for performance profiling.
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()
}

func TestNEI2005Dynamic(t *testing.T) {
	if testing.Short() {
		return
	}

	os.MkdirAll("nei2005", os.ModePerm)

	dynamic := true
	createGrid := false // this isn't used for the dynamic grid
	os.Setenv("InMAPRunType", "dynamic")
	if err := cmd.Startup("nei2005Config.toml"); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Run(dynamic, createGrid); err != nil {
		t.Fatal(err)
	}

	cfg := cmd.Config
	if err := obsCompare(cfg.OutputFile, cfg.InMAPData, filepath.Join(evalData, "annual_all_2005.csv"),
		filepath.Join(evalData, "states.shp"), filepath.Dir(cfg.OutputFile), "dynamic"); err != nil {
		t.Error(err)
	}
}

func TestNEI2005Static(t *testing.T) {
	if testing.Short() {
		return
	}

	os.MkdirAll("nei2005", os.ModePerm)

	dynamic := false
	createGrid := false
	os.Setenv("InMAPRunType", "static")
	if err := cmd.Startup("nei2005Config.toml"); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Run(dynamic, createGrid); err != nil {
		t.Fatal(err)
	}

	cfg := cmd.Config
	if err := obsCompare(cfg.OutputFile, cfg.InMAPData, filepath.Join(evalData, "annual_all_2005.csv"),
		filepath.Join(evalData, "states.shp"), filepath.Dir(cfg.OutputFile), "static"); err != nil {
		t.Error(err)
	}
}
