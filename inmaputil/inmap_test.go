/*
Copyright © 2013 the InMAP authors.
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
	"reflect"
	"testing"

	"github.com/spatialmodel/inmap"
	"github.com/spatialmodel/inmap/internal/postgis"
)

// Set up directory location for configuration files.
func init() {
	os.Setenv("INMAP_ROOT_DIR", "../")
}

func TestCreateGrid(t *testing.T) {
	cfg := InitializeConfig()
	cfg.Set("config", "../cmd/inmap/configExample.toml")
	cfg.Root.SetArgs([]string{"grid"})
	if err := cfg.Root.Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestInMAPStaticCreateGrid(t *testing.T) {
	cfg := InitializeConfig()
	cfg.Set("static", true)
	cfg.Set("createGrid", true)
	os.Setenv("InMAPRunType", "static")
	cfg.Set("config", "../cmd/inmap/configExample.toml")
	cfg.Root.SetArgs([]string{"run", "steady"})
	defer os.Remove(os.ExpandEnv("$INMAP_ROOT_DIR/cmd/inmap/testdata/output_static.log"))
	defer inmap.DeleteShapefile(os.ExpandEnv("$INMAP_ROOT_DIR/cmd/inmap/testdata/output_static.shp"))
	if err := cfg.Root.Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestInMAPStaticLoadGrid(t *testing.T) {
	cfg := InitializeConfig()
	cfg.Set("static", true)
	cfg.Set("createGrid", false)
	os.Setenv("InMAPRunType", "staticLoadGrid")
	cfg.Set("config", "../cmd/inmap/configExample.toml")
	cfg.Root.SetArgs([]string{"run", "steady"})
	defer os.Remove(os.ExpandEnv("$INMAP_ROOT_DIR/cmd/inmap/testdata/output_staticLoadGrid.log"))
	defer inmap.DeleteShapefile(os.ExpandEnv("$INMAP_ROOT_DIR/cmd/inmap/testdata/output_staticLoadGrid.shp"))
	if err := cfg.Root.Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestInMAPDynamic(t *testing.T) {
	cfg := InitializeConfig()
	cfg.Set("static", false)
	cfg.Set("createGrid", false) // this isn't used for the dynamic grid
	os.Setenv("InMAPRunType", "dynamic")
	cfg.Set("config", "../cmd/inmap/configExample.toml")
	cfg.Root.SetArgs([]string{"run", "steady"})
	defer os.Remove(os.ExpandEnv("$INMAP_ROOT_DIR/cmd/inmap/testdata/output_dynamic.log"))
	defer inmap.DeleteShapefile(os.ExpandEnv("$INMAP_ROOT_DIR/cmd/inmap/testdata/output_dynamic.shp"))
	if err := cfg.Root.Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestInMAPDynamic_mask(t *testing.T) {
	cfg := InitializeConfig()
	cfg.Set("static", false)
	cfg.Set("createGrid", false) // this isn't used for the dynamic grid
	os.Setenv("InMAPRunType", "dynamic")
	cfg.Set("config", "../cmd/inmap/configExample.toml")
	f, err := os.Create("tmp_mask.json")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove("tmp_mask.json")
	fmt.Fprint(f, `{"type": "Polygon","coordinates": [ [ [-4000, -4000], [4000, -4000], [4000, 4000], [-4000, 4000] ] ] }`)
	cfg.Set("EmissionMaskGeoJSON", "tmp_mask.json")
	cfg.Root.SetArgs([]string{"run", "steady"})
	defer os.Remove(os.ExpandEnv("$INMAP_ROOT_DIR/cmd/inmap/testdata/output_dynamic.log"))
	defer inmap.DeleteShapefile(os.ExpandEnv("$INMAP_ROOT_DIR/cmd/inmap/testdata/output_dynamic.shp"))
	if err := cfg.Root.Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestInMAPDynamic_coards(t *testing.T) {
	ctx := context.Background()
	postGISURL, postgresC := postgis.SetupTestDB(ctx, t, "../emissions/aep/testdata")
	defer postgresC.Terminate(ctx)

	cfg := InitializeConfig()
	cfg.Set("static", false)
	cfg.Set("createGrid", false) // this isn't used for the dynamic grid
	os.Setenv("InMAPRunType", "dynamic_coards")
	cfg.Set("config", "../cmd/inmap/configExample_coards.toml")
	cfg.Set("aep.PostGISURL", postGISURL)
	cfg.Root.SetArgs([]string{"run", "steady"})
	defer os.Remove(os.ExpandEnv("$INMAP_ROOT_DIR/cmd/inmap/testdata/output_dynamic_coards.log"))
	defer inmap.DeleteShapefile(os.ExpandEnv("$INMAP_ROOT_DIR/cmd/inmap/testdata/output_dynamic_coards.shp"))
	if err := cfg.Root.Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestInMAPDynamic_coardsflag(t *testing.T) {
	ctx := context.Background()
	postGISURL, postgresC := postgis.SetupTestDB(ctx, t, "../emissions/aep/testdata")
	defer postgresC.Terminate(ctx)

	cfg := InitializeConfig()
	cfg.Set("static", false)
	cfg.Set("createGrid", false) // this isn't used for the dynamic grid
	os.Setenv("InMAPRunType", "dynamic_coards")
	cfg.Set("config", "../cmd/inmap/configExample_coards.toml")
	cfg.Set("aep.InventoryConfig.COARDSFiles", "{\"all\":[\"${INMAP_ROOT_DIR}/emissions/aep/testdata/emis_coards_hawaii.nc\"]}")
	cfg.Set("aep.PostGISURL", postGISURL)
	cfg.Root.SetArgs([]string{"run", "steady"})
	defer os.Remove(os.ExpandEnv("$INMAP_ROOT_DIR/cmd/inmap/testdata/output_dynamic_coards.log"))
	defer inmap.DeleteShapefile(os.ExpandEnv("$INMAP_ROOT_DIR/cmd/inmap/testdata/output_dynamic_coards.shp"))
	if err := cfg.Root.Execute(); err != nil {
		t.Fatal(err)
	}
}
func TestInMAPDynamicRemote_http(t *testing.T) {
	cfg := InitializeConfig()
	if err := os.Mkdir("test_bucket", os.ModePerm); err != nil {
		t.Error(err)
	}
	defer os.RemoveAll("test_bucket")
	srv := httptest.NewServer(http.FileServer(http.Dir("../cmd/inmap/testdata/")))
	defer srv.Close()
	os.Setenv("TEST_URL", srv.URL)

	cfg.Set("static", false)
	cfg.Set("createGrid", false) // this isn't used for the dynamic grid
	os.Setenv("InMAPRunType", "dynamichttp")
	cfg.Set("config", "../cmd/inmap/configExampleRemote.toml")
	cfg.Root.SetArgs([]string{"run", "steady"})
	if err := cfg.Root.Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestInMAPDynamicRemote_bucket(t *testing.T) {
	cfg := InitializeConfig()
	if err := os.Mkdir("test_bucket", os.ModePerm); err != nil {
		t.Error(err)
	}
	defer os.RemoveAll("test_bucket")
	os.Setenv("TEST_URL", "file://../cmd/inmap/testdata")

	cfg.Set("static", false)
	cfg.Set("createGrid", false) // this isn't used for the dynamic grid
	os.Setenv("InMAPRunType", "dynamicbucket")
	cfg.Set("config", "../cmd/inmap/configExampleRemote.toml")
	cfg.Root.SetArgs([]string{"run", "steady"})
	if err := cfg.Root.Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestGetStringMapString(t *testing.T) {
	cfg := InitializeConfig()
	// Test regular map[string]string.
	save := cfg.Get("VarGrid.MortalityRateColumns")
	cfg.Set("VarGrid.MortalityRateColumns", map[string]string{"allcause": "TotalPop", "whnolmort": "WhiteNoLat"})
	a := GetStringMapString("VarGrid.MortalityRateColumns", cfg.Viper)
	wantA := map[string]string{"allcause": "TotalPop", "whnolmort": "WhiteNoLat"}
	if !reflect.DeepEqual(a, wantA) {
		t.Errorf("b: %v != %v", a, wantA)
	}
	// Test json object.
	cfg.Set("VarGrid.MortalityRateColumns", `{"AllCause":"TotalPop"}`)
	b := GetStringMapString("VarGrid.MortalityRateColumns", cfg.Viper)
	wantB := map[string]string{"AllCause": "TotalPop"}
	if !reflect.DeepEqual(b, wantB) {
		t.Errorf("b: %v != %v", b, wantB)
	}
	cfg.Set("VarGrid.MortalityRateColumns", save)
}
