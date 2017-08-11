// +build FORTRAN

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

// Package mosaic contains an interface to the MOSAIC atmospheric
// chemistry mechanism as desribed in:
//
// Zaveri, R. A., R. C. Easter, J. D. Fast, and L. K. Peters (2008), Model for
// Simulating Aerosol Interactions and Chemistry (MOSAIC), J. Geophys. Res., 113,
//  D13204, doi:10.1029/2007JD008782.
package mosaic

import (
	"fmt"
	"math"
	"sync"

	mos "bitbucket.org/ctessum/mosaic"

	"github.com/spatialmodel/inmap"
	"github.com/spatialmodel/inmap/science/drydep/simpledrydep"
	"github.com/spatialmodel/inmap/science/wetdep/emepwetdep"
)

const (
	// nGas is the number of gas species.
	nGas = 77
	// nAer is the number of aerosol species.
	nAer = 24
)

// Mechanism fulfils the github.com/spatialmodel/inmap.Mechanism
// interface for the CBMZ-MOSAIC chemical mechanism.
type Mechanism struct {
	mos.Mosaic

	// varIndices give the array index for each variable name.
	varIndices map[string]int
}

// NewMechanism returns a new MOSAIC chemical mechanism.
func NewMechanism() *Mechanism {
	m := &Mechanism{
		Mosaic: mos.Mosaic{
			BeginMonth:       03,
			BeginDay:         01,
			BeginHour:        00,
			BeginMinute:      00,
			BeginSecond:      00,
			DtAerOpticMin:    60.0,
			Lon:              -97,
			Lat:              40,
			RH:               30.0,
			P:                1.0,
			TypesMD1Aer:      1,
			TypesMD2Aer:      1,
			MethodBCFrac:     1,
			MethodKappa:      11,
			AerSizes:         1,
			SizeFramework:    2,
			AerSizeInitFlag:  1,
			HystMethod:       1,
			CoagFlag1:        0,
			CoagFreq:         1,
			MoveSectFlag:     0,
			NewNucFlag1:      0,
			SectionalFlag1:   0,
			Print:            1,
			WriteFullOut:     0,
			WriteGas:         0,
			WriteAerBin:      0,
			WriteAerDist:     0,
			WriteAerSpecies:  0,
			WriteAerOptic:    0,
			WriteSect170:     0,
			WriteSect171:     0,
			WriteSect172:     0,
			WriteSect190:     0,
			WriteSect180:     0,
			WriteSect183:     0,
			WriteSect184:     0,
			WriteSect185:     0,
			WriteSect186:     0,
			WriteSect188:     0,
			DiagSectCoag:     0,
			DiagSectMoveSect: 0,
			DiagSectNewNuc:   0,
			Mode:             1,
			GasOn:            1,
			AerOn:            1,
			CloudOn:          1,
			AerOpticOn:       0,
			ShellCore:        1,
			Solar:            1,
			Photo:            2,
			GasAerXfer:       1,
			DynamicSolver:    1,
			AlphaASTEM:       0.5,
			RTolEqBASTEM:     0.01,
			PTolMolASTEM:     0.01,
			PMCMOS:           0,
			CAer:             make([][nAer]float32, 1),
		},
		varIndices: make(map[string]int),
	}
	// Initialize varIndices.
	for i := 0; i < nGas; i++ {
		m.varIndices[fmt.Sprint(mos.GasSpecies(i))] = i
	}
	for i := nGas; i < nGas+nAer; i++ {
		m.varIndices[fmt.Sprint(mos.AerosolSpecies(i-nGas))] = i
	}
	return m
}

// Len returns the number of chemical species in this mechanism (101).
func (m *Mechanism) Len() int { return nGas + nAer }

// The FORTRAN code uses global variables and therefore
// cannot be paralellized, so we use a mutex to make
// sure multiple instances are not run concurrently.
var mx sync.Mutex

