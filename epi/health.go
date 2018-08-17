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

// Package epi holds a collection of functions for calculating the health impacts of air pollution.
package epi

import (
	"math"

	"github.com/gonum/floats"
)

// Nasari implements a class of simple approximations to the exposure response
// models described in:
//
// Nasari M, Szyszkowicz M, Chen H, Crouse D, Turner MC, Jerrett M, Pope CA III,
// Hubbell B, Fann N, Cohen A, Gapstur SM, Diver WR, Forouzanfar MH, Kim S-Y,
// Olives C, Krewski D, Burnett RT. (2015). A Class of Non-Linear
// Exposure-Response Models Suitable for Health Impact Assessment Applicable
// to Large Cohort Studies of Ambient Air Pollution.  Air Quality, Atmosphere,
// and Health: DOI: 10.1007/s11869-016-0398-z.
type Nasari struct {
	// Gamma, Delta, and Lambda are parameters fit using linear regression.
	Gamma, Delta, Lambda float64

	// F is the concentration transformation function.
	F func(z float64) float64

	// Label is the name of the function.
	Label string
}

// HR calculates the hazard ratio caused by concentration z.
func (n Nasari) HR(z float64) float64 {
	return math.Exp(n.Gamma * n.F(z) / (1 + math.Exp(-(z-n.Delta)/n.Lambda)))
}

// Name returns the label for this function.
func (n Nasari) Name() string { return n.Label }

// NasariACS is an exposure-response model fit to the American Cancer Society
// Cancer Prevention II cohort all causes of death from fine particulate matter.
var NasariACS = Nasari{
	Gamma:  0.0478,
	Delta:  6.94,
	Lambda: 3.37,
	F:      func(z float64) float64 { return math.Log(z + 1) },
	Label:  "NasariACS",
}

// Cox implements a Cox proportional hazards model.
type Cox struct {
	// Beta is the model coefficient
	Beta float64

	// Threshold is the concentration below which health effects are assumed
	// to be zero.
	Threshold float64

	// Label is the name of the function.
	Label string
}

// HR calculates the hazard ratio caused by concentration z.
func (c Cox) HR(z float64) float64 {
	return math.Exp(c.Beta * math.Max(0, z-c.Threshold))
}

// Name returns the label for this function.
func (c Cox) Name() string { return c.Label }

// Krewski2009 is a Cox proportional-hazards model from the study:
//
// Krewski, D., Jerrett, M., Burnett, R. T., Ma, R., Hughes, E., Shi, Y., … Thun, M. J. (2009).
// Extended Follow-Up and Spatial Analysis of the American Cancer Society Study Linking
// Particulate Air Pollution and Mortality. Retrieved from http://www.ncbi.nlm.nih.gov/pubmed/19627030
//
// This function is from Table 11 of the study and does not account for ecologic
// covariates.
var Krewski2009 = Cox{
	Beta:      0.005826890812, // ln(1.06) / 10
	Threshold: 5,              // Lowest observed concentration.
	Label:     "Krewski2009",
}

// Krewski2009Ecologic is a Cox proportional-hazards model from the study:
//
// Krewski, D., Jerrett, M., Burnett, R. T., Ma, R., Hughes, E., Shi, Y., … Thun, M. J. (2009).
// Extended Follow-Up and Spatial Analysis of the American Cancer Society Study Linking
// Particulate Air Pollution and Mortality. Retrieved from http://www.ncbi.nlm.nih.gov/pubmed/19627030
//
// This function is from Table 11 of the study and does not account for ecologic
// covariates.
var Krewski2009Ecologic = Cox{
	Beta:      0.007510747249, // ln(1.078) / 10
	Threshold: 5,              // Lowest observed concentration.
	Label:     "Krewski2009Ecologic",
}

// Lepeule2012 is a Cox proportional-hazards model from the study:
//
// Lepeule, J., Laden, F., Dockery, D., & Schwartz, J. (2012). Chronic exposure
// to fine particles and mortality: An extended follow-up of the Harvard six
// cities study from 1974 to 2009. Environmental Health Perspectives, 120(7),
// 965–970. http://doi.org/10.1289/ehp.1104660
var Lepeule2012 = Cox{
	Beta:      0.01310282624, // ln(1.14) / 10
	Threshold: 8,             // Lowest observed concentration.
	Label:     "Lepeule2012",
}

// HRer is an interface for any type that can calculate the hazard ratio
// caused by concentration z.
type HRer interface {
	HR(z float64) float64
	Name() string
}

// IoRegional returns the underlying regional average incidence rate for a region where
// the reported incidence rate is I, individual locations within the
// region have population p and concentration z, and hr specifies the
// hazard ratio as a function of z, as presented in Equations 2 and 3 of:
//
// Apte JS, Marshall JD, Cohen AJ, Brauer M (2015) Addressing Global
// Mortality from Ambient PM2.5. Environmental Science and Technology
// 49(13):8057–8066.
func IoRegional(p, z []float64, hr HRer, I float64) float64 {
	var hrBar float64
	for i, pi := range p {
		hrBar += pi * hr.HR(z[i])
	}
	pSum := floats.Sum(p)
	hrBar /= pSum
	if pSum == 0 || hrBar == 0 {
		return 0
	}
	return I / hrBar
}

// Io returns the underlying incidence rate where
// the reported incidence rate is I, concentration is z,
// and hr specifies the hazard ratio as a function of z. When possible,
// IoRegional should be used instead of this function.
func Io(z float64, hr HRer, I float64) float64 {
	return I / hr.HR(z)
}

// Outcome returns the number of incidences occuring in population p when
// exposed to concentration z given underlying incidence rate Io and
// hazard relationship hr(z), as presented in Equation 2 of:
//
// Apte JS, Marshall JD, Cohen AJ, Brauer M (2015) Addressing Global
// Mortality from Ambient PM2.5. Environmental Science and Technology
// 49(13):8057–8066.
func Outcome(p, z, Io float64, hr HRer) float64 {
	return p * Io * (hr.HR(z) - 1)
}
