/*
Copyright © 2017 the InMAP authors.
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
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ctessum/atmos/acm2"
	"github.com/ctessum/atmos/emep"
	"github.com/ctessum/atmos/seinfeld"
	"github.com/ctessum/atmos/wesely1989"
	"github.com/spatialmodel/inmap"

	"bitbucket.org/ctessum/cdf"
	"bitbucket.org/ctessum/sparse"
)

// physical constants
const (
	MWa      = 28.97   // g/mol, molar mass of air
	mwN      = 14.0067 // g/mol, molar mass of nitrogen
	mwS      = 32.0655 // g/mol, molar mass of sulfur
	mwNH4    = 18.03851
	mwSO4    = 96.0632
	mwNO3    = 62.00501
	g        = 9.80665 // m/s2
	κ        = 0.41    // Von Kármán constant
	atmPerPa = 9.86923267e-6
)

// NextData is a type of function that returns data for the next time step.
// If there are no more time steps, it should return the io.EOF error.
type NextData func() (*sparse.DenseArray, error)

// Preprocessor specifies the methods that are necessary for a
// variable to act as a preprocessor for InMAP inputs.
type Preprocessor interface {
	PBLH() NextData
	PH() NextData
	PHB() NextData
	U() NextData
	V() NextData
	W() NextData
	AVOC() NextData
	BVOC() NextData
	ASOA() NextData
	BSOA() NextData
	NOx() NextData
	PNO() NextData
	SOx() NextData
	PS() NextData
	NH3() NextData
	PNH() NextData
	TotalPM25() NextData
	ALT() NextData
	QRain() NextData
	QCloud() NextData
	CloudFrac() NextData
	UStar() NextData
	T() NextData
	PB() NextData
	P() NextData
	SurfaceHeatFlux() NextData
	HO() NextData
	H2O2() NextData
	LUIndex() NextData
	SWDown() NextData
	GLW() NextData
}

// Preprocess returns preprocessed InMAP input data
// based on the information available from the given
// preprocessor.
func Preprocess(p Preprocessor, config *ConfigInfo) error {
	var pblh, ph, phb, windSpeed, windSpeedInverse, windSpeedMinusThird, windSpeedMinusOnePointFour, uAvg, vAvg, wAvg *sparse.DenseArray

	errChan := make(chan error)

	go func() {
		var err error
		pblh, err = average(p.PBLH())
		errChan <- err
	}()

	go func() {
		var err error
		ph, err = average(p.PH())
		errChan <- err
	}()
	go func() {
		var err error
		phb, err = average(p.PHB())
		errChan <- err
	}()

	go func() {
		var err error
		windSpeed, windSpeedInverse, windSpeedMinusThird, windSpeedMinusOnePointFour, uAvg, vAvg, wAvg, err = calcWindSpeed(p.U(), p.V(), p.W())
		errChan <- err
	}()

	for i := 0; i < 4; i++ {
		err := <-errChan
		if err != nil {
			return err
		}
	}

	layerHeights, Dz := calcLayerHeights(ph, phb)

	var uDeviation, vDeviation, aOrgPartitioning, aVOC, aSOA, bOrgPartitioning, bVOC, bSOA,
		NOPartitioning, gNO, pNO, SPartitioning, gS, pS, NHPartitioning, gNH, pNH, totalpm25,
		alt, particleWetDep, SO2WetDep, otherGasWetDep, temperature, Sclass, S1, Kzz, M2u, M2d, SO2oxidation, particleDryDep, SO2DryDep,
		NOxDryDep, NH3DryDep, VOCDryDep, Kxxyy *sparse.DenseArray

	go func() {
		var err error
		// calculate deviation from average wind speed.
		// Only calculate horizontal deviations.
		uDeviation, err = windDeviation(uAvg, p.U())
		errChan <- err
	}()
	go func() {
		var err error
		vDeviation, err = windDeviation(vAvg, p.V())
		errChan <- err
	}()

	go func() {
		var err error
		// calculate gas/particle partitioning
		aOrgPartitioning, aVOC, aSOA, err = marginalPartitioning(p.AVOC(), p.ASOA())
		errChan <- err
	}()
	go func() {
		var err error
		bOrgPartitioning, bVOC, bSOA, err = marginalPartitioning(p.BVOC(), p.BSOA())
		errChan <- err
	}()
	go func() {
		var err error
		NOPartitioning, gNO, pNO, err = marginalPartitioning(p.NOx(), p.PNO())
		errChan <- err
	}()
	go func() {
		var err error
		SPartitioning, gS, pS, err = marginalPartitioning(p.SOx(), p.PS())
		errChan <- err
	}()
	go func() {
		var err error
		NHPartitioning, gNH, pNH, err = marginalPartitioning(p.NH3(), p.PNH())
		errChan <- err
	}()

	go func() {
		var err error
		// Get total PM2.5 averages for performance eval.
		totalpm25, err = average(p.TotalPM25())
		errChan <- err
	}()

	go func() {
		var err error
		// average inverse density
		alt, err = average(p.ALT())
		errChan <- err
	}()

	go func() {
		var err error
		// Calculate wet deposition.
		particleWetDep, SO2WetDep, otherGasWetDep, err = wetDeposition(Dz, p.QRain(), p.CloudFrac(), p.ALT())
		errChan <- err
	}()

	go func() {
		var err error
		// Calculate stability for plume rise, vertical mixing,
		// and chemical reaction rates.
		temperature, Sclass, S1, Kzz, M2u, M2d, SO2oxidation, particleDryDep, SO2DryDep,
			NOxDryDep, NH3DryDep, VOCDryDep, Kxxyy, err = stabilityMixingChemistry(layerHeights, p.PBLH(),
			p.UStar(), p.ALT(), p.T(), p.PB(), p.P(), p.SurfaceHeatFlux(), p.HO(), p.H2O2(),
			p.LUIndex(), p.QCloud(), p.SWDown(), p.GLW(), p.QRain())
		errChan <- err
	}()

	for i := 0; i < 11; i++ {
		err := <-errChan
		if err != nil {
			return err
		}
	}

	// write out data to file
	outputFile := filepath.Join(config.OutputDir, config.OutputFilePrefix+".ncf")
	fmt.Printf("Writing out data to %v...\n", outputFile)

	data := new(inmap.CTMData)
	data.AddVariable("UAvg", []string{"z", "y", "xStagger"},
		"Annual average x velocity", "m/s", uAvg)
	data.AddVariable("VAvg", []string{"z", "yStagger", "x"},
		"Annual average y velocity", "m/s", vAvg)
	data.AddVariable("WAvg", []string{"zStagger", "y", "x"},
		"Annual average z velocity", "m/s", wAvg)
	data.AddVariable("UDeviation", []string{"z", "y", "xStagger"},
		"Average deviation from average x velocity", "m/s", uDeviation)
	data.AddVariable("VDeviation", []string{"z", "yStagger", "x"},
		"Average deviation from average y velocity", "m/s", vDeviation)
	data.AddVariable("aOrgPartitioning", []string{"z", "y", "x"},
		"Mass fraction of anthropogenic organic matter in particle {vs. gas} phase",
		"fraction", aOrgPartitioning)
	data.AddVariable("aVOC", []string{"z", "y", "x"},
		"Average anthropogenic VOC concentration", "ug m-3", aVOC)
	data.AddVariable("aSOA", []string{"z", "y", "x"},
		"Average anthropogenic secondary organic aerosol concentration", "ug m-3", aSOA)
	data.AddVariable("bOrgPartitioning", []string{"z", "y", "x"},
		"Mass fraction of biogenic organic matter in particle {vs. gas} phase",
		"fraction", bOrgPartitioning)
	data.AddVariable("bVOC", []string{"z", "y", "x"},
		"Average biogenic VOC concentration", "ug m-3", bVOC)
	data.AddVariable("bSOA", []string{"z", "y", "x"},
		"Average biogenic secondary organic aerosol concentration", "ug m-3", bSOA)
	data.AddVariable("NOPartitioning", []string{"z", "y", "x"},
		"Mass fraction of N from NOx in particle {vs. gas} phase", "fraction",
		NOPartitioning)
	data.AddVariable("gNO", []string{"z", "y", "x"},
		"Average concentration of nitrogen fraction of gaseous NOx", "ug m-3",
		gNO)
	data.AddVariable("pNO", []string{"z", "y", "x"},
		"Average concentration of nitrogen fraction of particulate NO3",
		"ug m-3", pNO)
	data.AddVariable("SPartitioning", []string{"z", "y", "x"},
		"Mass fraction of S from SOx in particle {vs. gas} phase", "fraction",
		SPartitioning)
	data.AddVariable("gS", []string{"z", "y", "x"},
		"Average concentration of sulfur fraction of gaseous SOx", "ug m-3",
		gS)
	data.AddVariable("pS", []string{"z", "y", "x"},
		"Average concentration of sulfur fraction of particulate sulfate",
		"ug m-3", pS)
	data.AddVariable("NHPartitioning", []string{"z", "y", "x"},
		"Mass fraction of N from NH3 in particle {vs. gas} phase", "fraction",
		NHPartitioning)
	data.AddVariable("gNH", []string{"z", "y", "x"},
		"Average concentration of nitrogen fraction of gaseous ammonia",
		"ug m-3", gNH)
	data.AddVariable("pNH", []string{"z", "y", "x"},
		"Average concentration of nitrogen fraction of particulate ammonium",
		"ug m-3", pNH)
	data.AddVariable("SO2oxidation", []string{"z", "y", "x"},
		"Rate of SO2 oxidation to SO4 by hydroxyl radical and H2O2",
		"s-1", SO2oxidation)
	data.AddVariable("ParticleDryDep", []string{"z", "y", "x"},
		"Dry deposition velocity for particles", "m s-1", particleDryDep)
	data.AddVariable("SO2DryDep", []string{"z", "y", "x"},
		"Dry deposition velocity for SO2", "m s-1", SO2DryDep)
	data.AddVariable("NOxDryDep", []string{"z", "y", "x"},
		"Dry deposition velocity for NOx", "m s-1", NOxDryDep)
	data.AddVariable("NH3DryDep", []string{"z", "y", "x"},
		"Dry deposition velocity for NH3", "m s-1", NH3DryDep)
	data.AddVariable("VOCDryDep", []string{"z", "y", "x"},
		"Dry deposition velocity for VOCs", "m s-1", VOCDryDep)
	data.AddVariable("Kxxyy", []string{"z", "y", "x"},
		"Horizontal eddy diffusion coefficient", "m2 s-1", Kxxyy)
	data.AddVariable("LayerHeights", []string{"zStagger", "y", "x"},
		"Height at edge of layer", "m", layerHeights)
	data.AddVariable("Dz", []string{"z", "y", "x"},
		"Vertical grid size", "m", Dz)
	data.AddVariable("ParticleWetDep", []string{"z", "y", "x"},
		"Wet deposition rate constant for fine particles",
		"s-1", particleWetDep)
	data.AddVariable("SO2WetDep", []string{"z", "y", "x"},
		"Wet deposition rate constant for SO2 gas", "s-1", SO2WetDep)
	data.AddVariable("OtherGasWetDep", []string{"z", "y", "x"},
		"Wet deposition rate constant for other gases", "s-1", otherGasWetDep)
	data.AddVariable("Kzz", []string{"zStagger", "y", "x"},
		"Vertical turbulent diffusivity", "m2 s-1", Kzz)
	data.AddVariable("M2u", []string{"z", "y", "x"},
		"ACM2 nonlocal upward mixing {Pleim 2007}", "s-1", M2u)
	data.AddVariable("M2d", []string{"z", "y", "x"},
		"ACM2 nonlocal downward mixing {Pleim 2007}", "s-1", M2d)
	data.AddVariable("Pblh", []string{"y", "x"},
		"Planetary boundary layer height", "m", pblh)
	data.AddVariable("WindSpeed", []string{"z", "y", "x"},
		"RMS wind speed", "m s-1", windSpeed)
	data.AddVariable("WindSpeedInverse", []string{"z", "y", "x"},
		"RMS wind speed^(-1)", "(m s-1)^(-1)", windSpeedInverse)
	data.AddVariable("WindSpeedMinusThird", []string{"z", "y", "x"},
		"RMS wind speed^(-1/3)", "(m s-1)^(-1/3)", windSpeedMinusThird)
	data.AddVariable("WindSpeedMinusOnePointFour", []string{"z", "y", "x"},
		"RMS wind speed^(-1.4)", "(m s-1)^(-1.4)", windSpeedMinusOnePointFour)
	data.AddVariable("Temperature", []string{"z", "y", "x"},
		"Average Temperature", "K", temperature)
	data.AddVariable("S1", []string{"z", "y", "x"},
		"Stability parameter", "?", S1)
	data.AddVariable("Sclass", []string{"z", "y", "x"},
		"Stability parameter", "0=Unstable; 1=Stable", Sclass)
	data.AddVariable("alt", []string{"z", "y", "x"},
		"Inverse density", "m3 kg-1", alt)
	data.AddVariable("TotalPM25", []string{"z", "y", "x"},
		"Total PM2.5 concentration", "ug m-3", totalpm25)

	ff, err := os.Create(outputFile)
	if err != nil {
		panic(err)
	}
	data.Write(ff, config.CtmGridXo, config.CtmGridYo, config.CtmGridDx, config.CtmGridDy)
	ff.Close()
	return nil
}

// marginalPartitioning calculates marginal partitioning over a period
// of time between gas and particle
// phase of a chemical compound or group of compounds as defined by the
// equation f = Δp / (Δp + Δg), where f is the fraction in particle phase,
// Δp is the change in particle phase concentration between one time step
// and the next, and Δg is the change in gas phase concentration from
// one time step to the next. The fraction is forced to be
// between zero and one. Both gas phase and particle phase concentration
// should be in units of [mass/volume].
func marginalPartitioning(gasFunc, particleFunc NextData) (gasConc, particleConc, partitioning *sparse.DenseArray, err error) {
	var gas, particle, oldgas, oldparticle *sparse.DenseArray
	firstData := true
	var n int
	for {
		gasdata, err := gasFunc()
		if err != nil {
			if err == io.EOF {
				// Divide the arrays by the total number of timesteps and return.
				return arrayAverage(gas, n), arrayAverage(particle, n), arrayAverage(partitioning, n), nil
			}
			return nil, nil, nil, err
		}
		particledata, err := particleFunc()
		if err != nil {
			return nil, nil, nil, err
		}
		if firstData {
			partitioning = sparse.ZerosDense(gasdata.Shape...)
			gas = sparse.ZerosDense(gasdata.Shape...)
			particle = sparse.ZerosDense(gasdata.Shape...)
			oldgas = sparse.ZerosDense(gasdata.Shape...)
			oldparticle = sparse.ZerosDense(gasdata.Shape...)
			firstData = false
		}
		gas.AddDense(gasdata)
		particle.AddDense(particledata)

		for i, particleval := range particledata.Elements {
			particlechange := particleval - oldparticle.Elements[i]
			totalchange := particlechange + (gasdata.Elements[i] - oldgas.Elements[i])
			// Calculate the marginal partitioning coefficient, which is the
			// change in particle concentration divided by the change in overall
			// concentration. Force the coefficient to be between zero and
			// one.
			part := math.Min(math.Max(particlechange/totalchange, 0), 1)
			if !math.IsNaN(part) {
				partitioning.Elements[i] += part
			}
		}
		oldgas = gasdata.Copy()
		oldparticle = particledata.Copy()
		n++
	}
}

// average calculates the arithmatic mean of a
// set of arrays.
func average(dataFunc NextData) (*sparse.DenseArray, error) {
	var avgdata *sparse.DenseArray
	firstData := true
	var n int
	for {
		data, err := dataFunc()
		if err != nil {
			if err == io.EOF {
				return arrayAverage(avgdata, n), nil
			}
			return nil, err
		}
		if firstData {
			avgdata = sparse.ZerosDense(data.Shape...)
			firstData = false
		}
		avgdata.AddDense(data)
		n++
	}
}

// calcLayerHeights calculates the heights above the ground
// of the layers (in meters).
// For more information, refer to
// http://www.openwfm.org/wiki/How_to_interpret_WRF_variables
func calcLayerHeights(ph, phb *sparse.DenseArray) (layerHeights, Dz *sparse.DenseArray) {
	layerHeights = sparse.ZerosDense(ph.Shape...)
	Dz = sparse.ZerosDense(ph.Shape[0]-1, ph.Shape[1], ph.Shape[2])
	for k := 0; k < ph.Shape[0]; k++ {
		for j := 0; j < ph.Shape[1]; j++ {
			for i := 0; i < ph.Shape[2]; i++ {
				h := (ph.Get(k, j, i) + phb.Get(k, j, i) -
					ph.Get(0, j, i) - phb.Get(0, j, i)) / g // m
				layerHeights.Set(h, k, j, i)
				if k > 0 {
					Dz.Set(h-layerHeights.Get(k-1, j, i), k-1, j, i)
				}
			}
		}
	}
	return
}

// wetDeposition calculates wet deposition based on layer heights,
// mass fraction of rain in the grid cells, fraction of the grid cells
// filled with clouds, and inverse density.
func wetDeposition(Δz *sparse.DenseArray, qrainFunc, cloudFracFunc, altFunc NextData) (wdParticle, wdSO2, wdOtherGas *sparse.DenseArray, err error) {
	firstData := true
	var n int
	for {
		qrain, err := qrainFunc() // mass frac
		if err != nil {
			if err == io.EOF {
				return arrayAverage(wdParticle, n), arrayAverage(wdSO2, n), arrayAverage(wdOtherGas, n), nil
			}
			return nil, nil, nil, err
		}
		cloudFrac, err := cloudFracFunc() // frac
		if err != nil {
			return nil, nil, nil, err
		}
		alt, err := altFunc() // m3/kg
		if err != nil {
			return nil, nil, nil, err
		}
		if firstData {
			wdParticle = sparse.ZerosDense(qrain.Shape...) // units = 1/s
			wdSO2 = sparse.ZerosDense(qrain.Shape...)      // units = 1/s
			wdOtherGas = sparse.ZerosDense(qrain.Shape...) // units = 1/s
			firstData = false
		}
		for i := 0; i < len(qrain.Elements); i++ {
			wdp, wds, wdo := emep.WetDeposition(cloudFrac.Elements[i],
				qrain.Elements[i], 1/alt.Elements[i], Δz.Elements[i])
			wdParticle.Elements[i] += wdp
			wdSO2.Elements[i] += wds
			wdOtherGas.Elements[i] += wdo
		}
		n++
	}
}

// windDeviation calculates the average absolute deviation of the wind velocity.
// Output is based on a staggered grid.
func windDeviation(uAvg *sparse.DenseArray, uFunc NextData) (*sparse.DenseArray, error) {
	var uDeviation *sparse.DenseArray
	var n int
	firstData := true
	for {
		u, err := uFunc()
		if err != nil {
			if err == io.EOF {
				return arrayAverage(uDeviation, n), nil
			}
			return nil, err
		}
		if firstData {
			uDeviation = sparse.ZerosDense(u.Shape...)
			firstData = false
		}
		for i, uV := range u.Elements {
			avgV := uAvg.Elements[i]
			uDeviation.Elements[i] += math.Abs(uV - avgV)
		}
		n++
	}
}

// calcWindSpeed calculates RMS wind speed as well as average speeds in each
// direction.
func calcWindSpeed(uFunc, vFunc, wFunc NextData) (speed, speedInverse, speedMinusThird, speedMinusOnePointFour, uAvg, vAvg, wAvg *sparse.DenseArray, err error) {
	var n int
	firstData := true
	var dims []int
	for {
		u, err := uFunc()
		if err != nil {
			if err == io.EOF {
				return arrayAverage(speed, n), arrayAverage(speedInverse, n), arrayAverage(speedMinusThird, n),
					arrayAverage(speedMinusOnePointFour, n), arrayAverage(uAvg, n), arrayAverage(vAvg, n), arrayAverage(wAvg, n), nil
			}
			return nil, nil, nil, nil, nil, nil, nil, err
		}
		v, err := vFunc()
		if err != nil {
			return nil, nil, nil, nil, nil, nil, nil, err
		}
		w, err := wFunc()
		if err != nil {
			return nil, nil, nil, nil, nil, nil, nil, err
		}

		if firstData {
			uAvg = sparse.ZerosDense(u.Shape...)
			vAvg = sparse.ZerosDense(v.Shape...)
			wAvg = sparse.ZerosDense(w.Shape...)
			// get unstaggered grid sizes
			dims = make([]int, len(u.Shape))
			for i, ulen := range u.Shape {
				vlen := v.Shape[i]
				wlen := w.Shape[i]
				dims[i] = minInt(ulen, vlen, wlen)
			}
			speed = sparse.ZerosDense(dims...)
			speedInverse = sparse.ZerosDense(dims...)
			speedMinusThird = sparse.ZerosDense(dims...)
			speedMinusOnePointFour = sparse.ZerosDense(dims...)
			firstData = false
		}
		uAvg.AddDense(u)
		vAvg.AddDense(v)
		wAvg.AddDense(w)
		for k := 0; k < dims[0]; k++ {
			for j := 0; j < dims[1]; j++ {
				for i := 0; i < dims[2]; i++ {
					ucenter := (math.Abs(u.Get(k, j, i)) +
						math.Abs(u.Get(k, j, i+1))) / 2.
					vcenter := (math.Abs(v.Get(k, j, i)) +
						math.Abs(v.Get(k, j+1, i))) / 2.
					wcenter := (math.Abs(w.Get(k, j, i)) +
						math.Abs(w.Get(k+1, j, i))) / 2.
					s := math.Pow(math.Pow(ucenter, 2.)+
						math.Pow(vcenter, 2.)+math.Pow(wcenter, 2.), 0.5)
					speed.AddVal(s, k, j, i)
					speedInverse.AddVal(1./s, k, j, i)
					speedMinusThird.AddVal(math.Pow(s, -1./3.), k, j, i)
					speedMinusOnePointFour.AddVal(math.Pow(s, -1.4), k, j, i)
				}
			}
		}
	}
}

func minInt(vals ...int) int {
	minval := vals[0]
	for _, val := range vals {
		if val < minval {
			minval = val
		}
	}
	return minval
}

// USGSz0 holds Roughness lengths for USGS land classes ([m]), from WRF file
// VEGPARM.TBL.
var USGSz0 = []float64{.50, .1, .06, .1, 0.095, .20, .11,
	.03, .035, .15, .50, .50, .50, .50, .35, 0.0001, .20, .40,
	.01, .10, .30, .15, .075, 0.001, .01, .15, .01}

// USGSseinfeld lookup table to go from USGS land classes to land classes for
// particle dry deposition.
var USGSseinfeld = []seinfeld.LandUseCategory{
	seinfeld.Desert,    //'Urban and Built-Up Land'
	seinfeld.Grass,     //'Dryland Cropland and Pasture'
	seinfeld.Grass,     //'Irrigated Cropland and Pasture'
	seinfeld.Grass,     //'Mixed Dryland/Irrigated Cropland and Pasture'
	seinfeld.Grass,     //'Cropland/Grassland Mosaic'
	seinfeld.Grass,     //'Cropland/Woodland Mosaic'
	seinfeld.Grass,     //'Grassland'
	seinfeld.Shrubs,    //'Shrubland'
	seinfeld.Shrubs,    //'Mixed Shrubland/Grassland'
	seinfeld.Grass,     //'Savanna'
	seinfeld.Deciduous, //'Deciduous Broadleaf Forest'
	seinfeld.Evergreen, //'Deciduous Needleleaf Forest'
	seinfeld.Deciduous, //'Evergreen Broadleaf Forest'
	seinfeld.Evergreen, //'Evergreen Needleleaf Forest'
	seinfeld.Deciduous, //'Mixed Forest'
	seinfeld.Desert,    //'Water Bodies'
	seinfeld.Grass,     //'Herbaceous Wetland'
	seinfeld.Deciduous, //'Wooded Wetland'
	seinfeld.Desert,    //'Barren or Sparsely Vegetated'
	seinfeld.Shrubs,    //'Herbaceous Tundra'
	seinfeld.Deciduous, //'Wooded Tundra'
	seinfeld.Shrubs,    //'Mixed Tundra'
	seinfeld.Desert,    //'Bare Ground Tundra'
	seinfeld.Desert,    //'Snow or Ice'
	seinfeld.Desert,    //'Playa'
	seinfeld.Desert,    //'Lava'
	seinfeld.Desert}    //'White Sand'

// USGSwesely lookup table to go from USGS land classes to land classes for
// gas dry deposition.
var USGSwesely = []wesely1989.LandUseCategory{
	wesely1989.Urban,        //'Urban and Built-Up Land'
	wesely1989.RangeAg,      //'Dryland Cropland and Pasture'
	wesely1989.RangeAg,      //'Irrigated Cropland and Pasture'
	wesely1989.RangeAg,      //'Mixed Dryland/Irrigated Cropland and Pasture'
	wesely1989.RangeAg,      //'Cropland/Grassland Mosaic'
	wesely1989.Agricultural, //'Cropland/Woodland Mosaic'
	wesely1989.Range,        //'Grassland'
	wesely1989.RockyShrubs,  //'Shrubland'
	wesely1989.RangeAg,      //'Mixed Shrubland/Grassland'
	wesely1989.Range,        //'Savanna'
	wesely1989.Deciduous,    //'Deciduous Broadleaf Forest'
	wesely1989.Coniferous,   //'Deciduous Needleleaf Forest'
	wesely1989.Deciduous,    //'Evergreen Broadleaf Forest'
	wesely1989.Coniferous,   //'Evergreen Needleleaf Forest'
	wesely1989.MixedForest,  //'Mixed Forest'
	wesely1989.Water,        //'Water Bodies'
	wesely1989.Wetland,      //'Herbaceous Wetland'
	wesely1989.Wetland,      //'Wooded Wetland'
	wesely1989.Barren,       //'Barren or Sparsely Vegetated'
	wesely1989.RockyShrubs,  //'Herbaceous Tundra'
	wesely1989.MixedForest,  //'Wooded Tundra'
	wesely1989.RockyShrubs,  //'Mixed Tundra'
	wesely1989.Barren,       //'Bare Ground Tundra'
	wesely1989.Barren,       //'Snow or Ice'
	wesely1989.Barren,       //'Playa'
	wesely1989.Barren,       //'Lava'
	wesely1989.Barren}       //'White Sand'

// stabilityMixingChemistry calculates:
// 1) Stability parameters for use in plume rise calculation (ASME, 1973,
// as described in Seinfeld and Pandis, 2006).
// 2) Vertical turbulent diffusivity using a middling value (1 m2/s)
// from Wilson (2004) for grid cells above the planetary boundary layer
// and Pleim (2007) for grid cells within the planetary
// boundary layer.
// 3) SO2 oxidation to SO4 by HO (Stockwell 1997).
// 4) Dry deposition velocity (gocart and Seinfed and Pandis (2006)).
// 5) Horizontal eddy diffusion coefficient (Kyy, [m2/s]) assumed to be the
// same as vertical eddy diffusivity.
//
// Inputs are layer heights (m), friction velocity (ustar, m/s),
// planetary boundary layer height (pblh, m), inverse density (m3/kg),
// perturbation potential temperature (Temp,K), Pressure (Pb and P, Pa),
// surface heat flux (W/m2), HO mixing ratio (ppmv), and USGS land use index
// (luIndex).
func stabilityMixingChemistry(LayerHeights *sparse.DenseArray, pblhFunc, ustarFunc, altFunc, TFunc, PBFunc, PFunc, surfaceHeatFluxFunc, hoFunc, h2o2Func, luIndexFunc,
	qCloudFunc, swDownFunc, glwFunc, qrainFunc NextData) (Temp, Sclass, S1, KzzUnstaggered, M2u, M2d, SO2oxidation, particleDryDep, SO2DryDep, NOxDryDep, NH3DryDep, VOCDryDep, Kyy *sparse.DenseArray, err error) {
	const (
		po    = 101300. // Pa, reference pressure
		kappa = 0.2854  // related to von karman's constant
		Cp    = 1006.   // m2/s2-K; specific heat of air
	)

	var Kzz *sparse.DenseArray
	var n int
	firstData := true
	for {
		T, err := TFunc() // K
		if err != nil {
			if err == io.EOF { // done reading data: return results
				// Check for mass balance in convection coefficients
				for k := 0; k < M2u.Shape[0]-2; k++ {
					for j := 0; j < M2u.Shape[1]; j++ {
						for i := 0; i < M2u.Shape[2]; i++ {
							z := LayerHeights.Get(k, j, i)
							zabove := LayerHeights.Get(k+1, j, i)
							z2above := LayerHeights.Get(k+2, j, i)
							Δzratio := (z2above - zabove) / (zabove - z)
							m2u := M2u.Get(k, j, i)
							val := m2u - M2d.Get(k, j, i) +
								M2d.Get(k+1, j, i)*Δzratio
							if math.Abs(val/m2u) > 1.e-8 {
								panic(fmt.Errorf("M2u and M2d don't match: "+
									"(k,j,i)=(%v,%v,%v); val=%v; m2u=%v; "+
									"m2d=%v, m2dAbove=%v",
									k, j, i, val, m2u, M2d.Get(k, j, i),
									M2d.Get(k+1, j, i)))
							}
						}
					}
				}
				// convert Kzz to unstaggered grid
				KzzUnstaggered := sparse.ZerosDense(Temp.Shape...)
				for j := 0; j < KzzUnstaggered.Shape[1]; j++ {
					for i := 0; i < KzzUnstaggered.Shape[2]; i++ {
						for k := 0; k < KzzUnstaggered.Shape[0]; k++ {
							KzzUnstaggered.Set(
								(Kzz.Get(k, j, i)+Kzz.Get(k+1, j, i))/2.,
								k, j, i)
						}
					}
				}
				return arrayAverage(Temp, n), arrayAverage(Sclass, n), arrayAverage(S1, n),
					arrayAverage(KzzUnstaggered, n), arrayAverage(M2u, n), arrayAverage(M2d, n),
					arrayAverage(SO2oxidation, n), arrayAverage(particleDryDep, n),
					arrayAverage(SO2DryDep, n), arrayAverage(NOxDryDep, n), arrayAverage(NH3DryDep, n),
					arrayAverage(VOCDryDep, n), arrayAverage(Kyy, n), nil
			}
			return nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, err
		}
		PB, err := PBFunc() // Pa
		if err != nil {
			return nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, err
		}
		P, err := PFunc() // Pa
		if err != nil {
			return nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, err
		}
		hfx, err := surfaceHeatFluxFunc() // W/m2
		if err != nil {
			return nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, err
		}
		ho, err := hoFunc() // ppmv
		if err != nil {
			return nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, err
		}
		h2o2, err := h2o2Func() // ppmv
		if err != nil {
			return nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, err
		}
		luIndex, err := luIndexFunc() // land use index
		if err != nil {
			return nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, err
		}
		ustar, err := ustarFunc() // friction velocity (m/s)
		if err != nil {
			return nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, err
		}
		pblh, err := pblhFunc() // current boundary layer height (m)
		if err != nil {
			return nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, err
		}
		alt, err := altFunc() // inverse density (m3/kg)
		if err != nil {
			return nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, err
		}
		qCloud, err := qCloudFunc() // cloud water mixing ratio (kg/kg)
		if err != nil {
			return nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, err
		}
		swDown, err := swDownFunc() // Downwelling short wave at ground level (W/m2)
		if err != nil {
			return nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, err
		}
		glw, err := glwFunc() // Downwelling long wave at ground level (W/m2)
		if err != nil {
			return nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, err
		}
		qrain, err := qrainFunc() // mass fraction rain
		if err != nil {
			return nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, err
		}
		if firstData {
			Temp = sparse.ZerosDense(T.Shape...) // units = K
			S1 = sparse.ZerosDense(T.Shape...)
			Sclass = sparse.ZerosDense(T.Shape...)
			Kzz = sparse.ZerosDense(LayerHeights.Shape...) // units = m2/s
			M2u = sparse.ZerosDense(T.Shape...)            // units = 1/s
			M2d = sparse.ZerosDense(T.Shape...)            // units = 1/s
			SO2oxidation = sparse.ZerosDense(T.Shape...)   // units = 1/s
			particleDryDep = sparse.ZerosDense(T.Shape...) // units = m/s
			SO2DryDep = sparse.ZerosDense(T.Shape...)      // units = m/s
			NOxDryDep = sparse.ZerosDense(T.Shape...)      // units = m/s
			NH3DryDep = sparse.ZerosDense(T.Shape...)      // units = m/s
			VOCDryDep = sparse.ZerosDense(T.Shape...)      // units = m/s
			Kyy = sparse.ZerosDense(T.Shape...)            // units = m2/s
			firstData = false
		}
		type empty struct{}
		sem := make(chan empty, T.Shape[1]) // semaphore pattern
		for j := 0; j < T.Shape[1]; j++ {
			go func(j int) { // concurrent processing
				for i := 0; i < T.Shape[2]; i++ {
					// Get Layer index of PBL top (staggered)
					var pblTop int
					for k := 0; k < LayerHeights.Shape[0]; k++ {
						if LayerHeights.Get(k, j, i) >= pblh.Get(j, i) {
							pblTop = k
							break
						}
					}
					// Calculate boundary layer average temperature (K)
					To := 0.
					for k := 0; k < LayerHeights.Shape[0]; k++ {
						if k == pblTop {
							To /= float64(k)
							break
						}
						To += T.Get(k, j, i) + 300.
					}
					// Calculate convective mixing rate
					u := ustar.Get(j, i) // friction velocity
					h := LayerHeights.Get(pblTop, j, i)
					hflux := hfx.Get(j, i)                // heat flux [W m-2]
					ρ := 1 / alt.Get(0, j, i)             // density [kg/m3]
					L := acm2.ObukhovLen(hflux, ρ, To, u) // Monin-Obukhov length [m]
					fconv := acm2.ConvectiveFraction(L, h)
					m2u := acm2.M2u(LayerHeights.Get(1, j, i),
						LayerHeights.Get(2, j, i), h, L, u, fconv)

					// Calculate dry deposition
					p := (P.Get(0, j, i) + PB.Get(0, j, i)) // Pressure [Pa]
					//z: [m] surface layer; assumed to be 10% of boundary layer.
					z := h / 10.
					// z: [m] surface layer; assumed to be top of first model layer.
					//z := LayerHeights.Get(1, j, i)
					lu := f2i(luIndex.Get(j, i))
					//gocartObk := gocart.ObhukovLen(hflux, ρ, To, u)
					zo := USGSz0[lu]         // roughness length [m]
					const dParticle = 0.3e-6 // [m], Seinfeld & Pandis fig 8.11
					const ρparticle = 1830.  // [kg/m3] Jacobson (2005) Ex. 13.5
					const Θsurface = 0.      // surface slope [rad]; Assume surface is flat.

					// This is not the best way to tell what season it is.
					var iSeasonP seinfeld.SeasonalCategory // for particles
					var iSeasonG wesely1989.SeasonCategory // for gases
					switch {
					case To > 273.+20.:
						iSeasonP = seinfeld.Midsummer
						iSeasonG = wesely1989.Midsummer
					case To <= 273.+20 && To > 273.+10.:
						iSeasonP = seinfeld.Autumn
						iSeasonG = wesely1989.Autumn
					case To <= 273.+10 && To > 273.+0.:
						iSeasonP = seinfeld.LateAutumn
						iSeasonG = wesely1989.LateAutumn
					default:
						iSeasonP = seinfeld.Winter
						iSeasonG = wesely1989.Winter
					}
					const dew = false // don't know if there's dew.
					rain := qrain.Get(0, j, i) > 1.e-6

					G := swDown.Get(j, i) + glw.Get(j, i) // irradiation [W/m2]
					particleDryDep.AddVal(
						//gocart.ParticleDryDep(gocartObk, u, To, h,
						//	zo, dParticle/2., ρparticle, p), 0, j, i)
						seinfeld.DryDepParticle(z, zo, u, L, dParticle,
							To, p, ρparticle,
							ρ, iSeasonP, USGSseinfeld[lu]), 0, j, i)
					SO2DryDep.AddVal(
						seinfeld.DryDepGas(z, zo, u, L, To, ρ,
							G, Θsurface,
							wesely1989.So2Data, iSeasonG,
							USGSwesely[lu], rain, dew, true, false), 0, j, i)
					NOxDryDep.AddVal(
						seinfeld.DryDepGas(z, zo, u, L, To, ρ,
							G, Θsurface,
							wesely1989.No2Data, iSeasonG,
							USGSwesely[lu], rain, dew, false, false), 0, j, i)
					NH3DryDep.AddVal(
						seinfeld.DryDepGas(z, zo, u, L, To, ρ,
							G, Θsurface,
							wesely1989.Nh3Data, iSeasonG,
							USGSwesely[lu], rain, dew, false, false), 0, j, i)
					VOCDryDep.AddVal(
						seinfeld.DryDepGas(z, zo, u, L, To, ρ,
							G, Θsurface,
							wesely1989.OraData, iSeasonG,
							USGSwesely[lu], rain, dew, false, false), 0, j, i)

					for k := 0; k < T.Shape[0]; k++ {
						Tval := T.Get(k, j, i)
						var dthetaDz = 0. // potential temperature gradient
						if k < T.Shape[0]-1 {
							dthetaDz = (T.Get(k+1, j, i) - Tval) /
								(LayerHeights.Get(k+1, j, i) -
									LayerHeights.Get(k, j, i)) // K/m
						}

						p := P.Get(k, j, i) + PB.Get(k, j, i) // Pa
						pressureCorrection := math.Pow(p/po, kappa)

						// potential temperature, K
						θ := Tval + 300.
						// Ambient temperature, K
						t := θ * pressureCorrection
						Temp.AddVal(t, k, j, i)

						// Stability parameter
						s1 := dthetaDz / t * pressureCorrection
						S1.AddVal(s1, k, j, i)

						// Stability class
						if dthetaDz < 0.005 {
							Sclass.AddVal(0., k, j, i)
						} else {
							Sclass.AddVal(1., k, j, i)
						}

						// Mixing
						z := LayerHeights.Get(k, j, i)
						zabove := LayerHeights.Get(k+1, j, i)
						zcenter := (LayerHeights.Get(k, j, i) +
							LayerHeights.Get(k+1, j, i)) / 2
						Δz := zabove - z

						const freeAtmKzz = 3. // [m2 s-1]
						if k >= pblTop {      // free atmosphere (unstaggered grid)
							Kzz.AddVal(freeAtmKzz, k, j, i)
							Kyy.AddVal(freeAtmKzz, k, j, i)
							if k == T.Shape[0]-1 { // Top Layer
								Kzz.AddVal(freeAtmKzz, k+1, j, i)
							}
						} else { // Boundary layer (unstaggered grid)
							Kzz.AddVal(acm2.Kzz(z, h, L, u, fconv), k, j, i)
							M2d.AddVal(acm2.M2d(m2u, z, Δz, h), k, j, i)
							M2u.AddVal(m2u, k, j, i)
							kmyy := acm2.CalculateKm(zcenter, h, L, u)
							Kyy.AddVal(kmyy, k, j, i)
						}

						// Gas phase sulfur chemistry
						const Na = 6.02214129e23 // molec./mol (Avogadro's constant)
						const cm3perm3 = 100. * 100. * 100.
						const molarMassAir = 28.97 / 1000.             // kg/mol
						const airFactor = molarMassAir / Na * cm3perm3 // kg/molec.* cm3/m3
						M := 1. / (alt.Get(k, j, i) * airFactor)       // molec. air / cm3
						hoConc := ho.Get(k, j, i) * 1.e-6 * M          // molec. HO / cm3
						// SO2 oxidation rate (Stockwell 1997, Table 2d)
						const kinf = 1.5e-12
						ko := 3.e-31 * math.Pow(t/300., -3.3)
						SO2rate := (ko * M / (1 + ko*M/kinf)) * math.Pow(0.6,
							1./(1+math.Pow(math.Log10(ko*M/kinf), 2.))) // cm3/molec/s
						kso2 := SO2rate * hoConc

						// Aqueous phase sulfur chemistry
						qCloudVal := qCloud.Get(k, j, i)
						if qCloudVal > 0. {
							const pH = 3.5 // doesn't really matter for SO2
							qCloudVal /=
								alt.Get(k, j, i) * 1000. // convert to volume frac.
							kso2 += seinfeld.SulfurH2O2aqueousOxidationRate(
								h2o2.Get(k, j, i)*1000., pH, t, p*atmPerPa,
								qCloudVal)
						}
						SO2oxidation.AddVal(kso2, k, j, i) // 1/s
					}

					// Check for mass balance in convection coefficients
					for k := 0; k < M2u.Shape[0]-2; k++ {
						z := LayerHeights.Get(k, j, i)
						zabove := LayerHeights.Get(k+1, j, i)
						z2above := LayerHeights.Get(k+2, j, i)
						Δzratio := (z2above - zabove) / (zabove - z)
						m2u := M2u.Get(k, j, i)
						val := m2u - M2d.Get(k, j, i) +
							M2d.Get(k+1, j, i)*Δzratio
						if math.Abs(val/m2u) > 1.e-8 {
							panic(fmt.Errorf("M2u and M2d don't match: "+
								"(k,j,i)=(%v,%v,%v); val=%v; m2u=%v; "+
								"m2d=%v, m2dAbove=%v; kpbl=%v",
								k, j, i, val, m2u, M2d.Get(k, j, i),
								M2d.Get(k+1, j, i), pblTop))
						}
					}
				}
				sem <- empty{}
			}(j)
		}
		for j := 0; j < T.Shape[1]; j++ { // wait for routines to finish
			<-sem
		}
		n++
	}
}

// f2i converts a float to an int (rounding).
func f2i(f float64) int {
	return int(f + 0.5)
}

func arrayAverage(s *sparse.DenseArray, numTsteps int) *sparse.DenseArray {
	n := float64(numTsteps)
	for i, val := range s.Elements {
		s.Elements[i] = val / n
	}
	return s
}

// nextDataNCF returns a function that sequentially retrieves time series data
// for the specified variable (varName) from a series of NetCDF files
// with the given file name template between the given start and end times.
// recordDelta and fileDelta specify the length of time between each file
// and each record within a file, respectively. dateFormat is the format
// in which dates appear in the filename
func nextDataNCF(fileTemplate string, dateFormat string, varName string, start, end time.Time, recordDelta, fileDelta time.Duration) NextData {
	recordsPerFile := int(fileDelta / recordDelta)
	var i int
	date := start
	return func() (*sparse.DenseArray, error) {
		if !date.Before(end) {
			return nil, io.EOF
		}
		d := date.Format(dateFormat)
		file := strings.Replace(fileTemplate, "[DATE]", d, -1)
		f, err := os.Open(file)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		ff, err := cdf.Open(f)
		data, err := readNCF(varName, ff, i)
		if err != nil {
			return nil, err
		}
		i++
		if i == recordsPerFile {
			i = 0
			date = date.Add(fileDelta)
		}
		return data, err
	}
}

// read a variable out of a netcdf file.
func readNCF(pol string, ff *cdf.File, hour int) (*sparse.DenseArray, error) {
	dims := ff.Header.Lengths(pol)
	if len(dims) == 0 {
		return nil, fmt.Errorf("inmap: preprocessor read netcdf: variable %v not in file", pol)
	}
	dims = dims[1:]
	nread := 1
	for _, dim := range dims {
		nread *= dim
	}
	start, end := make([]int, len(dims)+1), make([]int, len(dims)+1)
	start[0], end[0] = hour, hour+1
	r := ff.Reader(pol, start, end)
	buf := r.Zero(nread)
	_, err := r.Read(buf)
	if err != nil {
		return nil, fmt.Errorf("inmap: preprocessor read netcdf: %v", err)
	}
	data := sparse.ZerosDense(dims...)
	for i, val := range buf.([]float32) {
		data.Elements[i] = float64(val)
	}
	return data, nil
}