// Chemistry returns a function that simulates chemical reactions using
// the CBMZ-MOSAIC model.
func (m *Mechanism) Chemistry() inmap.CellManipulator {
	return func(c *inmap.Cell, Δt float64) {
		mx.Lock()
		defer mx.Unlock()
		// Initialize parameters.
		m.Mosaic.T = c.Temperature
		m.Mosaic.Altitude = c.LayerHeight + c.Dz/2
		m.Mosaic.DtMinutes = Δt / 60
		m.Mosaic.RunSeconds = int(Δt + 0.5)

		// Initialize concentrations.
		for i := 0; i < nGas; i++ {
			m.Mosaic.CGas[i] = float32(c.Cf[i])
		}
		for i := nGas; i < nGas+nAer; i++ {
			m.Mosaic.CAer[0][i-nGas] = float32(c.Cf[i])
		}

		// Run the model.
		if err := m.Mosaic.Run(); err != nil {
			panic(err)
		}

		// Extract concentrations.
		for i := 0; i < nGas; i++ {
			c.Cf[i] = float64(m.Mosaic.CGas[i])
		}
		for i := nGas; i < nGas+nAer; i++ {
			c.Cf[i] = float64(m.Mosaic.CAer[0][i-nGas])
		}
	}
}

// AddEmisFlux adds emissions flux to Cell c based on the given
// pollutant name and amount in units of μg/s. The units of
// the resulting flux are the native units for each MOSAIC species per second.
func (m *Mechanism) AddEmisFlux(c *inmap.Cell, name string, val float64) error {
	if c.EmisFlux == nil {
		c.EmisFlux = make([]float64, m.Len())
	}
	const (
		rhoInv = 1 / 1.225 // [m³/kg] TODO: use actual air density
		mwA    = 28.97     // g/mol, molar mass of air
		mwPar  = 16.0
		mwNH3  = 17.03056
		mwSO2  = 64.0644
		mwNO   = 30.01
	)
	fluxScale := 1. / c.Dx / c.Dy / c.Dz // μg/s /m/m/m = μg/m³/s

	switch name {
	case "VOC":
		for i := 0; i < nGas; i++ {
			c.EmisFlux[i] += val * rhoInv * mwA / mwPar * fluxScale / float64(nGas) // [ppb/s]
		}
		//c.EmisFlux[mos.PAR] += val * rhoInv * mwA / mwPar * fluxScale // [ppb/s]
	case "NOx":
		//c.EmisFlux[mos.NO] += val * rhoInv * mwA / mwNO * fluxScale // [ppb/s]
		c.EmisFlux[mos.NO] += val * rhoInv * mwA / mwNO * fluxScale / 2  // [ppb/s]
		c.EmisFlux[mos.NO2] += val * rhoInv * mwA / mwNO * fluxScale / 2 // [ppb/s]
	case "NH3":
		c.EmisFlux[mos.NH3] += val * rhoInv * mwA / mwNH3 * fluxScale // [ppb/s]
	case "SOx":
		c.EmisFlux[mos.SO2] += val * rhoInv * mwA / mwSO2 * fluxScale // [ppb/s]
	case "PM2_5":
		//c.EmisFlux[mos.BC+nGas] += val * fluxScale // [μg/m³/s]
		for i := nGas; i < len(c.EmisFlux); i++ {
			c.EmisFlux[i] += val * fluxScale / float64(len(c.EmisFlux)-nGas) // [μg/m³/s]
		}
	default:
		return fmt.Errorf("mosaic: '%s' is not a valid emissions species; valid options are VOC, NOx, NH3, SOx, and PM2_5", name)
	}
	return nil
}

// Species returns the names of the emission and concentration pollutant
// species that are used by this chemical mechanism.
func (m *Mechanism) Species() []string {
	o := make([]string, nGas+nAer)
	for i := 0; i < nGas; i++ {
		o[i] = fmt.Sprint(mos.GasSpecies(i))
	}
	for i := 0; i < nAer; i++ {
		o[i+nGas] = fmt.Sprint(mos.AerosolSpecies(i))
	}
	return o
}

// Value returns the concentration or emissions value of
// the given variable in the given Cell. It returns an
// error if given an invalid variable name.
func (m *Mechanism) Value(c *inmap.Cell, variable string) (float64, error) {
	i, ok := m.varIndices[variable]
	if !ok {
		return math.NaN(), fmt.Errorf("mosaic: invalid variable name %s; valid names are %v", variable, m.Species())
	}
	return c.Cf[i], nil
}

