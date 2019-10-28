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

package inmap

import (
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	"reflect"
	"testing"

	"github.com/ctessum/sparse"
)

var regenGoldenFiles bool

func init() {
	// regen_golden_files is a command line flag that, if invoked as in
	// `go test -regen_golden_files`, will regenerate the golden files
	// based on the current output.
	flag.BoolVar(&regenGoldenFiles, "regen_golden_files", false, "regenerate golden files for preprocessor testing")
}

// regenGoldenFile writes out the given data to the given FilePath.
func regenGoldenFile(data *CTMData, filePath string) error {
	f, err := os.Create(filePath)
	if err != nil {
		return err
	}
	return data.Write(f)
}

func TestWRFChemToInMAP(t *testing.T) {
	flag.Parse()
	const tolerance = 1.0e-6

	wrf, err := NewWRFChem("cmd/inmap/testdata/preproc/wrfout_d01_[DATE]", "20050101", "20050103", nil)
	if err != nil {
		t.Fatal(err)
	}
	newData, err := Preprocess(wrf, -2004000, -540000, 12000, 12000)
	if err != nil {
		t.Fatal(err)
	}

	goldenFileName := "cmd/inmap/testdata/preproc/inmapData_WRFChem_golden.ncf"

	if regenGoldenFiles {
		err := regenGoldenFile(newData, goldenFileName)
		if err != nil {
			t.Errorf("regenerating golden file: %v", err)
		}
	}

	cfg := VarGridConfig{}
	f2, err := os.Open(goldenFileName)
	if err != nil {
		t.Fatalf("opening golden file: %v", err)
	}
	goldenData, err := cfg.LoadCTMData(f2)
	if err != nil {
		t.Fatalf("reading golden file: %v", err)
	}
	compareCTMData(goldenData, newData, tolerance, t)
}

func BenchmarkWRFChemToInMAP(b *testing.B) {
	wrf, err := NewWRFChem("cmd/inmap/testdata/preproc/wrfout_d01_[DATE]", "20050101", "20050103", nil)
	if err != nil {
		b.Fatal(err)
	}
	_, err = Preprocess(wrf, -2004000, -540000, 12000, 12000)
	if err != nil {
		b.Fatal(err)
	}
}

