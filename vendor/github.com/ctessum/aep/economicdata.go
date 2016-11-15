package aep

import (
	"fmt"
	"strings"
)

// EconomicData holds industry information about an emissions source
type EconomicData struct {
	// SIC is the Standard Industrial Classification Code (recommended)
	SIC string

	// North American Industrial Classification System Code
	// (6 characters maximum) (optional)
	NAICS string
}

// clean up NAICS code so it either has 0 or 6 characters
func (r *EconomicData) parseNAICS(NAICS string) {
	NAICS = trimString(NAICS)
	if NAICS == "" || NAICS == nullVal {
		r.NAICS = ""
	} else {
		r.NAICS = strings.Replace(fmt.Sprintf("%-6s", NAICS), " ", "0", -1)
	}
}

// clean up SIC code so it either has 0 or 4 characters
func (r *EconomicData) parseSIC(SIC string) {
	r.SIC = trimString(SIC)
	if r.SIC == "" || r.SIC == nullVal {
		r.SIC = ""
	} else {
		r.SIC = strings.Replace(fmt.Sprintf("%-4s", r.SIC), " ", "0", -1)
	}
}

// GetEconomicData returns r.
func (r *EconomicData) GetEconomicData() *EconomicData {
	return r
}

func (r *nobusinessPolygonRecord) GetEconomicData() *EconomicData {
	return nil
}
