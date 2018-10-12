package greet

import (
	"math"
	"testing"

	"github.com/ctessum/unit"
)

func TestEvaluator(t *testing.T) {

	type testResult struct {
		test   string
		result interface{}
	}

	var tests = []testResult{
		{"(1+3)*7", 28.},                             // 28, example from task description.
		{"1+3*7", 22.},                               // 22, shows operator precedence.
		{"7", 7.},                                    // 7, a single literal is a valid expression.
		{"7/3", 7. / 3.},                             // non-integer math.
		{"7.3", 7.3},                                 // floating point.
		{"7^3", 343.},                                // power.
		{"go", "1:1: expected operand, found 'go'"},  // a valid keyword, not valid in an expression.
		{"3@7", "1:2: illegal character U+0040 '@'"}, // error message is "illegal character."
		{"", "1:1: expected operand, found 'EOF'"},   // EOF seems a reasonable error message.
		{"0.1>0.0", 1.},                              // 1
		{"0.0>=0.1", 0.},                             // 0
		{"1.0 == 1.0", 1.},                           //1
		{"5<3", 0.},                                  // 0
		{"3<=3", 1.},                                 // 1
		{"IF(13>3,3*2,5*9)", 6.},                     // 6
		{"IF(13<3,3*2,5*9)", 45.},                    // 45
		{"LN(15)", 2.70805020110221},                 // 2.7080502011
		{"IF(2.,1,2)", "Boolean value 2 should be 0 or 1 but is not."},
		{"IF(1==1,1,2,3)", "Function IF got 4 arguments but requires 3"},
		{"LN()", "Function LN got 0 arguments but requires 1"},
		{"-1.3 * 2", -2.6},
		{"=0.3", 0.3},
	}

	for _, exp := range tests {
		var testResult interface{}
		if r, err := parseAndEval(exp.test); err == nil {
			testResult = r
		} else {
			testResult = err.Error()
		}
		passFail := "Pass"
		if testResult != exp.result {
			switch testResult.(type) {
			case float64:
				if math.Abs(testResult.(float64))-exp.result.(float64) >
					exp.result.(float64)*1.e-8 {
					t.Fail()
					passFail = "FAIL"
				}
			default:
				t.Fail()
				passFail = "FAIL"
			}
		}
		t.Logf("%s: '%v' equals '%v' and should equal '%v'", passFail, exp.test,
			testResult, exp.result)
	}
}

func TestExpression(t *testing.T) {
	type testResult struct {
		test   Expression
		result float64
	}

	db := new(DB)
	db.processedParameters = make(map[string]*unit.Unit)

	var tests = []testResult{
		{"16;0;True;emission_factor;;55861662161;;;", 16},
		{"0;66;False;emission_factor;;55861664232;;;", 66},
		{"0.000000000017559520445384246;0;True;emission_factor;;fuel_27tech_5_real_2010_fact_5;;;",
			0.000000000017559520445384246},
		{"0.000000000017559259970924515;0;True;emission_factor;;fuel_27tech_6_real_2010_fact_5;;;",
			0.000000000017559259970924515},
		{"[fuel_27tech_5_real_2010_fact_5]*[55861662161]/100;0;True;emission_factor;;55979022808;;;",
			0.000000000017559520445384246 * 16 / 100.}, // BC To
		{"[fuel_27tech_5_real_2010_fact_5]*[55861664232]/100;0;True;emission_factor;;55979353745;;;",
			0.000000000017559520445384246 * 66 / 100.}, // POC To
		{"[fuel_27tech_6_real_2010_fact_5]*[55861662161]/100;0;True;emission_factor;;55979022822;;;",
			0.000000000017559259970924515 * 16 / 100.}, // BC From
		{"[fuel_27tech_6_real_2010_fact_5]*[55861664232]/100;0;True;emission_factor;;55979359819;;;",
			0.000000000017559259970924515 * 66 / 100.}, // POC From
	}

	for _, exp := range tests {
		result := db.evalExpr(exp.test)
		passFail := "Pass"
		if result.Value() != exp.result {
			passFail = "FAIL"
			t.Fail()
		}
		t.Logf("%s: '%v' equals '%g' and should equal '%v'",
			passFail, exp.test, result, exp.result)
	}
}