func TestGEOSChemToInMAP(t *testing.T) {
	flag.Parse()
	const tolerance = 1.0e-6

	gc, err := NewGEOSChem(
		"cmd/inmap/testdata/preproc/GEOSFP.[DATE].A1.2x25.nc",
		"cmd/inmap/testdata/preproc/GEOSFP.[DATE].A3cld.2x25.nc",
		"cmd/inmap/testdata/preproc/GEOSFP.[DATE].A3dyn.2x25.nc",
		"cmd/inmap/testdata/preproc/GEOSFP.[DATE].I3.2x25.nc",
		"cmd/inmap/testdata/preproc/GEOSFP.[DATE].A3mstE.2x25.nc",
		"",
		"cmd/inmap/testdata/preproc/gc_output.[DATE].nc",
		"cmd/inmap/testdata/preproc/geoschem-new/Olson_2001_Land_Map.025x025.generic.nc",
		"20130102",
		"20130104",
		true,
		"3h",
		"3h",
		true,
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	newData, err := Preprocess(gc, -2.5, 50, 2.5, 2)
	if err != nil {
		t.Fatal(err)
	}

	goldenFileName := "cmd/inmap/testdata/preproc/inmapData_GEOSChem_golden.ncf"

	if regenGoldenFiles {
		err := regenGoldenFile(newData, goldenFileName)
		if err != nil {
			t.Errorf("regenerating golden file: %v", err)
		}
	}

	cfg := VarGridConfig{}
	f2, err := os.Open(goldenFileName)
	if err != nil {
		t.Fatalf("opening golden file: %v", err)
	}
	goldenData, err := cfg.LoadCTMData(f2)
	if err != nil {
		t.Fatalf("reading golden file: %v", err)
	}
	compareCTMData(goldenData, newData, tolerance, t)
}

func BenchmarkGEOSChemToInMAP(b *testing.B) {
	gc, err := NewGEOSChem(
		"cmd/inmap/testdata/preproc/GEOSFP.[DATE].A1.2x25.nc",
		"cmd/inmap/testdata/preproc/GEOSFP.[DATE].A3cld.2x25.nc",
		"cmd/inmap/testdata/preproc/GEOSFP.[DATE].A3dyn.2x25.nc",
		"cmd/inmap/testdata/preproc/GEOSFP.[DATE].I3.2x25.nc",
		"cmd/inmap/testdata/preproc/GEOSFP.[DATE].A3mstE.2x25.nc",
		"",
		"cmd/inmap/testdata/preproc/gc_output.[DATE].nc",
		"cmd/inmap/testdata/preproc/geoschem-new/Olson_2001_Land_Map.025x025.generic.nc",
		"20130102",
		"20130104",
		true,
		"3h",
		"3h",
		true,
		nil,
	)
	if err != nil {
		b.Fatal(err)
	}
	_, err = Preprocess(gc, -2.5, 50, 2.5, 2)
	if err != nil {
		b.Fatal(err)
	}
}

func TestGEOSChemToInMAP_new(t *testing.T) {
	flag.Parse()
	const tolerance = 1.0e-6

	gc, err := NewGEOSChem(
		"cmd/inmap/testdata/preproc/geoschem-new/MERRA2.[DATE].A1.2x25.nc3",
		"cmd/inmap/testdata/preproc/geoschem-new/MERRA2.[DATE].A3cld.2x25.nc3",
		"cmd/inmap/testdata/preproc/geoschem-new/MERRA2.[DATE].A3dyn.2x25.nc3",
		"cmd/inmap/testdata/preproc/geoschem-new/MERRA2.[DATE].I3.2x25.nc3",
		"cmd/inmap/testdata/preproc/geoschem-new/MERRA2.[DATE].A3mstE.2x25.nc3",
		"cmd/inmap/testdata/preproc/geoschem-new/GEOSFP.ApBp.nc",
		"cmd/inmap/testdata/preproc/geoschem-new/ts.[DATE].nc",
		"cmd/inmap/testdata/preproc/geoschem-new/Olson_2001_Land_Map.025x025.generic.nc",
		"20160102",
		"20160103",
		false,
		"3h",
		"24h",
		false,
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	newData, err := Preprocess(gc, -2.5, 50, 2.5, 2)
	if err != nil {
		t.Fatal(err)
	}

	goldenFileName := "cmd/inmap/testdata/preproc/geoschem-new/inmapData_GEOSChem_golden.ncf"

	if regenGoldenFiles {
		err := regenGoldenFile(newData, goldenFileName)
		if err != nil {
			t.Errorf("regenerating golden file: %v", err)
		}
	}

	cfg := VarGridConfig{}
	f2, err := os.Open(goldenFileName)
	if err != nil {
		t.Fatalf("opening golden file: %v", err)
	}
	goldenData, err := cfg.LoadCTMData(f2)
	if err != nil {
		t.Fatalf("reading golden file: %v", err)
	}
	compareCTMData(goldenData, newData, tolerance, t)
}

func compareCTMData(goldenData, newData *CTMData, tolerance float64, t *testing.T) {
	if len(goldenData.Data) != len(newData.Data) {
		t.Errorf("new and old ctmdata have different number of variables (%d vs. %d)",
			len(newData.Data), len(goldenData.Data))
	}
	for name, dd1 := range goldenData.Data {
		if _, ok := newData.Data[name]; !ok {
			t.Errorf("newData doesn't have variable %s", name)
			continue
		}
		dd2 := newData.Data[name]
		if !reflect.DeepEqual(dd1.Dims, dd2.Dims) {
			t.Errorf("%s dims problem: %v != %v", name, dd1.Dims, dd2.Dims)
		}
		if dd1.Description != dd2.Description {
			t.Errorf("%s description problem: %s != %s", name, dd1.Description, dd2.Description)
		}
		if dd1.Units != dd2.Units {
			t.Errorf("%s units problem: %s != %s", name, dd1.Units, dd2.Units)
		}
		arrayCompare(dd2.Data, dd1.Data, tolerance, name, t)
	}
}

func testNextData(v []*sparse.DenseArray) NextData {
	var i int
	return func() (*sparse.DenseArray, error) {
		if i == 2 {
			return nil, io.EOF
		}
		i++
		return v[i-1], nil
	}
}

func TestCalcPartitioning(t *testing.T) {
	const tolerance = 1.0e-8

	gasFunc := testNextData(so2)
	particleFunc := testNextData(sulf)
	partitioning, gasResult, particleResult, err := marginalPartitioning(gasFunc, particleFunc)
	if err != nil {
		t.Fatal(err)
	}

	gasWant := sparse.ZerosDense(10, 2, 2)
	gasWant.Elements = []float64{1.305e-05, 3.415e-05, 2.4999999999999998e-05, 6.85e-05, 1.2775e-05, 3.075e-05, 1.76e-05, 5.4499999999999997e-05, 1.2145e-05, 2.89e-05, 1.3949999999999999e-05, 4.7000000000000004e-05, 1.1565e-05, 2.79e-05, 1.164e-05, 4.2e-05, 1.3555e-06, 2.605e-05, 1.000355e-05, 3.79e-05, 3.85e-08, 2.0255000000000002e-05, 3.615e-07, 3.4512e-05, 1.12e-09, 5.17e-07, 6.0499999999999996e-09, 3.3507e-05, 1.855e-09, 2.5e-09, 3.305e-09, 3.65095e-06, 1.80165e-06, 1.55075e-06, 1.949e-06, 1.6006499999999999e-06, 2.025e-06, 2.314e-06, 2.3699999999999998e-06, 2.645e-06}
	arrayCompare(gasResult, gasWant, tolerance, "gas", t)

	particleWant := sparse.ZerosDense(10, 2, 2)
	particleWant.Elements = []float64{2.15e-05, 2.25e-05, 2.2499999999999998e-05, 2.35e-05, 2.2e-05, 2.3e-05, 2.2499999999999998e-05, 2.35e-05, 2.2e-05, 2.3e-05, 2.2499999999999998e-05, 2.35e-05, 2.2e-05, 2.3e-05, 2.2499999999999998e-05, 2.35e-05, 2.2499999999999998e-05, 2.3e-05, 2.3e-05, 2.4e-05, 2.3e-05, 2.3500000000000002e-05, 2.3500000000000002e-05, 2.4e-05, 2.2499999999999998e-05, 2.4e-05, 2.3e-05, 2.4999999999999998e-05, 2.2499999999999998e-05, 2.3e-05, 2.3500000000000002e-05, 2.3e-05, 3.45e-05, 2.7e-05, 3.35e-05, 2.7e-05, 4.6e-05, 3.65e-05, 4.4e-05, 3.65e-05}
	arrayCompare(particleResult, particleWant, tolerance, "particle", t)

	partitioningWant := sparse.ZerosDense(10, 2, 2)
	partitioningWant.Elements = []float64{0.20930232558139536, 0.125, 0.16964285714285715, 0.08750000000000001, 0.20930232558139536, 0.12987012987012989, 0.19387755102040818, 0.09722222222222222, 0.21428571428571427, 0.13333333333333333, 0.21111111111111114, 0.10294117647058823, 0.21951219512195122, 0.13513513513513514, 0.2261904761904762, 0.10714285714285714, 0.9347826086956521, 0.14084507042253522, 0.2435897435897436, 0.11052631578947368, 0.9978977649922549, 0.16666666666666666, 0.9819888381532218, 0.11666666666666667, 0.9999122339365392, 0.9761904761904762, 0.9993530690641208, 0.12359550561797754, 0.9998614168586635, 0.999887234991758, 0.9997255680087973, 0.8557312252964426, 0.7870222750769289, 0.4467353951890034, 0.6748447147643264, 0.9432624113475178, 0.8820058997050148, 0.7811263902461725, 0.8502055271130794, 0.7681937338109415}
	arrayCompare(partitioning, partitioningWant, tolerance, "partitioning", t)
}

func TestAverage(t *testing.T) {
	const tolerance = 1.0e-8

	dataFunc := testNextData(PHB)
	result, err := average(dataFunc)
	if err != nil {
		t.Fatal(err)
	}
	want := PHB[0].Copy()
	want.AddDense(PHB[1])
	want.Scale(0.5)
	arrayCompare(result, want, tolerance, "average", t)
}

func TestCalcLayerHeights(t *testing.T) {
	const tolerance = 1.0e-8

	layerHeights := geopotentialToHeight(PH[0], PHB[0])
	dz := layerThickness(layerHeights)

	heightsWant := sparse.ZerosDense(11, 2, 2)
	heightsWant.Elements = []float64{0, 0, 0, 0, 60.57114305088894, 60.693508996446276, 50.455558218147914, 50.59831848796478, 141.23069549744307, 131.33944823155716, 131.13550498896157, 131.54339147415274, 241.7747140970668, 242.28457220355577, 231.8834668311809, 242.79443031004473, 362.3051704710579, 373.21613394992175, 362.7130569562491, 373.92993529900633, 563.4951792916032, 534.025380736541, 482.63168360245345, 535.045096949519, 763.7674435204683, 714.1072639484432, 785.1814839930048, 746.0243814146523, 963.6318212641422, 1015.6373481260166, 985.0458617366787, 996.262740079436, 1465.332198049283, 1415.3661036133644, 1384.7746172240265, 1395.991495566784, 1868.1201021755646, 1918.086196611483, 1888.5144264351231, 1898.7115885649025, 2375.938776238573, 2423.8654382485356, 2394.2936680721755, 2403.471113988977}
	arrayCompare(layerHeights, heightsWant, tolerance, "layerHeights", t)

	dzWant := sparse.ZerosDense(10, 2, 2)
	dzWant.Elements = []float64{60.57114305088894, 60.693508996446276, 50.455558218147914, 50.59831848796478, 80.65955244655413, 70.64593923511089, 80.67994677081366, 80.94507298618797, 100.54401859962374, 110.94512397199861, 100.74796184221933, 111.25103883589199, 120.53045637399111, 130.931561746366, 130.8295901250682, 131.1355049889616, 201.1900088205453, 160.80924678661927, 119.91862664620436, 161.11516165051268, 200.27226422886508, 180.0818832119022, 302.54980039055135, 210.97928446513333, 199.8643777436739, 301.53008417757337, 199.8643777436739, 250.23835866478362, 501.70037678514086, 399.7287554873478, 399.7287554873478, 399.72875548734794, 402.78790412628155, 502.7200929981186, 503.7398092110966, 502.7200929981186, 507.81867406300853, 505.77924163705256, 505.77924163705234, 504.75952542407435}
	arrayCompare(dz, dzWant, tolerance, "dz", t)
}

func TestPreprocWetDeposition(t *testing.T) {
	const tolerance = 1.0e-8
	layerHeights := geopotentialToHeight(PH[0], PHB[0])
	dz := layerThickness(layerHeights)
	qrainFunc := testNextData(QRAIN)
	cloudFracFunc := testNextData(QCLOUD)
	altFunc := testNextData(ALT)

	wdParticle, wdSO2, wdOtherGas, err := wetDeposition(dz, qrainFunc, cloudFracFunc, altFunc)
	if err != nil {
		t.Fatal(err)
	}

	// Want values are taken from existing function output, just to avoid regression.
	wdParticleWant := sparse.ZerosDense(10, 2, 2)
	wdParticleWant.Elements = []float64{5.977011494252873e-06, 0, 3.3255813953488373e-06, 0, 6.5000000000000004e-06, 0, 3.885057471264367e-06, 0, 7.386363636363637e-06, 0, 4.183908045977011e-06, 0, 8.179775280898877e-06, 0, 4.727272727272727e-06, 5.744186046511629e-08, 8.28571429759415e-06, 1.8022727272727276e-07, 5.258426966292135e-06, 2.988505747126436e-07, 8.478261073092499e-06, 4.0898876404494386e-07, 5.428571459626166e-06, 5.842696629213484e-07, 6.085106529354578e-06, 4.857142861634599e-07, 6.430107628967074e-06, 7.142857142857143e-07, 6.367346948782296e-06, 2.2715789483519827e-07, 7.237113416859262e-06, 4.7021276991624836e-07, 0, 0, 0, 0, 0, 0, 0, 0}
	arrayCompare(wdParticle, wdParticleWant, tolerance, "wdParticle", t)
	wdSO2Want := sparse.ZerosDense(10, 2, 2)
	wdSO2Want.Elements = []float64{1.4232337745268778e-10, 0, 9.50640880271098e-11, 0, 1.1622925884955751e-10, 0, 6.945280085248075e-11, 0, 1.0595766152268119e-10, 0, 5.989682221136394e-11, 0, 9.788192266012663e-11, 0, 5.2114982640119044e-11, 6.317803650222431e-13, 5.940110066644896e-11, 1.6164691211016316e-12, 6.324513443017658e-11, 2.6753191837625487e-12, 6.108871364243891e-11, 3.275663436700725e-12, 2.5883605476588623e-11, 3.9942150308730816e-12, 4.393471283719606e-11, 2.323382651500026e-12, 4.641774655889978e-11, 4.1169538743652434e-12, 1.8306596836672473e-11, 8.196495043653666e-13, 2.6115288001829442e-11, 1.6972230470530962e-12, 0, 0, 0, 0, 0, 0, 0, 0}
	arrayCompare(wdSO2, wdSO2Want, tolerance, "wdSO2", t)
	wdOtherGasWant := sparse.ZerosDense(10, 2, 2)
	wdOtherGasWant.Elements = []float64{4.74411258175626e-10, 0, 3.1688029342369936e-10, 0, 3.8743086283185844e-10, 0, 2.3150933617493587e-10, 0, 3.5319220507560394e-10, 0, 1.996560740378798e-10, 0, 3.2627307553375545e-10, 0, 1.7371660880039682e-10, 2.1059345500741436e-12, 1.980036688881632e-10, 5.388230403672106e-12, 2.1081711476725523e-10, 8.917730612541829e-12, 2.0362904547479636e-10, 1.091887812233575e-11, 8.627868492196207e-11, 1.3314050102910271e-11, 1.464490427906535e-10, 7.74460883833342e-12, 1.5472582186299924e-10, 1.3723179581217477e-11, 6.10219894555749e-11, 2.7321650145512225e-12, 8.705096000609814e-11, 5.657410156843654e-12, 0, 0, 0, 0, 0, 0, 0, 0}
	arrayCompare(wdOtherGas, wdOtherGasWant, tolerance, "wdOtherGas", t)
}

func TestWindDeviation(t *testing.T) {
	const tolerance = 1.0e-8
	uFunc1 := testNextData(U)
	uFunc2 := testNextData(U)
	uAvg, err := average(uFunc1)
	if err != nil {
		t.Fatal(err)
	}
	uDev, err := windDeviation(uAvg, uFunc2)
	if err != nil {
		t.Fatal(err)
	}
	uDevWant := sparse.ZerosDense(10, 2, 3)
	uDevWant.Elements = []float64{3.4499999999999997, 5.699999999999999, 3.6345, 2.3499999999999996, 4.54, 3.8449999999999998, 3.8499999999999996, 6.199999999999999, 4.26, 2.85, 4.91, 4.46, 3.8500000000000005, 6.2, 4.57, 2.9, 4.76, 4.7265, 3.7499999999999996, 5.9, 4.705, 3.05, 4.625, 4.790000000000001, 4.050000000000001, 5.35, 4.615, 3.75, 4.949999999999999, 4.84, 5, 5.1850000000000005, 4.22, 4.75, 4.949999999999999, 5.074999999999999, 6.045, 4.949999999999999, 3.8299999999999996, 5.550000000000001, 4.75, 5.35, 6.25, 4.4, 3.5, 5.75, 4.15, 3.4000000000000004, 4.55, 4.6, 3.4, 4.95, 3.55, 2.8499999999999996, 3.25, 3.55, 3.35, 3.45, 3.05, 2.7}
	arrayCompare(uDev, uDevWant, tolerance, "windDeviation", t)
}

func TestWindSpeed(t *testing.T) {
	const tolerance = 1.0e-8
	uFunc := testNextData(U)
	vFunc := testNextData(V)
	wFunc := testNextData(W)
	speed, speedInverse, speedMinusThird, speedMinusOnePointFour, uAvg, vAvg, wAvg, err := calcWindSpeed(uFunc, vFunc, wFunc)
	if err != nil {
		t.Fatal(err)
	}

	// Want values are taken from existing function output, just to avoid regression.
	speedWant := sparse.ZerosDense(10, 2, 2)
	speedWant.Elements = []float64{5.631151384654825, 4.719842843105441, 6.1106905110287855, 4.431864596932381, 6.212505755454474, 5.326809535234906, 6.8057312596940065, 4.942228578290417, 6.29934489057941, 5.536324073058163, 6.881146714981624, 5.089255768512075, 6.215927537829076, 5.520876879215075, 6.9090625678540425, 5.211617182737886, 6.089942758684841, 5.294076260919778, 7.4543598035672485, 5.650553899893711, 6.668146623211156, 5.494539371002746, 7.4875456565374705, 5.9782326502780085, 6.827921888802733, 5.805342526007754, 7.216785114279576, 6.05285860982377, 8.349628727563278, 7.0490260660896205, 7.802993774315482, 6.445763652445638, 9.487400014758816, 8.003858963691737, 8.80645579791047, 7.446789744076287, 9.290560551516883, 8.365166363512806, 8.959698965397227, 8.013923215806832}
	speedInverseWant := sparse.ZerosDense(10, 2, 2)
	speedInverseWant.Elements = []float64{0.25500761309430076, 0.363908330142886, 0.2401784937972983, 1.5479288820157384, 0.25318902229670304, 0.3569417877986804, 0.2246942970391589, 1.8593335021876498, 0.2647574602586248, 0.3683825693047293, 0.21894598283788155, 1.3566056417748071, 0.28443123267772097, 0.39536823516477837, 0.22013877392235692, 0.8539758517064696, 0.36360432833967415, 0.5143494336831772, 0.2157910918995023, 0.5859841702515048, 0.3899347876517948, 0.5776960043553734, 0.2300498904663676, 0.4970156551471795, 0.37522528745016537, 0.4172614057473988, 0.2493976384377616, 0.3434338756445628, 0.17642821496013306, 0.19997507640533507, 0.18120715139766316, 0.2374172068957778, 0.13028020346699198, 0.15493397525860153, 0.1381354564124194, 0.15879198406735212, 0.12298608038254136, 0.13934421307495948, 0.1262875221124757, 0.13823642776212025}
	speedMinusThirdWant := sparse.ZerosDense(10, 2, 2)
	speedMinusThirdWant.Elements = []float64{0.6098135612055001, 0.6739479644524811, 0.5963903556716044, 0.9641359919435925, 0.6026094703111714, 0.6627733591047715, 0.5807936483151956, 1.002538227845926, 0.6079733538839961, 0.6651175574344188, 0.5766915657533246, 0.9221966551188707, 0.618980117152311, 0.676441828464123, 0.5771666884142757, 0.8191668158662379, 0.6568970467328519, 0.7229469807038361, 0.5700244883638423, 0.7415628749570478, 0.6620790006965365, 0.7408896521529951, 0.5782421589501993, 0.7089443169887435, 0.654563966368142, 0.6817881584123346, 0.591374803248716, 0.6485265080936607, 0.5379063778262967, 0.5634443894744083, 0.5450595422033542, 0.591509873317563, 0.49533168005446904, 0.5246053773129509, 0.505960975177211, 0.5316452411972752, 0.4900601507448086, 0.5097823508665811, 0.4949333742842665, 0.5112550301580621}
	speedMinusOnePointFourWant := sparse.ZerosDense(10, 2, 2)
	speedMinusOnePointFourWant.Elements = []float64{0.1605188483011608, 0.27239523989679304, 0.1482092084770319, 2.329427852515723, 0.16155882056794363, 0.26919121387697936, 0.13601706241946884, 3.0427553705295063, 0.17368143777041542, 0.2840712625756522, 0.1308213181076628, 1.9370762446052951, 0.1937898182688676, 0.3163680295912299, 0.13204469250165954, 0.9909448818416791, 0.28186285031277053, 0.4686491495804963, 0.12966252904386272, 0.572586309579191, 0.31654060963340996, 0.5592871191138545, 0.14340262275579158, 0.45043044947118277, 0.29948512661127324, 0.34537320770258456, 0.16165470499759152, 0.2581529469638711, 0.09629874282058462, 0.11380931663309213, 0.09920053604585943, 0.14693942964048434, 0.06078678002891349, 0.07753532020004533, 0.0657440849990742, 0.07939003385171392, 0.055064142042121, 0.0658972365388769, 0.057009051067798425, 0.06436361007498577}
	uAvgWant := sparse.ZerosDense(10, 2, 3)
	uAvgWant.Elements = []float64{5.25, 2.4999999999999996, 3.5655, 7.35, 4.36, 4.255, 5.85, 3.3, 3.9399999999999995, 8.15, 5.09, 4.64, 5.95, 3.7, 4.029999999999999, 8.1, 5.24, 4.6735, 5.949999999999999, 3.9000000000000004, 3.9949999999999997, 7.95, 5.375, 4.510000000000001, 5.95, 4.15, 3.885, 8.25, 6.05, 4.359999999999999, 7, 4.715, 3.88, 8.25, 6.05, 4.5249999999999995, 6.955, 6.05, 4.369999999999999, 7.45, 6.25, 4.05, 7.75, 7.6, 6, 7.25, 6.85, 6, 9.45, 8.4, 6.6, 9.05, 7.45, 6.85, 9.75, 8.45, 7.65, 9.55, 7.95, 7.3}
	vAvgWant := sparse.ZerosDense(10, 3, 2)
	vAvgWant.Elements = []float64{1.3, 0.825, 0.725, -0.1395, 1.535, 0.6497850000000001, 1.4169999999999998, 0.97, 1.02, 0.21500000000000002, 1.655, 0.61, 1.485, 1.205, 1.2349999999999999, 0.44999999999999996, 1.775, 0.625, 1.56, 1.29, 1.5350000000000001, 0.675, 1.915, 0.675, 1.96, 1.58, 2.075, 0.98, 2.1, 0.8800000000000001, 2.275, 2.605, 2.44, 1.5999999999999999, 1.7000000000000002, 1.06, 0.8, 2.23, 1.25, 0.8499999999999999, 0.9600000000000001, 0.29500000000000004, 2.25, 2.55, 1.95, 0.45, 1.18, -0.38, 2.65, 2.35, 2.3499999999999996, 1.2095, 1.47, 0.35, 1.75, 1.65, 1.5, 1.389, 1.2049999999999998, 0.9349999999999998}
	wAvgWant := sparse.ZerosDense(11, 2, 2)
	wAvgWant.Elements = []float64{-0.064, -0.07350000000000001, -0.132, -0.0733, -0.094, -0.106, -0.17149999999999999, -0.09275, -0.089, -0.12050000000000001, -0.172, -0.10350000000000001, -0.07500000000000001, -0.1263, -0.15785, -0.1014, -0.06, -0.128, -0.1465, -0.0912, -0.05, -0.127, -0.1185, -0.07300000000000001, -0.03499999999999999, -0.11299999999999999, -0.078, -0.036, -0.04, -0.0795, -0.037000000000000005, 0.006499999999999999, -0.05215, 0.01425, 0.0035000000000000014, 0.059500000000000004, -0.0185, 0.0995, 0.0534, 0.1128, 0.042499999999999996, 0.1535, 0.1173, 0.1492}

	arrayCompare(speed, speedWant, tolerance, "speed", t)
	arrayCompare(speedInverse, speedInverseWant, tolerance, "speedInverse", t)
	arrayCompare(speedMinusThird, speedMinusThirdWant, tolerance, "speedMinusThird", t)
	arrayCompare(speedMinusOnePointFour, speedMinusOnePointFourWant, tolerance, "speedMinusOnePointFour", t)
	arrayCompare(uAvg, uAvgWant, tolerance, "uAvg", t)
	arrayCompare(vAvg, vAvgWant, tolerance, "vAvg", t)
	arrayCompare(wAvg, wAvgWant, tolerance, "wAvg", t)
}

func TestTemperature(t *testing.T) {
	const tolerance = 1.0e-8
	tempFunc := wrfTemperatureConvert(testNextData(T), wrfPressureConvert(testNextData(P), testNextData(PB)))
	Temp, err := average(tempFunc)
	if err != nil {
		t.Error(err)
	}
	// Want values are taken from existing function output, just to avoid regression.
	TempWant := sparse.ZerosDense([]int{10, 2, 2}...)
	TempWant.Elements = []float64{279.808286167611, 282.77093912866354, 281.05933619833, 283.10148176991595, 278.90558942027724, 281.93887037798595, 280.1719711763049, 282.27694539848557, 278.03451473475184, 281.10060719333427, 279.3203375385492, 281.4463339873839, 277.1521520449787, 280.2138622644722, 278.4191440049893, 280.60953901975, 276.258261214384, 278.46139922055556, 276.67317947807453, 278.8744515464799, 274.8957773498135, 276.67693664094236, 275.37048717995606, 277.5965341996561, 273.56783559076655, 275.37425990858594, 275.42305570477583, 276.27980873583203, 271.98862752109744, 273.5771846460377, 272.4858597370849, 274.5202235464134, 269.9101027378091, 272.0081183292417, 271.14609288028373, 271.60681981346085, 268.3242695531737, 269.53336375045876, 268.53742776665933, 270.0273862217476}
	arrayCompare(Temp, TempWant, tolerance, "temperature", t)

}

func TestStabilityMixingChemistry(t *testing.T) {
	const tolerance = 1.0e-8
	layerHeights := geopotentialToHeight(PH[0], PHB[0])

	surfaceHeatFluxFunc := testNextData(HFX)
	hoFunc := testNextData(ho)
	h2o2Func := testNextData(h2o2)
	ustarFunc := testNextData(UST)
	pblhFunc := testNextData(PBLH)
	altFunc := testNextData(ALT)
	qCloudFunc := testNextData(QCLOUD)
	qrainFunc := testNextData(QRAIN)

	pFunc := wrfPressureConvert(testNextData(P), testNextData(PB))
	tempFunc := wrfTemperatureConvert(testNextData(T), wrfPressureConvert(testNextData(P), testNextData(PB)))

	radiationDownFunc := wrfRadiationDown(testNextData(SWDOWN), testNextData(GLW))

	z0Func := wrfZ0(testNextData(LUIndex))
	seinfeldLandUseFunc := wrfSeinfeldLandUse(testNextData(LUIndex))
	weselyLandUseFunc := wrfWeselyLandUse(testNextData(LUIndex))

	Sclass, S1, KzzUnstaggered, M2u, M2d, SO2oxidation, particleDryDep, SO2DryDep, NOxDryDep, NH3DryDep, VOCDryDep, Kyy, err := stabilityMixingChemistry(layerHeights, pblhFunc, ustarFunc, altFunc, tempFunc,
		pFunc, surfaceHeatFluxFunc, hoFunc, h2o2Func, z0Func, seinfeldLandUseFunc, weselyLandUseFunc, qCloudFunc, radiationDownFunc, qrainFunc)
	if err != nil {
		t.Fatal(err)
	}

	// Want values are taken from existing function output, just to avoid regression.
	SclassWant := sparse.ZerosDense([]int{10, 2, 2}...)
	SclassWant.Elements = []float64{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0.5, 0.5, 0.5, 0.5, 0.5, 0, 0.5, 1, 0.5, 1, 0.5, 0.5, 0.5, 0.5, 0.5, 0.5, 0.5, 0.5, 0, 0, 0, 0}
	S1Want := sparse.ZerosDense([]int{10, 2, 2}...)
	S1Want.Elements = []float64{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 8.659278305910272e-06, 0, 1.4578650516150517e-05, 1.0850944941134813e-05, 2.606688445139948e-05, 9.674267325912213e-06, 2.3053171991801797e-05, 8.25749448891299e-06, 2.433465611644154e-05, 2.315189932403436e-05, 1.561794382176757e-05, 2.7897378280835205e-05, 2.1687003285148284e-05, 2.1292824101886347e-05, 2.4259319806233757e-05, 1.7388338146444446e-05, 2.4554316171646257e-05, 1.9156943397541684e-05, 1.8685891088611138e-05, 1.7818692810858116e-05, 0, 0, 0, 0}
	KzzUnstaggeredWant := sparse.ZerosDense([]int{10, 2, 2}...)
	KzzUnstaggeredWant.Elements = []float64{2.485297294200668, 1.1340448458091898, 1.6639329618281913, 1.332171920586989, 5.3877057283586955, 2.340785854879271, 3.820439151898395, 3.0741603634637054, 5.025098099237127, 2.1197592664196927, 4.024342418942832, 3.320048145504126, 3.0678619129386795, 1.4223207453719247, 3.0734870511573096, 2.748042123865984, 2.44517224785958, 0.657088977103797, 2.6044600991017743, 1.8448742882747844, 3, 1.6477864890814842, 2.8988092768170937, 1.6302043943113804, 3, 3, 3, 2.4553125272751704, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3}
	M2uWant := sparse.ZerosDense([]int{10, 2, 2}...)
	M2uWant.Elements = []float64{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	M2dWant := sparse.ZerosDense([]int{10, 2, 2}...)
	M2dWant.Elements = []float64{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	SO2oxidationWant := sparse.ZerosDense([]int{10, 2, 2}...)
	SO2oxidationWant.Elements = []float64{1.721657225308307e-07, 2.81397237473639e-07, 2.407546336659953e-07, 2.61349340908674e-07, 1.657502677072367e-07, 2.6616347667235814e-07, 1.965803268391172e-07, 2.68991898227076e-07, 1.7287147453981165e-07, 2.3699094049610386e-07, 1.915248452652225e-07, 2.5193332436575475e-07, 1.8623149968362105e-07, 0.2528135843303613, 1.970548982458274e-07, 2.3530268601622454e-07, 2.926399319434474e-07, 2.1665156133438385e-07, 2.1729669290828353e-07, 2.0962361619205938e-07, 2.6956563786526554e-07, 2.093463026377419e-07, 3.626047816640662e-07, 1.6288422079864346e-07, 5.345276052707975e-07, 2.1937229945140494e-07, 8.698723439241161e-07, 1.5580791448153274e-07, 1.9863750471240534e-06, 3.599331457411426e-07, 5.001428803731995e-07, 6.339050075943203e-07, 4.831824472501523e-07, 4.477029865692818e-07, 4.622916727421153e-07, 3.603982632752272e-07, 2.0642162959544838e-07, 3.965027082274449e-07, 2.3196340932676016e-07, 3.6686577403379013e-07}
	particleDryDepWant := sparse.ZerosDense([]int{10, 2, 2}...)
	particleDryDepWant.Elements = []float64{0.010162865526722476, 0.0017065693477232012, 0.0023751725529831532, 0.0017312970049009032, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	SO2DryDepWant := sparse.ZerosDense([]int{10, 2, 2}...)
	SO2DryDepWant.Elements = []float64{0.0006568912613621955, 0.0015159703284781817, 0.0019091511896542302, 0.0013656110424365697, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	NOxDryDepWant := sparse.ZerosDense([]int{10, 2, 2}...)
	NOxDryDepWant.Elements = []float64{0.0005026924483343035, 0.0005932331015299358, 0.0007109444200385571, 0.0005316489167541452, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	NH3DryDepWant := sparse.ZerosDense([]int{10, 2, 2}...)
	NH3DryDepWant.Elements = []float64{0.0007134287459725018, 0.0005291881365846268, 0.0010781565788536144, 0.00047609893335600543, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	VOCDryDepWant := sparse.ZerosDense([]int{10, 2, 2}...)
	VOCDryDepWant.Elements = []float64{0.004007289236558869, 0.004335934482798151, 0.005862160692304825, 0.004142667516734282, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	KyyWant := sparse.ZerosDense([]int{10, 2, 2}...)
	KyyWant.Elements = []float64{3.314078249333397, 1.6963492952810897, 2.2043457151327797, 1.7755557553414636, 5.81661362886252, 2.4524110263991976, 4.127658827070609, 3.2998877771882564, 5.178982032133868, 2.163232562976917, 4.133745757981916, 3.4080629388484924, 3.0245935106815245, 1.4093207475370766, 3.0880710514206777, 2.764735389058494, 0.5030066407522406, 0.6084716233029834, 1.8236003202595226, 1.8249670827324578, 3, 0.07489190318322451, 1.8373335564537299, 0.8183546621141308, 3, 3, 3, 1.6039803699322372, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3}

	want := []*sparse.DenseArray{
		SclassWant, S1Want, KzzUnstaggeredWant, M2uWant, M2dWant, SO2oxidationWant,
		particleDryDepWant, SO2DryDepWant, NOxDryDepWant, NH3DryDepWant, VOCDryDepWant, KyyWant}

	for i, arr := range []*sparse.DenseArray{
		Sclass, S1, KzzUnstaggered, M2u, M2d, SO2oxidation,
		particleDryDep, SO2DryDep, NOxDryDep, NH3DryDep, VOCDryDep, Kyy} {
		arrayCompare(arr, want[i], tolerance, fmt.Sprintf("%d", i), t)
	}
}

func arrayCompare(have, want *sparse.DenseArray, tolerance float64, name string, t *testing.T) {
	if !reflect.DeepEqual(want.Shape, have.Shape) {
		t.Errorf("%s: want shape %v but have shape %v", name, want.Shape, have.Shape)
		return
	}
	for i, wantv := range want.Elements {
		havev := have.Elements[i]
		if math.IsNaN(havev) || math.IsInf(havev, 0) {
			t.Errorf("%s, element %d: is %g", name, i, havev)
		} else if math.IsNaN(wantv) || math.IsInf(wantv, 0) {
			t.Errorf("%s, golden data element %d: is %g", name, i, wantv)
		}
		if math.Abs(havev-wantv)/math.Abs(havev+wantv)*2 > tolerance {
			t.Errorf("%s, element %d: want %g but have %g", name, i, wantv, havev)
		}
	}
}

func TestGeosLayerConvert(t *testing.T) {
	convFull := geosLayerConvert(72)
	convChem := geosLayerConvert(47)

	a := sparse.ZerosDense(72, 1, 1)
	for i := range a.Elements {
		a.Elements[i] = float64(i)
	}
	b := sparse.ZerosDense(73, 1, 1)
	for i := range b.Elements {
		b.Elements[i] = float64(i)
	}
	c := sparse.ZerosDense(1, 1)
	c.Elements[0] = 6
	d := sparse.ZerosDense(1, 1, 1)
	d.Elements[0] = 12

	resultUnstaggered := sparse.ZerosDense(47, 1, 1)
	resultUnstaggered.Elements = []float64{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33, 34, 35, 36.5, 38.5, 40.5, 42.5, 45.5, 49.5, 53.5, 57.5, 61.5, 65.5, 69.5}

	resultStaggered := sparse.ZerosDense(48, 1, 1)
	resultStaggered.Elements = []float64{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33, 34, 35, 36, 38, 40, 42, 44, 48, 52, 56, 60, 64, 68, 72}

	tests := []struct {
		name      string
		f         func(NextData) NextData
		v, result *sparse.DenseArray
	}{
		{
			name:   "full",
			f:      convFull,
			v:      a,
			result: a,
		},
		{
			name:   "unstaggered",
			f:      convChem,
			v:      a,
			result: resultUnstaggered,
		},
		{
			name:   "staggered",
			f:      convChem,
			v:      b,
			result: resultStaggered,
		},
		{
			name:   "2d-c",
			f:      convChem,
			v:      c,
			result: c,
		},
		{
			name:   "2d-d",
			f:      convChem,
			v:      d,
			result: d,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := average(test.f(testNextData([]*sparse.DenseArray{test.v, test.v})))
			if err != nil {
				t.Fatal(err)
			}
			arrayCompare(result, test.result, 1.0e-8, test.name, t)
		})
	}
}

func TestStagger(t *testing.T) {
	a := sparse.ZerosDense(2, 2, 2)
	a.Elements = []float64{
		0, 1,
		2, 3,
		4, 5,
		6, 7,
	}

	k := sparse.ZerosDense(3, 2, 2)
	k.Elements = []float64{
		0, 1, 2, 3,
		2, 3, 4, 5,
		4, 5, 6, 7,
	}

	j := sparse.ZerosDense(2, 3, 2)
	j.Elements = []float64{
		0, 1,
		1, 2,
		2, 3,
		4, 5,
		5, 6,
		6, 7,
	}

	i := sparse.ZerosDense(2, 2, 3)
	i.Elements = []float64{
		0, 0.5, 1,
		2, 2.5, 3,
		4, 4.5, 5,
		6, 6.5, 7,
	}

	want := []*sparse.DenseArray{k, j, i}

	for dim := 0; dim < 3; dim++ {
		result := staggerWorker(a, dim)
		arrayCompare(result, want[dim], 1.0e-8, fmt.Sprintf("dim %d", dim), t)
	}
}

func TestReadApBp(t *testing.T) {
	f := nextDataConstantNCF("ap", "cmd/inmap/testdata/preproc/GEOSFP.ApBp.nc")
	dataWant := sparse.ZerosDense(73)
	dataWant.Elements = []float64{0, 0.04804826155304909, 6.593751907348633,
		13.13479995727539, 19.613109588623047, 26.092010498046875, 32.57080841064453,
		38.98200988769531, 45.33900833129883, 51.696109771728516, 58.0532112121582,
		64.36264038085938, 70.62197875976562, 78.83422088623047, 89.09992218017578,
		99.3652114868164, 109.18170166015625, 118.95860290527344, 128.69590759277344,
		142.91000366210938, 156.25999450683594, 169.60899353027344, 181.61900329589844,
		193.0970001220703, 203.25900268554688, 212.14999389648438, 218.7760009765625,
		223.8979949951172, 224.36300659179688, 216.86500549316406, 201.19200134277344,
		176.92999267578125, 150.39300537109375, 127.83699798583984, 108.66300201416016,
		92.36572265625, 78.5123062133789, 66.60340881347656, 56.387908935546875,
		47.6439094543457, 40.175411224365234, 33.81000900268555, 28.367809295654297,
		23.730409622192383, 19.79159927368164, 16.45709991455078, 13.643400192260742,
		11.276900291442871, 9.29294204711914, 7.619842052459717, 6.216801166534424,
		5.0468010902404785, 4.076570987701416, 3.276431083679199, 2.620210886001587,
		2.084969997406006, 1.6507899761199951, 1.300510048866272, 1.0194400548934937,
		0.7951341271400452, 0.616779088973999, 0.4758060872554779, 0.3650411069393158,
		0.27852609753608704, 0.2113489955663681, 0.15949499607086182, 0.11970300227403641,
		0.08934502303600311, 0.06600000709295273, 0.04758501052856445, 0.03269999846816063,
		0.019999999552965164, 0.009999999776482582,
	}
	data, err := f()
	if err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(data, dataWant) {
		t.Errorf("1: %v != %v", data, dataWant)
	}

	data, err = f()
	if err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(data, dataWant) {
		t.Errorf("2: %v != %v", data, dataWant)
	}
}
