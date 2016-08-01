package eval

import (
	"log"
	"net/http"
	_ "net/http/pprof" // pprof serves a performance profiler.
	"os"
	"path/filepath"
	"testing"

	"github.com/spatialmodel/inmap/inmap/cmd"
)

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

	dynamic := true
	createGrid := false // this isn't used for the dynamic grid
	os.Setenv("InMAPRunType", "dynamic")
	if err := cmd.Startup("nei2005/nei2005Config.toml"); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Run(dynamic, createGrid); err != nil {
		t.Fatal(err)
	}

	cfg := cmd.Config
	if err := obsCompare(cfg.OutputFile, cfg.InMAPData, os.Getenv("AQSObs2005"),
		os.Getenv("StatesShapefile"), filepath.Dir(cfg.OutputFile), "dynamic"); err != nil {
		t.Error(err)
	}
}

func TestNEI2005Static(t *testing.T) {
	if testing.Short() {
		return
	}

	dynamic := false
	createGrid := false
	os.Setenv("InMAPRunType", "static")
	if err := cmd.Startup("nei2005/nei2005Config.toml"); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Run(dynamic, createGrid); err != nil {
		t.Fatal(err)
	}

	cfg := cmd.Config
	if err := obsCompare(cfg.OutputFile, cfg.InMAPData, os.Getenv("AQSObs2005"),
		os.Getenv("StatesShapefile"), filepath.Dir(cfg.OutputFile), "static"); err != nil {
		t.Error(err)
	}
}
