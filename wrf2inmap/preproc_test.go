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

package main

import (
	"flag"
	"math"
	"os"
	"reflect"
	"testing"

	"github.com/spatialmodel/inmap"

	"bitbucket.org/ctessum/sparse"
)

const evalDataEnv = "evaldata"

func TestWRF2InMAP(t *testing.T) {
	const tolerance = 2.0 // TODO: The preprocessor gives different results
	// every time. This needs to be fixed.

	err := flag.Set("config", "configExample.json")
	if err != nil {
		t.Fatal(err)
	}
	main()

	cfg := inmap.VarGridConfig{}

	f1, err := os.Open("testdata/inmapData.ncf")
	if err != nil {
		t.Fatalf("opening new file: %v", err)
	}
	newData, err := cfg.LoadCTMData(f1)
	if err != nil {
		t.Fatalf("reading new file: %v", err)
	}
	f2, err := os.Open("testdata/inmapData_golden.ncf")
	if err != nil {
		t.Fatalf("opening golden file: %v", err)
	}
	goldenData, err := cfg.LoadCTMData(f2)
	if err != nil {
		t.Fatalf("reading golden file: %v", err)
	}
	compareCTMData(goldenData, newData, tolerance, t)
}

func compareCTMData(ctmdata, ctmdata2 *inmap.CTMData, tolerance float64, t *testing.T) {
	if len(ctmdata.Data) != len(ctmdata2.Data) {
		t.Fatalf("new and old ctmdata have different number of variables (%d vs. %d)",
			len(ctmdata2.Data), len(ctmdata.Data))
	}
	for name, dd1 := range ctmdata.Data {
		if _, ok := ctmdata2.Data[name]; !ok {
			t.Errorf("ctmdata2 doesn't have variable %s", name)
			continue
		}
		dd2 := ctmdata2.Data[name]
		if !reflect.DeepEqual(dd1.Dims, dd2.Dims) {
			t.Errorf("%s dims problem: %v != %v", name, dd1.Dims, dd2.Dims)
		}
		if dd1.Description != dd2.Description {
			t.Errorf("%s description problem: %s != %s", name, dd1.Description, dd2.Description)
		}
		if dd1.Units != dd2.Units {
			t.Errorf("%s units problem: %s != %s", name, dd1.Units, dd2.Units)
		}
		if !reflect.DeepEqual(dd1.Data.Shape, dd2.Data.Shape) {
			t.Errorf("%s data shape problem: %v != %v", name, dd1.Data.Shape, dd2.Data.Shape)
		}
		if arrayDifferent(dd1.Data, dd2.Data, tolerance) {
			t.Errorf("%s data problem: %v != %v", name, dd1.Data.Elements, dd2.Data.Elements)
		}
	}
}

