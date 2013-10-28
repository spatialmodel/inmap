package aim

import (
	"bitbucket.org/ctessum/sparse"
	"sync"
)

type PlumeRiseInfo struct {
	Nx, Ny, Nz   int
	layerHeights *sparse.DenseArray // heights at layer edges, m
	temperature  *sparse.DenseArray // Average temperature, K
	windSpeed    *sparse.DenseArray // RMS wind speed, m/s
	s1           *sparse.DenseArray // stability parameter
	sClass       *sparse.DenseArray // stability class: "0=Unstable; 1=Stable
	wg           sync.WaitGroup
}

func GetPlumeRiseInfo(filename string) *PlumeRiseInfo {
	d := new(PlumeRiseInfo)
	ff, err := os.Open(filename)
	if err != nil {
		panic(err)
	}
	defer ff.Close()
	f, err := cdf.Open(ff)
	if err != nil {
		panic(err)
	}
	dims := f.Header.Lengths("orgPartitioning")
	d.Nz = dims[0]
	d.Ny = dims[1]
	d.Nx = dims[2]
	wg := sync.WaitGroup
	wg.Add(5)
	d.layerHeights = sparse.ZerosDense(d.Nz, d.Ny, d.Nx)
	go readNCF(filename, &wg, "layerHeights", d.layerHeights)
	d.temperature = sparse.ZerosDense(d.Nz, d.Ny, d.Nx)
	go readNCF(filename, &wg, "temperature", d.temperature)
	d.windSpeed = sparse.ZerosDense(d.Nz, d.Ny, d.Nx)
	go readNCF(filename, &wg, "windSpeed", d.windSpeed)
	d.s1 = sparse.ZerosDense(d.Nz, d.Ny, d.Nx)
	go readNCF(filename, &wg, "s1", d.s1)
	d.sClass = sparse.ZerosDense(d.Nz, d.Ny, d.Nx)
	go readNCF(filename, &wg, "sClass", d.sClass)
	wg.Wait()
	return d
}

// CalcPlumeRise takes emissions stack height(m), diameter (m), temperature (K),
// and exit velocity (m/s) and calculates the k index of the equivalent
// emissions height after accounting for plume rise at grid index (y=j,x=i).
// Uses the plume rise calculation: ASME (1973), as described in Sienfeld and Pandis,
// ``Atmospheric Chemistry and Physics - From Air Pollution to Climate Change
func (m *PlumeRiseInfo) CalcPlumeRise(stackHeight, stackDiam, stackTemp,
	stackVel float64, j, i int) (kPlume int) {
	// Find K level of stack
	kStak := 0
	for m.layerHeights.Get(kStak+1, j, i) < stackHeight {
		if kStak > m.Nz {
			msg := "stack height > top of grid"
			panic(msg)
		}
		kStak++
	}
	deltaH := 0. // Plume rise, (m).
	var calcType string

	airTemp := m.temperature.Get(kStak, j, i)
	windSpd := m.windSpeed.Get(kStak, j, i)

	if (stackTemp-airTemp) < 50. &&
		stackVel > windSpd && stackVel > 10. {
		// Plume is dominated by momentum forces
		calcType = "Momentum"

		deltaH = stackDiam * math.Pow(stackVel, 1.4) / math.Pow(windSpd, 1.4)

	} else { // Plume is dominated by buoyancy forces

		// Bouyancy flux, m4/s3
		F := g * (stackTemp - airTemp) / stackTemp * stackVel *
			math.Pow(stackDiam/2, 2)

		if m.sClass.Get(kStak, j, i) > 0.5 { // stable conditions
			calcType = "Stable"

			deltaH = 29. * math.Pow(
				F/m.s1.Get(kStak, j, i), 0.333333333) /
				math.Pow(windSpd, 0.333333333)

		} else { // unstable conditions
			calcType = "Unstable"

			deltaH = 7.4 * math.Pow(F*math.Pow(stackHeight, 2.),
				0.333333333) / windSpd

		}
	}
	if math.IsNaN(deltaH) {
		msg := "plume height == NaN\n" +
			fmt.Sprintf("calcType: %v, deltaH: %v, stackDiam: %v,\n",
				calcType, deltaH, stackDiam) +
			fmt.Sprintf("stackVel: %v, windSpd: %v, stackTemp: %v,\n",
				stackVel, windSpd, stackTemp) +
			fmt.Sprintf("airTemp: %v, stackHeight: %v\n", airTemp, stackHeight)
		panic(msg)
	}

	plumeHeight := stackHeight + deltaH

	// Find K level of plume
	for kPlume = 0; m.layerHeights.Get(kPlume+1, j, i) < plumeHeight; kPlume++ {
		if kPlume > m.Nz {
			break
		}
	}
	return
}
