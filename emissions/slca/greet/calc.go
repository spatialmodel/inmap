package greet

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"math"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/ctessum/unit"
)

var fDim, swuDim, dollarsDim, yearDim unit.Dimension

func init() {
	fDim = unit.NewDimension("Fahr")  // fahrenheight
	swuDim = unit.NewDimension("swu") // separative work units
	dollarsDim = unit.NewDimension("$")
	yearDim = unit.NewDimension("yr")
}

// Units
var (
	joulesPerKg = unit.Dimensions{
		unit.LengthDim: 2,
		unit.TimeDim:   -2}
	joulesPerMPerKg = unit.Dimensions{
		unit.LengthDim: 1,
		unit.TimeDim:   -2}
	joulesPerM3 = unit.Dimensions{
		unit.MassDim:   1,
		unit.LengthDim: -1,
		unit.TimeDim:   -2}
	joulesPerMeter = unit.Dimensions{
		unit.MassDim:   1,
		unit.LengthDim: 1,
		unit.TimeDim:   -2}
	metersNeg2 = unit.Dimensions{
		unit.LengthDim: -2}
	fahrenheight = unit.Dimensions{
		fDim: 1}
	kgPerJoule = unit.Dimensions{
		unit.LengthDim: -2,
		unit.TimeDim:   2}
	kgPerMeter = unit.Dimensions{
		unit.MassDim:   1,
		unit.LengthDim: -1}
	kgPerMeter2 = unit.Dimensions{
		unit.MassDim:   1,
		unit.LengthDim: -2}
	meter3PerKg = unit.Dimensions{
		unit.MassDim:   -1,
		unit.LengthDim: 3}
	weirdPressureUnits = unit.Dimensions{
		unit.MassDim:   1,
		unit.LengthDim: -2}
	wattsPerKg = unit.Dimensions{
		unit.LengthDim: 2,
		unit.TimeDim:   -3}
	perSecond = unit.Dimensions{
		unit.TimeDim: -1}
	currency = unit.Dimensions{
		dollarsDim: 1}
	dollarsPerKg = unit.Dimensions{
		dollarsDim:   1,
		unit.MassDim: -1}
	dollars = unit.Dimensions{
		dollarsDim: 1,
	}
	dollarsPerM3 = unit.Dimensions{
		dollarsDim:     1,
		unit.LengthDim: -3}
	dollarsPerJoule = unit.Dimensions{
		dollarsDim:     1,
		unit.MassDim:   -1,
		unit.LengthDim: -2,
		unit.TimeDim:   2}
	year = unit.Dimensions{
		yearDim: 1}
	separativeWork = unit.Dimensions{
		swuDim: 1}
	joulesPerSwu = unit.Dimensions{
		unit.MassDim:   1,
		unit.LengthDim: 2,
		unit.TimeDim:   -2,
		swuDim:         -1}
)

// Expression is an expression that can be evaluated
type Expression string

// expression is a parsed Expression
type expression struct {
	val   string   // value of this expression, this may contain variables
	units string   // Units for value
	names []string // Name(s) for this variable
	user  string   // The person that last edited this
	date  string   // Date and time of most recent edit
	notes string   // Notes regarding this expression
}

// encode creates an Expression string from an expression struct.
func (e expression) encode() Expression {
	name1 := ""
	if len(e.names) > 0 {
		name1 = e.names[0]
	}
	name2 := ""
	if len(e.names) > 1 {
		name2 = e.names[1]
	}
	return Expression(strings.Join([]string{
		"",       // default value
		e.units,  // default units
		e.val,    // user specified value
		e.units,  // user specified units
		falseStr, // use default value?
		"",       // unknown field
		name1,    // first variable name
		e.user,   // user name
		e.date,   // edit date
		name2,    // second variable name
	}, ";"))
}

const (
	falseStr = "False"
	trueStr  = "True"
)