func TestCalcPartitioning(t *testing.T) {
	numTsteps = 2
	const tolerance = 1.0e-8

	gasChan := make(chan *sparse.DenseArray)
	particleChan := make(chan *sparse.DenseArray)
	go calcPartitioning(gasChan, particleChan)
	gasChan <- so2[0]
	particleChan <- sulf[0]
	gasChan <- so2[1]
	particleChan <- sulf[1]
	gasChan <- nil
	partitioning := <-gasChan
	gasResult := <-gasChan
	particleResult := <-gasChan

	gasWant := sparse.ZerosDense(10, 2, 2)
	gasWant.Elements = []float64{1.305e-05, 3.415e-05, 2.4999999999999998e-05, 6.85e-05, 1.2775e-05, 3.075e-05, 1.76e-05, 5.4499999999999997e-05, 1.2145e-05, 2.89e-05, 1.3949999999999999e-05, 4.7000000000000004e-05, 1.1565e-05, 2.79e-05, 1.164e-05, 4.2e-05, 1.3555e-06, 2.605e-05, 1.000355e-05, 3.79e-05, 3.85e-08, 2.0255000000000002e-05, 3.615e-07, 3.4512e-05, 1.12e-09, 5.17e-07, 6.0499999999999996e-09, 3.3507e-05, 1.855e-09, 2.5e-09, 3.305e-09, 3.65095e-06, 1.80165e-06, 1.55075e-06, 1.949e-06, 1.6006499999999999e-06, 2.025e-06, 2.314e-06, 2.3699999999999998e-06, 2.645e-06}
	if arrayDifferent(gasResult, gasWant, tolerance) {
		t.Errorf("gas: want %v but have %v", gasWant, gasResult)
	}

	particleWant := sparse.ZerosDense(10, 2, 2)
	particleWant.Elements = []float64{2.15e-05, 2.25e-05, 2.2499999999999998e-05, 2.35e-05, 2.2e-05, 2.3e-05, 2.2499999999999998e-05, 2.35e-05, 2.2e-05, 2.3e-05, 2.2499999999999998e-05, 2.35e-05, 2.2e-05, 2.3e-05, 2.2499999999999998e-05, 2.35e-05, 2.2499999999999998e-05, 2.3e-05, 2.3e-05, 2.4e-05, 2.3e-05, 2.3500000000000002e-05, 2.3500000000000002e-05, 2.4e-05, 2.2499999999999998e-05, 2.4e-05, 2.3e-05, 2.4999999999999998e-05, 2.2499999999999998e-05, 2.3e-05, 2.3500000000000002e-05, 2.3e-05, 3.45e-05, 2.7e-05, 3.35e-05, 2.7e-05, 4.6e-05, 3.65e-05, 4.4e-05, 3.65e-05}
	if arrayDifferent(particleResult, particleWant, tolerance) {
		t.Errorf("particle: want %#v but have %#v", particleWant, particleResult)
	}

	partitioningWant := sparse.ZerosDense(10, 2, 2)
	partitioningWant.Elements = []float64{0.20930232558139536, 0.125, 0.16964285714285715, 0.08750000000000001, 0.20930232558139536, 0.12987012987012989, 0.19387755102040818, 0.09722222222222222, 0.21428571428571427, 0.13333333333333333, 0.21111111111111114, 0.10294117647058823, 0.21951219512195122, 0.13513513513513514, 0.2261904761904762, 0.10714285714285714, 0.9347826086956521, 0.14084507042253522, 0.2435897435897436, 0.11052631578947368, 0.9978977649922549, 0.16666666666666666, 0.9819888381532218, 0.11666666666666667, 0.9999122339365392, 0.9761904761904762, 0.9993530690641208, 0.12359550561797754, 0.9998614168586635, 0.999887234991758, 0.9997255680087973, 0.8557312252964426, 0.7870222750769289, 0.4467353951890034, 0.6748447147643264, 0.9432624113475178, 0.8820058997050148, 0.7811263902461725, 0.8502055271130794, 0.7681937338109415}
	if arrayDifferent(partitioning, partitioningWant, tolerance) {
		t.Errorf("partitioning: want %#v but have %#v", partitioningWant, partitioning)
	}
}

func TestAverage(t *testing.T) {
	numTsteps = 2
	const tolerance = 1.0e-8

	dataChan := make(chan *sparse.DenseArray)
	go average(dataChan)
	dataChan <- PHB[0]
	dataChan <- PHB[1]
	dataChan <- nil
	result := <-dataChan
	want := PHB[0].Copy()
	want.AddDense(PHB[1])
	want.Scale(0.5)
	if arrayDifferent(result, want, tolerance) {
		t.Errorf("want %#v but have %#v", want, result)
	}
}

