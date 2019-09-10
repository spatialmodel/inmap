// +build FORTRAN

package eval

/*import (
	_ "net/http/pprof" // pprof serves a performance profiler.
	"os"
	"path/filepath"
	"testing"

	"github.com/spatialmodel/inmap"
	"github.com/spatialmodel/inmap/inmaputil"
	"github.com/spatialmodel/inmap/science/chem/mosaic"
)

func TestNEI2005Dynamic_mosaic(t *testing.T) {
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
	os.Setenv("InMAPRunType", "dynamic_mosaic")
	cfg, err := inmaputil.ReadConfigFile("nei2005Config.toml")
	if err != nil {
		t.Fatal(err)
	}

	cfg.OutputVariables["TotalPM25"] = "96.0*SO4 + 62.0*PNO3 + 35.5*Cl + 18.0*NH4 + 95.0*PMSA + 150.0*Aro1 + 150.0*Aro2 + 140.0*Alk1 + 140.0*Ole1 + 184.0*PApi1 + 184.0*PApi2 + 200.0*Lim1 + 200.0*Lim2 + 60.0*CO3 + 23.0*Na + 40.0*Ca + Oin + 2.1*OC + BC"
	cfg.OutputVariables["SOx"] = "SO2" // TODO: Gas species don't have the correct units for comparing to observations.
	cfg.OutputVariables["pSO4"] = "96.0*SO4"
	cfg.OutputVariables["NOx"] = "NO + NO2"
	cfg.OutputVariables["pNO3"] = "62.0*PNO3"
	cfg.OutputVariables["NH3"] = "NH3"
	cfg.OutputVariables["pNH4"] = "18.0*NH4"
	delete(cfg.OutputVariables, "BasePM25")
	cfg.VarGrid.Xnests = []int{18} //, 3} //, 2} //, 2, 2} //, 3, 2, 2}
	cfg.VarGrid.Ynests = []int{14} //, 3} //, 2} //, 2, 2} //, 3, 2, 2}

	m := mosaic.NewMechanism()
	drydep, err := m.DryDep("simple")
	if err != nil {
		t.Fatal(err)
	}
	wetdep, err := m.WetDep("emep")
	if err != nil {
		t.Fatal(err)
	}

	scienceFuncs := []inmap.CellManipulator{
		inmap.UpwindAdvection(),
		inmap.Mixing(),
		inmap.MeanderMixing(),
		drydep,
		wetdep,
		m.Chemistry(),
	}

	if err := inmaputil.Run(cfg, dynamic, createGrid, scienceFuncs, nil, nil, nil, m); err != nil {
		t.Fatal(err)
	}

	if err := obsCompare(cfg.OutputFile, cfg.InMAPData, filepath.Join(evalData, "annual_all_2005.csv"),
		filepath.Join(evalData, "states.shp"), filepath.Dir(cfg.OutputFile), "dynamic"); err != nil {
		t.Error(err)
	}
}*/