// decode parses the new type of Expression, where the selector between
// default and user-defined values is in position 4.
func (e Expression) decode() expression {
	s := strings.Split(string(e), ";")
	if len(e) == 0 {
		return expression{}
	} else if len(s) < 9 {
		panic(fmt.Errorf("malformed Expression: length must be 9 or 10: %s", string(e)))
	} else if len(s) > 10 {
		panic(fmt.Errorf("malformed Expression: length must be 9 or 10: %s", string(e)))
	}
	var ee expression
	// There are two options for the expression. The first option is the default
	// value, and the second option is the use defined value. If s[4] = true,
	// the user-defined value should be used.
	if s[4] == trueStr { // Use GREET default value and units.
		ee.val = s[0]
		if s[1] == "" {
			// It is not clear why sometimes one of the spots is empty and other
			// times the other spot is empty.
			ee.units = s[3]
		} else {
			ee.units = s[1]
		}
	} else if s[4] == falseStr { // Use user-defined value and units.
		ee.val = s[2]
		if s[3] == "" {
			ee.units = s[1]
		} else {
			ee.units = s[3]
		}
	} else {
		return e.decodeOld()
	}
	ee.notes = s[5]
	if s[6] != "" {
		ee.names = append(ee.names, s[6])
	}
	ee.user = s[7]
	ee.date = s[8]
	if len(s) == 10 && s[9] != "" {
		ee.names = append(ee.names, s[9])
	}
	return ee
}

// decodeOld parses the old type of Expression, where the selector between
// default and user-defined values is in position 2.
func (e Expression) decodeOld() expression {
	s := strings.Split(string(e), ";")
	if len(s) == 0 {
		return expression{}
	} else if len(s) != 9 {
		for _, ss := range s[9 : len(s)+1] {
			if ss != "" {
				panic(fmt.Errorf("malformed Expression: length must be 9: %s", string(e)))
			}
		}
	}
	var ee expression
	// There are two options for the expression. The first option is the default
	// value, and the second option is the use defined value. If s[2] = true,
	// the user-defined value should be used.
	if s[2] == trueStr { // Use GREET default value and units.
		ee.val = s[0]
	} else if s[2] == falseStr { // Use user-defined value and units.
		ee.val = s[1]
	} else {
		panic(fmt.Errorf("malformed expression: "+
			"position 2 must be True or False: %v", string(e)))
	}
	ee.units = s[3]
	ee.names = append(ee.names, s[5])
	ee.user = s[6]
	ee.date = s[7]
	if len(s) == 9 && s[8] != "" {
		ee.names = append(ee.names, s[8])
	}
	return ee
}

// Arithmetic evaluator adapted from http://rosettacode.org/wiki/Arithmetic_Evaluator/Go
func parseAndEval(exp string) (float64, error) {
	if len(exp) > 1 && exp[0:1] == "=" {
		exp = exp[1:len(exp)] // get rid of leading equals sign.
	}
	tree, err := parser.ParseExpr(exp)
	if err != nil {
		return 0, err
	}
	return eval(tree)
}

// eval evaluates and arithmatic expression
func eval(tree ast.Expr) (float64, error) {
	switch n := tree.(type) {
	case *ast.BasicLit:
		if n.Kind != token.INT && n.Kind != token.FLOAT {
			return unsup(n.Kind)
		}
		i, _ := strconv.ParseFloat(n.Value, 64)
		return i, nil
	case *ast.UnaryExpr:
		x, err := eval(n.X)
		if err != nil {
			return 0, err
		}
		switch n.Op {
		case token.SUB:
			return -x, nil
		default:
			return unsup(n.Op)
		}
	case *ast.BinaryExpr:
		x, err := eval(n.X)
		if err != nil {
			return 0, err
		}
		y, err := eval(n.Y)
		if err != nil {
			return 0, err
		}
		switch n.Op {
		case token.ADD:
			return x + y, nil
		case token.SUB:
			return x - y, nil
		case token.MUL:
			return x * y, nil
		case token.QUO:
			return x / y, nil
		case token.XOR: // Use this to mean exponent rather than xor
			return math.Pow(x, y), nil
		case token.EQL: // ==
			if x == y {
				return 1, nil
			}
			return 0, nil
		case token.LSS: // <
			if x < y {
				return 1, nil
			}
			return 0, nil
		case token.GTR: // >
			if x > y {
				return 1, nil
			}
			return 0, nil
		case token.NEQ: // !=
			if x != y {
				return 1, nil
			}
			return 0, nil
		case token.LEQ: // <=
			if x <= y {
				return 1, nil
			}
			return 0, nil
		case token.GEQ: // >=
			if x >= y {
				return 1, nil
			}
			return 0, nil
		default:
			return unsup(n.Op)
		}
	case *ast.ParenExpr:
		return eval(n.X)
	case *ast.CallExpr:
		var err error
		fun := n.Fun.(*ast.Ident).Name
		args := make([]float64, len(n.Args))
		for i, a := range n.Args {
			args[i], err = eval(a)
			if err != nil {
				return 0, err
			}
		}
		switch fun {
		case "IF":
			if len(args) != 3 {
				return argErr(fun, len(args), 3)
			}
			if args[0] != 0. && args[0] != 1 {
				return notBoolErr(args[0])
			}
			if args[0] == 1 {
				return args[1], nil
			}
			return args[2], nil
		case "LN":
			if len(args) != 1 {
				return argErr(fun, len(args), 1)
			}
			return math.Log(args[0]), nil
		default:
			return unsup(fun)
		}
	}
	return unsup(reflect.TypeOf(tree))
}