func TestCalcLayerHeights(t *testing.T) {
	const tolerance = 1.0e-8

	layerHeights, dz := calcLayerHeights(PH[0], PHB[0])

	heightsWant := sparse.ZerosDense(11, 2, 2)
	heightsWant.Elements = []float64{0, 0, 0, 0, 60.57114305088894, 60.693508996446276, 50.455558218147914, 50.59831848796478, 141.23069549744307, 131.33944823155716, 131.13550498896157, 131.54339147415274, 241.7747140970668, 242.28457220355577, 231.8834668311809, 242.79443031004473, 362.3051704710579, 373.21613394992175, 362.7130569562491, 373.92993529900633, 563.4951792916032, 534.025380736541, 482.63168360245345, 535.045096949519, 763.7674435204683, 714.1072639484432, 785.1814839930048, 746.0243814146523, 963.6318212641422, 1015.6373481260166, 985.0458617366787, 996.262740079436, 1465.332198049283, 1415.3661036133644, 1384.7746172240265, 1395.991495566784, 1868.1201021755646, 1918.086196611483, 1888.5144264351231, 1898.7115885649025, 2375.938776238573, 2423.8654382485356, 2394.2936680721755, 2403.471113988977}
	if arrayDifferent(layerHeights, heightsWant, tolerance) {
		t.Errorf("layerHeights: want %#v but have %#v", heightsWant, layerHeights)
	}

	dzWant := sparse.ZerosDense(10, 2, 2)
	dzWant.Elements = []float64{60.57114305088894, 60.693508996446276, 50.455558218147914, 50.59831848796478, 80.65955244655413, 70.64593923511089, 80.67994677081366, 80.94507298618797, 100.54401859962374, 110.94512397199861, 100.74796184221933, 111.25103883589199, 120.53045637399111, 130.931561746366, 130.8295901250682, 131.1355049889616, 201.1900088205453, 160.80924678661927, 119.91862664620436, 161.11516165051268, 200.27226422886508, 180.0818832119022, 302.54980039055135, 210.97928446513333, 199.8643777436739, 301.53008417757337, 199.8643777436739, 250.23835866478362, 501.70037678514086, 399.7287554873478, 399.7287554873478, 399.72875548734794, 402.78790412628155, 502.7200929981186, 503.7398092110966, 502.7200929981186, 507.81867406300853, 505.77924163705256, 505.77924163705234, 504.75952542407435}
	if arrayDifferent(dz, dzWant, tolerance) {
		t.Errorf("dz: want %#v but have %#v", dzWant, dz)
	}
}

