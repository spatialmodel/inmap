package main

import (
	"encoding/json"
	"fmt"
	"github.com/ctessum/aep"
)

func main() {
	const (
		wrfNamelist = "/home/marshall/tessumcm/WRFchem_output/WRF.2005_inmaptest.la/1/namelist.input"
		wpsNamelist = "/home/marshall/tessumcm/WRFchem_output/WPS.socab/namelist.wps"
	)
	config, err := aep.ParseWRFConfig(wpsNamelist, wrfNamelist)
	if err != nil {
		panic(err)
	}
	b, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		panic(err)
	}
	fmt.Print(string(b))
}