// processVal goes through the database to find all of the expressions, checking each one
// to see if it matches the variable name varName and evaluating it if it does.
func (db *DB) processVal(val reflect.Value, varName string) *unit.Unit {
	// IsVariable returns whether the given expression matches the given variable name.
	isVariable := func(names []string) bool {
		for _, n := range names {
			if varName == n {
				return true
			}
		}
		return false
	}

	val = reflect.Indirect(val)
	if !val.IsValid() || !val.CanInterface() {
		return nil
	}
	t := val.Type()
	if t.Name() == "Expression" {
		expr := val.Interface().(Expression)
		e := expr.decode()
		if isVariable(e.names) {
			return db.evalExpr(expr)
		}
	} else if t.Name() == "Param" {
		p := val.Interface().(Param)
		if isVariable(p.varNames()) {
			return p.process(db)
		}
	} else if t.Name() == "InputTable" {
		it := val.Interface().(InputTable)
		if v, ok := it.processParameter(varName, db); ok {
			return v
		}
	}
	switch t.Kind() {
	case reflect.Struct:
		for i := 0; i < val.NumField(); i++ {
			if result := db.processVal(val.Field(i), varName); result != nil {
				return result
			}
		}
	case reflect.Array, reflect.Slice:
		for j := 0; j < val.Len(); j++ {
			if result := db.processVal(val.Index(j), varName); result != nil {
				return result
			}
		}
	}
	return nil
}

// GetVariableValue finds the desired variable in the database, evaluates it,
// and returns the result.
func (db *DB) GetVariableValue(varName string) *unit.Unit {
	// check if value has already been calculated
	if result, ok := db.processedParameters[varName]; ok {
		return result
	}

	// Otherwise, find the variable and evaluate it.
	result := db.processVal(reflect.ValueOf(db.Data), varName)
	if result == nil {
		panic("Couldn't find variable " + varName)
	}
	return result
}

var variableExp *regexp.Regexp

func init() {
	// The GREET database identifies variables as text between brackets (e.g., "[variable]").
	// This regexp will find them.
	variableExp = regexp.MustCompile(`\[(.*?)\]`)
}

// evalExpr evaluates an expression from the GREET database
func (db *DB) evalExpr(exp Expression, extraNames ...string) *unit.Unit {
	e := exp.decode()

	// check if value has already been calculated
	for _, n := range e.names {
		if value, ok := db.processedParameters[n]; ok {
			return value
		}
	}

	variables := variableExp.FindAllString(e.val, -1)
	for _, varName := range variables {
		val := db.GetVariableValue(strings.TrimRight(strings.TrimLeft(varName, "["), "]"))
		e.val = strings.Replace(e.val, varName,
			fmt.Sprintf("%v", val.Value()), -1)
	}
	// `if` causes a problem, so change it to `IF`
	e.val = strings.Replace(e.val, "if", "IF", -1)
	val, err := parseAndEval(e.val)
	if err != nil {
		panic(fmt.Errorf("expr: '%v'\nerror: %v", exp, err))
	}
	value := addUnits(val, e.units)

	// store value quick retrieval later
	for _, n := range e.names {
		if n != "" {
			db.processedParameters[n] = value
		}
	}
	for _, n := range extraNames {
		if n != "" {
			db.processedParameters[n] = value
		}
	}
	return value
}