func TestCalcWetDeposition(t *testing.T) {
	layerHeights, _ := calcLayerHeights(PH[0], PHB[0])
	qrainChan := make(chan *sparse.DenseArray)
	cloudFracChan := make(chan *sparse.DenseArray)
	altChan := make(chan *sparse.DenseArray)
	go calcWetDeposition(layerHeights, qrainChan, cloudFracChan, altChan)
	qrainChan <- QRAIN[0]
	cloudFracChan <- QCLOUD[0]
	altChan <- ALT[0]
	qrainChan <- QRAIN[1]
	cloudFracChan <- QCLOUD[0]
	altChan <- ALT[0]
	qrainChan <- nil
	wdParticle := <-qrainChan
	wdSO2 := <-qrainChan
	wdOtherGas := <-qrainChan

	// Want values are taken from existing function output, just to avoid regression.
	wdParticleWant := sparse.ZerosDense(10, 2, 2)
	wdParticleWant.Elements = []float64{5.977011494252873e-06, 0, 3.3255813953488373e-06, 0, 6.5000000000000004e-06, 0, 3.885057471264367e-06, 0, 7.386363636363637e-06, 0, 4.183908045977011e-06, 0, 8.179775280898877e-06, 0, 4.727272727272727e-06, 5.744186046511629e-08, 8.285714285714285e-06, 1.8022727272727276e-07, 5.258426966292135e-06, 2.988505747126436e-07, 8.478260869565217e-06, 4.0898876404494386e-07, 5.428571428571429e-06, 5.842696629213484e-07, 6.085106382978724e-06, 4.857142857142856e-07, 6.430107526881721e-06, 7.142857142857143e-07, 6.367346938775511e-06, 2.271578947368421e-07, 7.237113402061857e-06, 4.702127659574468e-07, 0, 0, 0, 0, 0, 0, 0, 0}
	if arrayDifferent(wdParticle, wdParticleWant, tolerance) {
		t.Errorf("wdParticle: want %#v but have %#v", wdParticleWant, wdParticle)
	}
	wdSO2Want := sparse.ZerosDense(10, 2, 2)
	wdSO2Want.Elements = []float64{1.4232337745268778e-10, 0, 9.50640880271098e-11, 0, 1.1622925884955751e-10, 0, 6.945280085248075e-11, 0, 1.0595766152268119e-10, 0, 5.989682221136394e-11, 0, 9.788192266012663e-11, 0, 5.2114982640119044e-11, 6.317803650222431e-13, 5.939931868688835e-11, 1.6164691211016316e-12, 6.324513443017658e-11, 2.6753191837625487e-12, 6.105818455016383e-11, 3.275663436700725e-12, 2.5878947266080734e-11, 3.9942150308730816e-12, 4.3912756458966567e-11, 2.3233152753570407e-12, 4.640243375576037e-11, 4.1169538743652434e-12, 1.8305095818815327e-11, 8.196347509398497e-13, 2.6113068391016206e-11, 1.696629226823708e-12, 0, 0, 0, 0, 0, 0, 0, 0}
	if arrayDifferent(wdSO2, wdSO2Want, tolerance) {
		t.Errorf("wdSO2: want %#v but have %#v", wdSO2Want, wdSO2)
	}
	wdOtherGasWant := sparse.ZerosDense(10, 2, 2)
	wdOtherGasWant.Elements = []float64{4.74411258175626e-10, 0, 3.1688029342369936e-10, 0, 3.8743086283185844e-10, 0, 2.3150933617493587e-10, 0, 3.5319220507560394e-10, 0, 1.996560740378798e-10, 0, 3.2627307553375545e-10, 0, 1.7371660880039682e-10, 2.1059345500741436e-12, 1.9799772895629453e-10, 5.388230403672106e-12, 2.1081711476725523e-10, 8.917730612541829e-12, 2.0352728183387943e-10, 1.091887812233575e-11, 8.626315755360243e-11, 1.3314050102910271e-11, 1.463758548632219e-10, 7.744384251190135e-12, 1.546747791858679e-10, 1.3723179581217477e-11, 6.101698606271776e-11, 2.7321158364661655e-12, 8.704356130338736e-11, 5.6554307560790266e-12, 0, 0, 0, 0, 0, 0, 0, 0}
	if arrayDifferent(wdOtherGas, wdOtherGasWant, tolerance) {
		t.Errorf("wdOtherGas: want %#v but have %#v", wdOtherGasWant, wdOtherGas)
	}
}

func TestWindDeviation(t *testing.T) {
	numTsteps = 2
	const tolerance = 1.0e-8
	uChan := make(chan *sparse.DenseArray)
	go average(uChan)
	uChan <- U[0]
	uChan <- U[1]
	uChan <- nil
	uAvg := <-uChan
	go windDeviation(uAvg, uChan)
	uChan <- U[0]
	uChan <- U[1]
	uChan <- nil
	uDev := <-uChan
	uDevWant := sparse.ZerosDense(10, 2, 3)
	uDevWant.Elements = []float64{3.4499999999999997, 5.699999999999999, 3.6345, 2.3499999999999996, 4.54, 3.8449999999999998, 3.8499999999999996, 6.199999999999999, 4.26, 2.85, 4.91, 4.46, 3.8500000000000005, 6.2, 4.57, 2.9, 4.76, 4.7265, 3.7499999999999996, 5.9, 4.705, 3.05, 4.625, 4.790000000000001, 4.050000000000001, 5.35, 4.615, 3.75, 4.949999999999999, 4.84, 5, 5.1850000000000005, 4.22, 4.75, 4.949999999999999, 5.074999999999999, 6.045, 4.949999999999999, 3.8299999999999996, 5.550000000000001, 4.75, 5.35, 6.25, 4.4, 3.5, 5.75, 4.15, 3.4000000000000004, 4.55, 4.6, 3.4, 4.95, 3.55, 2.8499999999999996, 3.25, 3.55, 3.35, 3.45, 3.05, 2.7}
	if arrayDifferent(uDev, uDevWant, tolerance) {
		t.Errorf("want %#v but have %#v", uDevWant, uDev)
	}
}

