package proj

import (
	"fmt"
	"strings"
)

func testWKT(code string) bool {
	var codeWords = []string{"GEOGCS", "GEOCCS", "PROJCS", "LOCAL_CS"}
	for _, c := range codeWords {
		if strings.Contains(code, c) {
			return true
		}
	}
	return false
}

func testProj(code string) bool {
	return len(code) >= 1 && code[0] == '+'
}

// Parse parses a WKT- or PROJ4-formatted projection string into a Proj object.
func Parse(code string) (*SR, error) {
	//check to see if this is a WKT string
	if p, ok := defs[code]; ok {
		return p, nil
	} else if testWKT(code) {
		p, err := wkt(code)
		if err != nil {
			return nil, err
		}
		p.DeriveConstants()
		return p, nil
	} else if testProj(code) {
		p, err := projString(code)
		if err != nil {
			return nil, err
		}
		p.DeriveConstants()
		return p, nil
	}
	return nil, fmt.Errorf("unsupported projection definition '%s'; only proj4 and "+
		"WKT are supported", code)
}
