package greet

import (
	"encoding/csv"
	"fmt"
	"io"
	"strings"

	"github.com/spatialmodel/inmap/emissions/slca"
)

// AddSCCs adds SCC codes to the greet database by matching information in
// sscFile.
func (db *DB) AddSCCs(stationaryProcessFile, vehicleFile, technologyFile io.Reader) error {
	allSCCsMap, err := db.stationarySCCs(stationaryProcessFile)
	if err != nil {
		return err
	}
	allSCCsMap, err = db.vehicleSCCs(vehicleFile, allSCCsMap)
	if err != nil {
		return err
	}
	allSCCsMap, err = db.technologySCCs(technologyFile, allSCCsMap)
	if err != nil {
		return err
	}

	allSCCs := make([]slca.SCC, 0, len(allSCCsMap))
	for scc := range allSCCsMap {
		allSCCs = append(allSCCs, scc)
	}
	db.SpatialSCC = allSCCs
	return nil
}

func (db *DB) stationarySCCs(sccFile io.Reader) (map[slca.SCC]int8, error) {
	r := csv.NewReader(sccFile)
	lines, err := r.ReadAll()
	if err != nil {
		return nil, err
	}
	sccs := make(map[string][]slca.SCC) // map[stationary process ID]scc codes
	srgs := make(map[string]string)     // map[stationary process ID]surrogate code
	noSpatial := make(map[string]bool)  // map[StationaryProcesses ID]Ignore spatial?
	allSCCsMap := make(map[slca.SCC]int8)
	for _, line := range lines[1:len(lines)] {
		id := line[2]
		c := strings.Split(line[5], ";")
		cc := make([]slca.SCC, 0, len(c))
		for _, code := range c {
			if code != "" {
				scc := slca.SCC(adjSCC(code))
				cc = append(cc, scc)
				allSCCsMap[scc] = 0
			}
		}
		if len(cc) > 0 {
			sccs[id] = cc
		}
		srg := line[12]
		if srg != "" {
			srgs[id] = srg
		}
		if line[8] != "" || line[11] != "" {
			// Line[8] = code 9993: not a real process
			// line[11] = code 9996: not listed and assumed negligible emissions
			noSpatial[id] = true
		}
	}
	for _, p := range db.Data.StationaryProcesses {
		codes := sccs[string(p.ID)]
		srg := srgs[string(p.ID)]
		noSp := noSpatial[string(p.ID)]
		p.SpatialReference = &slca.SpatialRef{
			SCCs:      codes,
			Surrogate: srg,
			NoSpatial: noSp,
			Type:      slca.Stationary,
		}
	}
	return allSCCsMap, nil
}

func (db *DB) vehicleSCCs(sccFile io.Reader, allSCCs map[slca.SCC]int8) (map[slca.SCC]int8, error) {
	r := csv.NewReader(sccFile)
	lines, err := r.ReadAll()
	if err != nil {
		return nil, err
	}
	sccs := make(map[string]slca.SCC)
	for i, line := range lines {
		if i == 0 {
			continue
		}
		name := line[0]
		scc := slca.SCC(adjSCC(line[1]))
		sccs[name] = scc
		// allSCCs[scc] = 1 This SCC is not used for spatialization so we don't add it here.
	}
	for _, v := range db.Data.Vehicles {
		scc, ok := sccs[string(v.GetName())]
		if !ok {
			return nil, fmt.Errorf("no SCC code for vehicle ID %s", v.GetID())
		}
		v.SCC = scc
	}
	return allSCCs, nil
}

func (db *DB) technologySCCs(sccFile io.Reader, allSCCs map[slca.SCC]int8) (map[slca.SCC]int8, error) {
	r := csv.NewReader(sccFile)
	lines, err := r.ReadAll()
	if err != nil {
		return nil, err
	}
	sccs := make(map[string]slca.SCC)
	for i, line := range lines {
		if i == 0 {
			continue
		}
		id := line[0]
		scc := slca.SCC(adjSCC(line[2]))
		sccs[id] = scc
		// allSCCs[scc] = 1 This SCC is not used for spatialization so we don't add it here.
	}
	for _, t := range db.Data.Technologies {
		scc, ok := sccs[string(t.GetID())]
		if !ok {
			return nil, fmt.Errorf("no SCC code for technology ID %s", t.GetID())
		}
		t.SCC = scc
	}
	return allSCCs, nil
}

func adjSCC(SCC string) string {
	SCC = strings.Replace(SCC, "'", "", -1)
	if len(SCC) == 8 {
		SCC = "00" + SCC
	} else if len(SCC) == 7 {
		SCC = "00" + SCC + "0"
	} else if len(SCC) == 6 {
		SCC = "00" + SCC + "00"
	} else if len(SCC) == 5 {
		SCC = "00" + SCC + "000"
	} else if len(SCC) == 4 {
		SCC = "00" + SCC + "0000"
	} else if len(SCC) == 3 {
		SCC = "00" + SCC + "00000"
	} else if len(SCC) == 2 {
		SCC = "00" + SCC + "000000"
	}
	return SCC
}