func TestWindSpeed(t *testing.T) {
	uChan := make(chan *sparse.DenseArray)
	vChan := make(chan *sparse.DenseArray)
	wChan := make(chan *sparse.DenseArray)
	go windSpeed(uChan, vChan, wChan)
	uChan <- U[0]
	vChan <- V[0]
	wChan <- W[0]
	uChan <- U[1]
	vChan <- V[1]
	wChan <- W[1]
	uChan <- nil
	vChan <- nil
	wChan <- nil

	speed := <-uChan
	speedInverse := <-uChan
	speedMinusThird := <-uChan
	speedMinusOnePointFour := <-uChan
	uAvg := <-uChan
	vAvg := <-vChan
	wAvg := <-wChan

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

	if arrayDifferent(speed, speedWant, tolerance) {
		t.Errorf("speed: want %#v but have %#v", speedWant, speed)
	}
	if arrayDifferent(speedInverse, speedInverseWant, tolerance) {
		t.Errorf("speedInverse: want %#v but have %#v", speedInverseWant, speedInverse)
	}
	if arrayDifferent(speedMinusThird, speedMinusThirdWant, tolerance) {
		t.Errorf("speedMinusThird: want %#v but have %#v", speedMinusThirdWant, speedMinusThird)
	}
	if arrayDifferent(speedMinusOnePointFour, speedMinusOnePointFourWant, tolerance) {
		t.Errorf("speedMinusOnePointFour: want %#v but have %#v", speedMinusOnePointFourWant, speedMinusOnePointFour)
	}
	if arrayDifferent(uAvg, uAvgWant, tolerance) {
		t.Errorf("uAvg: want %#v but have %#v", uAvgWant, uAvg)
	}
	if arrayDifferent(vAvg, vAvgWant, tolerance) {
		t.Errorf("vAvg: want %#v but have %#v", vAvgWant, vAvg)
	}
	if arrayDifferent(wAvg, wAvgWant, tolerance) {
		t.Errorf("wAvg: want %#v but have %#v", wAvgWant, wAvg)
	}
}