// addUnits creates a Unit from the value and the units from GREET.net.
// GREET2015 has a new "feature" where the unit label can be in either SI
// or british units but the corresponding value is always in SI units.
func addUnits(val float64, units string) *unit.Unit {
	var value *unit.Unit
	switch units { // figure out units
	case "distance", "m", "mi", "km":
		value = unit.New(val, unit.Meter)
	case "area":
		value = unit.New(val, unit.Meter2)
	case "mass", "kg", "t", "g", "ton", "lb", "mg", "bushel": // presumably a bushel = 56 pounds
		value = unit.New(val, unit.Kilogram)
	case "time":
		value = unit.New(val, unit.Second)
	case "energy", "J", "Btu", "mmBtu", "kWh":
		value = unit.New(val, unit.Joule)
	case "power", "W", "hp":
		value = unit.New(val, unit.Watt)
	case "volume", "volume_solid_gas", "gal", "bu": // bu=bushel
		value = unit.New(val, unit.Meter3)
	case "temperature":
		value = unit.New(val, fahrenheight)
	case "percentage", "ratio_group", "unitless", "efficiency", "%", "", "gal/bushel":
		value = unit.New(val, unit.Dimless) // dimensionless
	case "concentration":
		value = unit.New(val*1.e-6, unit.Dimless) // ppm
	case "currency":
		value = unit.New(val, currency)
	case "date":
		value = unit.New(val, year)
	case "separative_work":
		value = unit.New(val, separativeWork)
	case "pressure":
		value = unit.New(val, weirdPressureUnits)
	case "fuel_consumption":
		value = unit.New(val, unit.Meter2) // m3/m
	case "fuel_economy", "mi/gal":
		value = unit.New(val, metersNeg2) // Meters travelled per m3 fuel
	case "energy_intensity", "temp", "J/(kg m)", "Btu/(mi ton)", "Btu/(ton mi)":
		value = unit.New(val, joulesPerMPerKg)
	case "heating_value_mass", "Btu/ton", "Btu/lb", "mmBtu/ton", "J/kg":
		value = unit.New(val, joulesPerKg)
	case "heating_value_volume", "Btu/gal", "mmBtu/gal", "Btu/ft^3", "J/m^3":
		value = unit.New(val, joulesPerM3)
	case "density", "g/gal", "g/ft^3", "kg/m^3":
		value = unit.New(val, unit.KilogramPerMeter3)
	case "yield":
		value = unit.New(val, unit.Kilogram)
	case "acre_yield":
		value = unit.New(val, kgPerMeter2)
	case "corn_yield":
		value = unit.New(val, kgPerMeter2)
	case "bagasse_yield":
		value = unit.New(val, kgPerMeter2)
	case "ethanol_yield", "ethannol_yield":
		value = unit.New(val, meter3PerKg)
	case "nutrient_add":
		value = unit.New(val, nil) // mass per mass
	case "energy_density":
		value = unit.New(val, joulesPerKg)
	case "emission_factor", "kg/J", "g/kWh", "g/mmBtu", "ng/J", "g/Btu", "mg/Btu", "kg/mmBtu", "mg/mmBtu", "g/Wh", "g/MJ":
		value = unit.New(val, kgPerJoule) // kg per J fuel burned
	case "emission_factor_grams":
		value = unit.New(val, nil) // mass per mass
	case "vehicle_emission_factor", "g/mi", "kg/m":
		value = unit.New(val, kgPerMeter) // meters traveled per kg fuel burned
	case "vehicle_energy", "Btu/mi", "Wh/mi", "J/m":
		value = unit.New(val, joulesPerMeter) // meters traveled per kg fuel burned
	case "energy_use":
		value = unit.New(val, nil) // J/J
	case "speed", "mi/h":
		value = unit.New(val, unit.MeterPerSecond)
	case "market_value_volume", "$/m^3":
		value = unit.New(val, dollarsPerM3)
	case "market_value_mass", "$/lb", "$/kg":
		value = unit.New(val, dollarsPerKg)
	case "market_value_energy", "$/J":
		value = unit.New(val, dollarsPerJoule)
	case "$":
		value = unit.New(val, dollars)
	case "joules/swu":
		value = unit.New(val, joulesPerSwu)
	case "SWU":
		value = unit.New(val, separativeWork)
	case "loss_rate_per_day":
		value = unit.New(val, perSecond)
	case "horsepower_factor", "W/g", "hp/ton":
		value = unit.New(val, wattsPerKg)
	case "ppm":
		value = unit.New(val*1.0e-6, unit.Dimless)
	default:
		panic(fmt.Sprintf("Unknown unit %v", units))
	}
	return value
}

func unsup(i interface{}) (float64, error) {
	return 0, fmt.Errorf("%v unsupported", i)
}

func argErr(fun string, nArgs, requiredNArgs int) (float64, error) {
	return 0, fmt.Errorf("Function %v got %v arguments but "+
		"requires %v", fun, nArgs, requiredNArgs)
}

func notBoolErr(i float64) (float64, error) {
	return 0, fmt.Errorf("Boolean value %v should be 0 or 1 but is not.", i)
}