// Units returns the units of the given variable.
func (m *Mechanism) Units(variable string) (string, error) {
	aerUnits := map[string]string{
		"Num":    "#/cc",
		"DpgN":   "um",
		"Sigmag": "-",
		"JHyst":  "flag",
		"Water":  "kg/m³",
		"SO4":    "umol/m³",
		"PNO3":   "umol/m³",
		"Cl":     "umol/m³",
		"NH4":    "umol/m³",
		"PMSA":   "umol/m³",
		"Aro1":   "umol/m³",
		"Aro2":   "umol/m³",
		"Alk1":   "umol/m³",
		"Ole1":   "umol/m³",
		"PApi1":  "umol/m³",
		"PApi2":  "umol/m³",
		"Lim1":   "umol/m³",
		"Lim2":   "umol/m³",
		"CO3":    "umol/m³",
		"Na":     "umol/m³",
		"Ca":     "umol/m³",
		"Oin":    "μg/m³",
		"OC":     "μg/m³",
		"BC":     "μg/m³",
	}
	if u, ok := aerUnits[variable]; ok {
		return u, nil
	}
	for i := 0; i < nGas; i++ {
		if variable == fmt.Sprint(mos.GasSpecies(i)) {
			// Gas species are all [ppb].
			return "ppb", nil
		}
	}
	return "", fmt.Errorf("mosaic: invalid variable name %s; valid names are %v", variable, m.Species())
}

// simpleDryDepIndices provides array indices for use with package simpledrydep.
func simpleDryDepIndices() (simpledrydep.SOx, simpledrydep.NH3, simpledrydep.NOx, simpledrydep.VOC, simpledrydep.PM25) {
	return simpledrydep.SOx{
			int(mos.SO2),
			int(mos.H2SO4),
			int(mos.SULFHOX),
		},
		simpledrydep.NH3{
			int(mos.NH3),
		},
		simpledrydep.NOx{
			int(mos.HNO3),
			int(mos.HCl),
			int(mos.NO),
			int(mos.NO2),
			int(mos.NO3),
			int(mos.N2O5),
			int(mos.HONO),
			int(mos.HNO4),
		},
		simpledrydep.VOC{
			int(mos.O3), // TODO: Some of these are clearly not VOCs.
			int(mos.O1D),
			int(mos.O3P),
			int(mos.OH),
			int(mos.HO2),
			int(mos.H2O2),
			int(mos.CO),
			int(mos.CH4),
			int(mos.C2H6),
			int(mos.CH3O2),
			int(mos.ETHP),
			int(mos.HCHO),
			int(mos.CH3OH),
			int(mos.ANOL),
			int(mos.CH3OOH),
			int(mos.ETHOOH),
			int(mos.ALD2),
			int(mos.HCOOH),
			int(mos.RCOOH),
			int(mos.C2O3),
			int(mos.PAN),
			int(mos.ARO1),
			int(mos.ARO2),
			int(mos.ALK1),
			int(mos.OLE1),
			int(mos.API1),
			int(mos.API2),
			int(mos.LIM1),
			int(mos.LIM2),
			int(mos.PAR),
			int(mos.AONE),
			int(mos.MGLY),
			int(mos.ETH),
			int(mos.OLET),
			int(mos.OLEI),
			int(mos.TOL),
			int(mos.XYL),
			int(mos.CRES),
			int(mos.TO2),
			int(mos.CRO),
			int(mos.OPEN),
			int(mos.ONIT),
			int(mos.ROOH),
			int(mos.RO2),
			int(mos.ANO2),
			int(mos.NAP),
			int(mos.XO2),
			int(mos.XPAR),
			int(mos.ISOP),
			int(mos.ISOPRD),
			int(mos.ISOPP),
			int(mos.ISOPN),
			int(mos.ISOPO2),
			int(mos.API),
			int(mos.LIM),
			int(mos.DMS),
			int(mos.MSA),
			int(mos.DMSO),
			int(mos.DMSO2),
			int(mos.CH3SO2H),
			int(mos.CH3SCH2OO),
			int(mos.CH3SO2),
			int(mos.CH3SO3),
			int(mos.CH3SO2OO),
			int(mos.CH3SO2CH2OO),
		},
		simpledrydep.PM25{
			int(mos.SO4),
			int(mos.PNO3),
			int(mos.Cl),
			int(mos.NH4),
			int(mos.PMSA),
			int(mos.Aro1),
			int(mos.Aro2),
			int(mos.Alk1),
			int(mos.Ole1),
			int(mos.PApi1),
			int(mos.PApi2),
			int(mos.Lim1),
			int(mos.Lim2),
			int(mos.CO3),
			int(mos.Na),
			int(mos.Ca),
			int(mos.Oin),
			int(mos.OC),
			int(mos.BC),
		}
}

