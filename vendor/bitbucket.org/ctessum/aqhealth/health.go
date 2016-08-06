package aqhealth

import (
	"math"
	"sort"
)

// Relative risk from PM2.5 concentration change, assuming a
// log-log dose response (almost a linear relationship).
// From From Krewski et al (2009, Table 11)
// and Josh Apte (personal communication).
func RRpm25Linear(pm25 float64) float64 {
	return math.Exp(PM25linearCoefficient * pm25)
}

var PM25linearCoefficient = 0.007510747

// Relative risk from PM2.5 concentration change, assuming a
// log-linear dose response. From From Krewski et al (2009, Table 11)
// and Josh Apte (personal communication).
func RRpm25Log(baselinePM25, Δpm25 float64) float64 {
	var RR = 0.
	if baselinePM25 != 0. {
		RR = PM25logCoefficient * math.Pow((Δpm25+baselinePM25)/
			baselinePM25, 0.109532154)
	}
	return RR
}

var PM25logCoefficient = 1.000112789
var PM25logExponent = 0.109532154

// Relative risk from O3 concentration change, assuming a
// log-log dose response (almost a linear relationship).
// From From Jerrett et al (2009)
// and Josh Apte (personal communication).
func RRo3Linear(o3 float64) float64 {
	return math.Exp(O3linearCoefficient * o3)
}

var O3linearCoefficient = 0.003922071

// MR (baseline mortality rate) is in units of deaths per 100,000 people
// per year.
// people * deathsPer100,000 / 100,000 * RR = delta deaths
func Deaths(RR, population, MR float64) float64 {
	return (RR - 1) * population * MR / 100000.
}

// Calculate a metric to be used in the calculation of inequality indicies
// (atkinson and gini). It is the absolute value of the difference between
// changes in mortality rates at specific locations and the maximum change
// in mortality rate. Inputs are location-specific total numbers of
// deaths and population.
func NegativeDeltaMortalityRate(ΔDeaths, population []float64) (
	ndmr, nonZeroPop []float64) {
	ndmr = make([]float64, 0, len(population))
	nonZeroPop = make([]float64, 0, len(population))
	for i, d := range ΔDeaths {
		p := population[i]
		if p != 0 {
			nonZeroPop = append(nonZeroPop, p)
			ndmr = append(ndmr, d/p)
		}
	}
	maxdmr := max(ndmr...)
	for i, n := range ndmr {
		ndmr[i] = math.Abs(maxdmr*1.001 - n)
	}
	return
}

func max(vals ...float64) float64 {
	max := vals[0]
	for _, val := range vals {
		if val > max {
			max = val
		}
	}
	return max
}

// Calculates Atkinson Index from grouped data, where:
//	negativeDeltaMortalityRate = see above;
//	population = location-specific population,
//	b = inequality aversion parameter
func AtkinsonGrouped(negativeDeltaMortalityRate,
	population []float64, b float64) float64 {
	w := 0.
	psum := 0.
	for i, ci := range negativeDeltaMortalityRate {
		w += ci * population[i]
		psum += population[i]
	}
	w /= psum
	f := make([]float64, len(population))
	for i, pi := range population {
		f[i] = pi / psum
	}
	var A float64
	if b == 1. {
		x := 0.
		for i, ci := range negativeDeltaMortalityRate {
			x += f[i] * math.Log(ci/w)
		}
		A = 1. - math.Exp(x)
	} else {
		x := 0.
		for i, ci := range negativeDeltaMortalityRate {
			x += f[i] * math.Pow(ci/w, 1.-b)
		}
		A = 1 - math.Pow(x, 1./(1.-b))
	}
	return A
}

// FORMAT: gini(values, population)
// Given these values, this computes the GINI coefficient according to
// footnote 17 of Noorbakhsh.
// ndmr is negativeDeltaMortalityRate (see above)
// pop is population
// ndmr and pop must be arrays of the same length
func Gini(ndmr, pop []float64) float64 {
	P := 0.
	for _, p := range pop {
		P += p
	}
	mu := 0.
	for i, p := range pop {
		mu += ndmr[i] * p / P
	}
	total := 0.
	for i := 0; i < len(pop); i++ {
		if pop[i] != 0. {
			for j := 0; j < len(pop); j++ {
				if pop[j] != 0. && (ndmr[i] != 0. || ndmr[j] != 0.) {
					total += pop[i] * pop[j] / P / P *
						math.Abs(ndmr[i]-ndmr[j])
				}
			}
		}
	}
	return total / mu
}

type ginisort struct{ x, w, p, nu float64 }
type ginisort2 []*ginisort

func (g ginisort2) Len() int { return len(g) }
func (g ginisort2) Swap(i, j int) {
	g[i].x, g[i].w, g[j].x, g[i].w = g[j].x, g[j].w, g[i].x, g[i].w
}
func (g ginisort2) Less(i, j int) bool { return g[i].x < g[j].x }

func Gini2(x, weights []float64) float64 {
	g := make([]*ginisort, len(x))
	for i, xi := range x {
		g[i] = new(ginisort)
		g[i].x = xi
		g[i].w = weights[i]
	}
	sort.Sort(ginisort2(g))

	weightSum := 0.
	for _, gi := range g {
		weightSum += gi.w
	}
	for _, gi := range g {
		gi.w /= weightSum
	}
	for i, gi := range g { // cumulative sum of w
		if i == 0 {
			gi.p = gi.w
		} else {
			gi.p = gi.w + g[i-1].p
		}
	}
	for i, gi := range g { // cumulative sum of w*x
		if i == 0 {
			gi.nu = gi.w * gi.x
		} else {
			gi.nu = gi.w*gi.x + g[i-1].nu
		}
	}
	numax := g[len(g)-1].nu
	for _, gi := range g { // normalize nu
		gi.nu /= numax
	}
	asum, bsum := 0., 0.
	for i := 1; i < len(g); i++ {
		asum += g[i].nu * g[i-1].p
		bsum += g[i-1].nu * g[i].p
	}
	return asum - bsum
}
