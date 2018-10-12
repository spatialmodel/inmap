package greet

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/ctessum/unit"
)

// InputTable is a holder for a GREET input data table.
type InputTable struct {
	ID         string        `xml:"id,attr"`
	TabID      string        `xml:"tabid,attr"`
	Notes      string        `xml:"notes,attr"`
	ModifiedOn string        `xml:"modified_on,attr"`
	ModifiedBy string        `xml:"modified_by,attr"`
	Columns    []InputColumn `xml:"column"`
}

// InputColumn is a holder for a GREET input data table column.
type InputColumn struct {
	Name       string      `xml:"name,attr"`
	ID         string      `xml:"id,attr"`
	Parameters []Parameter `xml:"param"`
}

// Parameter is one type of variable parameter from the GREET
// database (different from Param).
// It holds the data in a GREET input table cell.
type Parameter struct {
	Name       string     `xml:"name,attr"`
	ID         string     `xml:"id,attr"`
	Value      Expression `xml:"value,attr"`
	ValueYears *Param     `xml:"values"`
}

// processParameters calculates values for a parameters in the table.
// It only processes a parameter if the cell name matches varName,
// otherwise the value will be processed elsewhere.
func (it *InputTable) processParameter(varName string, db *DB) (
	v *unit.Unit, ok bool) {

	const (
		letters = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	)
	for _, col := range it.Columns {
		letter := numberToColumn(col.ID)
		for _, param := range col.Parameters {
			cellName := fmt.Sprintf("%s!%s%s", it.ID, letter, param.ID)
			if cellName == varName {
				return param.process(db, cellName), true
			}
		}
	}
	return nil, false
}

// process calculates the value of this parameter.
func (p Parameter) process(db *DB, extraNames ...string) *unit.Unit {
	if p.Value != "" {
		return db.evalExpr(p.Value, extraNames...)
	}
	if p.ValueYears != nil {
		v := db.InterpolateValue(p.ValueYears.ValueYears)
		for _, n := range p.ValueYears.varNames() {
			if n != "" {
				db.processedParameters[n] = v
			}
		}
		for _, n := range extraNames {
			if n != "" {
				db.processedParameters[n] = v
			}
		}
		return v
	}
	panic("value and ValueYears are both nil")
}

// Param is one type of variable parameter from the GREET
// database (different from Parameter).
type Param struct {
	// TODO: What does MostRecent mean?
	MostRecent string       `xml:"mostRecent,attr"`
	ValueYears []*ValueYear `xml:"year"`
}

func (p Param) process(db *DB) *unit.Unit {
	v := db.InterpolateValue(p.ValueYears)
	for _, n := range p.varNames() {
		if n != "" {
			db.processedParameters[n] = v
		}
	}
	return v
}

// varNames returns the variable names attributed to p, after stripping off
// the year suffix (e.g. "xxxx_2010" is returned as "xxxx").
func (p Param) varNames() []string {
	var s []string
	if len(p.ValueYears) == 0 {
		return nil
	}
	vy := p.ValueYears[0]
	e := vy.Value.decode()
	for _, n := range e.names {
		s = append(s, strings.TrimSuffix(n, "_"+vy.Year))
	}
	return s
}

func numberToColumn(colID string) string {
	const (
		letters = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	)
	v, err := strconv.ParseInt(colID, 10, 64)
	handle(err)
	value := int(v)

	s := ""
	for {
		value--
		s = string(letters[value%len(letters)]) + s
		value /= len(letters)
		if value == 0 {
			break
		}
	}
	return s
}