func TestStabilityMixingChemistry(t *testing.T) {
	const tolerance = 1.0e-8
	numTsteps = 2
	layerHeights, _ := calcLayerHeights(PH[0], PHB[0])

	pblhChan := make(chan *sparse.DenseArray)
	ustarChan := make(chan *sparse.DenseArray)
	altChan := make(chan *sparse.DenseArray)
	Tchan := make(chan *sparse.DenseArray)
	PBchan := make(chan *sparse.DenseArray)
	Pchan := make(chan *sparse.DenseArray)
	surfaceHeatFluxChan := make(chan *sparse.DenseArray)
	hoChan := make(chan *sparse.DenseArray)
	h2o2Chan := make(chan *sparse.DenseArray)
	luIndexChan := make(chan *sparse.DenseArray)
	qCloudChan := make(chan *sparse.DenseArray)
	swDownChan := make(chan *sparse.DenseArray)
	glwChan := make(chan *sparse.DenseArray)
	qrainChan := make(chan *sparse.DenseArray)

	go StabilityMixingChemistry(layerHeights, pblhChan, ustarChan, altChan, Tchan,
		PBchan, Pchan, surfaceHeatFluxChan, hoChan, h2o2Chan, luIndexChan, qCloudChan, swDownChan, glwChan, qrainChan)

	Tchan <- T[0]
	PBchan <- PB[0]
	Pchan <- P[0]
	surfaceHeatFluxChan <- HFX[0]
	hoChan <- ho[0]
	h2o2Chan <- h2o2[0]
	luIndexChan <- LU_INDEX[0]
	ustarChan <- UST[0]
	pblhChan <- PBLH[0]
	altChan <- ALT[0]
	qCloudChan <- QCLOUD[0]
	swDownChan <- SWDOWN[0]
	glwChan <- GLW[0]
	qrainChan <- QRAIN[0]
	Tchan <- nil

	Temp := <-Tchan
	Sclass := <-Tchan
	S1 := <-Tchan
	KzzUnstaggered := <-Tchan
	M2u := <-Tchan
	M2d := <-Tchan
	SO2oxidation := <-Tchan
	particleDryDep := <-Tchan
	SO2DryDep := <-Tchan
	NOxDryDep := <-Tchan
	NH3DryDep := <-Tchan
	VOCDryDep := <-Tchan
	Kyy := <-Tchan

	// Want values are taken from existing function output, just to avoid regression.
	TempWant := sparse.ZerosDense([]int{10, 2, 2}...)
	TempWant.Elements = []float64{140.04125654073547, 141.77620056107904, 140.445497645103, 141.69594973493577, 139.60983418765272, 141.35794410278606, 139.9799361812147, 141.28220683090106, 139.17068260199952, 140.93656581868726, 139.55330791167367, 140.86540771218404, 138.7236332790822, 140.51200920992426, 139.12338877236473, 140.445497645103, 138.26851064234478, 139.60983418765272, 138.24466071217805, 139.59611782739063, 137.83898390963805, 138.73243367492665, 137.8230261570994, 139.17504766994244, 137.3976105608935, 138.30773406413087, 138.33852599736096, 138.77253980914966, 136.32788114429707, 137.40695961616467, 136.82143640358532, 137.8909312345815, 134.6641258240994, 136.12466668716573, 135.54498722332644, 135.95724674563974, 133.38488247333925, 134.29202874522622, 133.65061421309764, 134.54632072212553}
	SclassWant := sparse.ZerosDense([]int{10, 2, 2}...)
	SclassWant.Elements = []float64{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0.5, 0.5, 0.5, 0.5, 0.5, 0, 0, 0.5, 0, 0.5, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	S1Want := sparse.ZerosDense([]int{10, 2, 2}...)
	S1Want.Elements = []float64{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 8.659278305910272e-06, 0, 1.4578650516150517e-05, 1.0850944941134813e-05, 1.7337509193256397e-05, 9.674267325912213e-06, 1.727477866285235e-05, 8.25749448891299e-06, 6.9012315270936046e-06, 1.1515342595723896e-05, 6.901231527093604e-06, 1.3875659354342116e-05, 6.168842469889617e-06, 6.038577586206899e-06, 6.452105644527409e-06, 4.313269704433497e-06, 7.636454484888867e-06, 5.119711304870799e-06, 5.093615690717622e-06, 4.443182214229452e-06, 0, 0, 0, 0}
	KzzUnstaggeredWant := sparse.ZerosDense([]int{10, 2, 2}...)
	KzzUnstaggeredWant.Elements = []float64{2.454483897942035, 1.1141180228867564, 1.6106040457723587, 1.3216233282700538, 5.33460361555517, 2.304771712397139, 3.7302121104979435, 3.0552555443941665, 4.989764959167504, 2.093064605459659, 3.9684292060571718, 3.3060639985254765, 3.049696545874661, 1.4061627212384793, 3.0500969139779537, 2.7393318748287205, 1.690051304320292, 0.6499871570518773, 1.85008504946346, 1.8408000427976865, 1.5, 0.8962353517626742, 1.3988092768170937, 0.8792124776454997, 1.5, 1.5, 1.5, 0.9553125272751706, 1.5, 1.5, 1.5, 1.5, 1.5, 1.5, 1.5, 1.5, 1.5, 1.5, 1.5, 1.5}
	M2uWant := sparse.ZerosDense([]int{10, 2, 2}...)
	M2uWant.Elements = []float64{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	M2dWant := sparse.ZerosDense([]int{10, 2, 2}...)
	M2dWant.Elements = []float64{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	SO2oxidationWant := sparse.ZerosDense([]int{10, 2, 2}...)
	SO2oxidationWant.Elements = []float64{1.0277819949068232e-07, 2.1620553322597986e-07, 1.4700212206278526e-07, 1.9649881524635097e-07, 9.826294095099549e-08, 2.0538412059613606e-07, 1.1176013997967447e-07, 2.0861396380353674e-07, 1.0297464271840184e-07, 1.8034823495072285e-07, 1.0758832545859143e-07, 1.945943091995075e-07, 1.0946074235633741e-07, 0.2528135305881805, 1.1075962796735504e-07, 1.8092568869265878e-07, 1.280624730773278e-07, 1.6561169823201485e-07, 1.207848836968412e-07, 1.5685594709021174e-07, 1.160932970223121e-07, 1.532361982162077e-07, 1.1773683999214064e-07, 1.0804544250863865e-07, 1.1327825708771841e-07, 1.1736011512578788e-07, 1.0166830122565492e-07, 9.359909244748123e-08, 1.1800545880132146e-07, 9.4477041178819e-08, 1.0923542037283018e-07, 8.724356669474563e-08, 1.259977463773005e-07, 9.72009858923421e-08, 1.1560632648328172e-07, 9.365721750084438e-08, 9.518241445456828e-08, 1.0688427887292786e-07, 9.500164758092751e-08, 9.699928370299036e-08}
	particleDryDepWant := sparse.ZerosDense([]int{10, 2, 2}...)
	particleDryDepWant.Elements = []float64{0.00970722450479153, 0.0014495399149542074, 0.001932090281189961, 0.0015726221224152172, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	SO2DryDepWant := sparse.ZerosDense([]int{10, 2, 2}...)
	SO2DryDepWant.Elements = []float64{0.0003524927891196722, 0.0011202878614203435, 0.0011527597725163327, 0.0011248998144100868, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	NOxDryDepWant := sparse.ZerosDense([]int{10, 2, 2}...)
	NOxDryDepWant.Elements = []float64{0.00026007610320734933, 0.00036449321729952876, 0.00036766039908993646, 0.0003649553498378682, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	NH3DryDepWant := sparse.ZerosDense([]int{10, 2, 2}...)
	NH3DryDepWant.Elements = []float64{0.00014614568297576156, 0.00031874623910327127, 0.00032116056888837604, 0.0003190554297104977, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	VOCDryDepWant := sparse.ZerosDense([]int{10, 2, 2}...)
	VOCDryDepWant.Elements = []float64{0.0029273729275386953, 0.0038071817325971073, 0.004199026376285014, 0.0038582119063847167, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	KyyWant := sparse.ZerosDense([]int{10, 2, 2}...)
	KyyWant.Elements = []float64{3.2477573098203343, 1.6538686820553758, 2.0928389153326115, 1.7534139107693807, 5.763530068552916, 2.416387667237476, 4.037108658598404, 3.280978071554933, 5.144208178595066, 2.136791051259152, 4.07915010647612, 3.394201670481238, 3.007323491182678, 1.3935613240104898, 3.0672195935515694, 2.75621259016805, 0.5004418624270893, 0.6019842974870082, 1.8214095043449887, 1.8211805783462849, 1.5, 0.07411597819387362, 0.3373335564537299, 0.8178585189984321, 1.5, 1.5, 1.5, 0.10398036993223725, 1.5, 1.5, 1.5, 1.5, 1.5, 1.5, 1.5, 1.5, 1.5, 1.5, 1.5, 1.5}

	want := []*sparse.DenseArray{TempWant,
		SclassWant, S1Want, KzzUnstaggeredWant, M2uWant, M2dWant, SO2oxidationWant,
		particleDryDepWant, SO2DryDepWant, NOxDryDepWant, NH3DryDepWant, VOCDryDepWant, KyyWant}

	for i, arr := range []*sparse.DenseArray{Temp,
		Sclass, S1, KzzUnstaggered, M2u, M2d, SO2oxidation,
		particleDryDep, SO2DryDep, NOxDryDep, NH3DryDep, VOCDryDep, Kyy} {

		if arrayDifferent(arr, want[i], tolerance) {
			t.Errorf("uAvg: want %#v but have %#v", arr, want[i])
		}
	}
}

func arrayDifferent(a, b *sparse.DenseArray, tolerance float64) bool {
	if !reflect.DeepEqual(a.Shape, b.Shape) {
		return true
	}
	for i, av := range a.Elements {
		bv := b.Elements[i]
		if math.Abs(av-bv)/math.Abs(av+bv)*2 > tolerance {
			return true
		}
	}
	return false
}