// DryDep returns a dry deposition function of the type indicated by
// name that is compatible with this chemical mechanism.
// Currently, the only valid option is "simple".
func (m *Mechanism) DryDep(name string) (inmap.CellManipulator, error) {
	options := map[string]inmap.CellManipulator{
		"simple": simpledrydep.DryDeposition(simpleDryDepIndices),
	}
	f, ok := options[name]
	if !ok {
		return nil, fmt.Errorf("mosaic: invalid dry deposition option %s; 'chem' is the only valid option", name)
	}
	return f, nil
}

// emepWetDepIndices provides array indices for use with package emepwetdep.
func emepWetDepIndices() (emepwetdep.SO2, emepwetdep.OtherGas, emepwetdep.PM25) {
	return emepwetdep.SO2{
			int(mos.SO2),
			int(mos.H2SO4),
			int(mos.SULFHOX),
		},
		emepwetdep.OtherGas{
			int(mos.NH3),
			int(mos.HNO3),
			int(mos.HCl),
			int(mos.NO),
			int(mos.NO2),
			int(mos.NO3),
			int(mos.N2O5),
			int(mos.HONO),
			int(mos.HNO4),
			int(mos.O3),
			int(mos.O1D),
			int(mos.O3P),
			int(mos.OH),
			int(mos.HO2),
			int(mos.H2O2),
			int(mos.CO),
			int(mos.CH4),
			int(mos.C2H6),
			int(mos.CH3O2),
			int(mos.ETHP),
			int(mos.HCHO),
			int(mos.CH3OH),
			int(mos.ANOL),
			int(mos.CH3OOH),
			int(mos.ETHOOH),
			int(mos.ALD2),
			int(mos.HCOOH),
			int(mos.RCOOH),
			int(mos.C2O3),
			int(mos.PAN),
			int(mos.ARO1),
			int(mos.ARO2),
			int(mos.ALK1),
			int(mos.OLE1),
			int(mos.API1),
			int(mos.API2),
			int(mos.LIM1),
			int(mos.LIM2),
			int(mos.PAR),
			int(mos.AONE),
			int(mos.MGLY),
			int(mos.ETH),
			int(mos.OLET),
			int(mos.OLEI),
			int(mos.TOL),
			int(mos.XYL),
			int(mos.CRES),
			int(mos.TO2),
			int(mos.CRO),
			int(mos.OPEN),
			int(mos.ONIT),
			int(mos.ROOH),
			int(mos.RO2),
			int(mos.ANO2),
			int(mos.NAP),
			int(mos.XO2),
			int(mos.XPAR),
			int(mos.ISOP),
			int(mos.ISOPRD),
			int(mos.ISOPP),
			int(mos.ISOPN),
			int(mos.ISOPO2),
			int(mos.API),
			int(mos.LIM),
			int(mos.DMS),
			int(mos.MSA),
			int(mos.DMSO),
			int(mos.DMSO2),
			int(mos.CH3SO2H),
			int(mos.CH3SCH2OO),
			int(mos.CH3SO2),
			int(mos.CH3SO3),
			int(mos.CH3SO2OO),
			int(mos.CH3SO2CH2OO),
		},
		emepwetdep.PM25{
			int(mos.SO4),
			int(mos.PNO3),
			int(mos.Cl),
			int(mos.NH4),
			int(mos.PMSA),
			int(mos.Aro1),
			int(mos.Aro2),
			int(mos.Alk1),
			int(mos.Ole1),
			int(mos.PApi1),
			int(mos.PApi2),
			int(mos.Lim1),
			int(mos.Lim2),
			int(mos.CO3),
			int(mos.Na),
			int(mos.Ca),
			int(mos.Oin),
			int(mos.OC),
			int(mos.BC),
		}
}

// WetDep returns a dry deposition function of the type indicated by
// name that is compatible with this chemical mechanism.
// Currently, the only valid option is "emep".
func (m *Mechanism) WetDep(name string) (inmap.CellManipulator, error) {
	options := map[string]inmap.CellManipulator{
		"emep": emepwetdep.WetDeposition(emepWetDepIndices),
	}
	f, ok := options[name]
	if !ok {
		return nil, fmt.Errorf("mosaic: invalid wet deposition option %s; 'emep' is the only valid option", name)
	}
	return f, nil
}
