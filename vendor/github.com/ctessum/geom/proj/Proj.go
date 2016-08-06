package proj

import (
	"fmt"
	"math"
	"reflect"
	"strings"

	"github.com/gonum/floats"
)

// A Transformer takes input coordinates and returns output coordinates and an error.
type Transformer func(X, Y float64) (x, y float64, err error)

// A TransformerFunc creates forward and inverse Transformers from a projection.
type TransformerFunc func(*SR) (forward, inverse Transformer, err error)

var projections map[string]TransformerFunc

// SR holds information about a spatial reference (projection).
type SR struct {
	Name, Title                string
	SRSCode                    string
	DatumCode                  string
	Rf                         float64
	Lat0, Lat1, Lat2, LatTS    float64
	Long0, Long1, Long2, LongC float64
	Alpha                      float64
	X0, Y0, K0, K              float64
	A, A2, B, B2               float64
	Ra                         bool
	Zone                       float64
	UTMSouth                   bool
	DatumParams                []float64
	ToMeter                    float64
	Units                      string
	FromGreenwich              float64
	NADGrids                   string
	Axis                       string
	local                      bool
	sphere                     bool
	Ellps                      string
	EllipseName                string
	Es                         float64
	E                          float64
	Ep2                        float64
	DatumName                  string
	NoDefs                     bool
	datum                      *datum
	Czech                      bool
}

// NewSR initializes a SR object and sets fields to default values.
func NewSR() *SR {
	p := new(SR)
	// Initialize floats to NaN.
	v := reflect.ValueOf(p).Elem()
	for i := 0; i < v.NumField(); i++ {
		f := v.Field(i)
		ft := f.Type().Kind()
		if ft == reflect.Float64 {
			f.SetFloat(math.NaN())
		}
	}
	p.ToMeter = 1.
	return p
}

func registerTrans(proj TransformerFunc, names ...string) {
	if projections == nil {
		projections = make(map[string]TransformerFunc)
	}
	for _, n := range names {
		projections[strings.ToLower(n)] = proj
	}
}

// Transformers returns forward and inverse transformation functions for
// this projection.
func (sr *SR) Transformers() (forward, inverse Transformer, err error) {
	t, ok := projections[strings.ToLower(sr.Name)]
	if !ok {
		err = fmt.Errorf("in proj.Proj.TransformFuncs, could not find "+
			"transformer for %s", sr.Name)
		return
	}
	forward, inverse, err = t(sr)
	return
}

// Equal determines whether spatial references sr and sr2 are equal to within ulp
// floating point units in the last place.
func (sr *SR) Equal(sr2 *SR, ulp uint) bool {
	v1 := reflect.ValueOf(sr).Elem()
	v2 := reflect.ValueOf(sr2).Elem()
	return equal(v1, v2, ulp)
}

// equal determines whether two values are equal to each other within ulp
func equal(v1, v2 reflect.Value, ulp uint) bool {
	for i := 0; i < v1.NumField(); i++ {
		f1 := v1.Field(i)
		f2 := v2.Field(i)
		ft := f1.Type().Kind()
		switch ft {
		case reflect.Float64:
			fv1 := f1.Float()
			fv2 := f2.Float()
			if math.IsNaN(fv1) != math.IsNaN(fv2) {
				return false
			}
			if !math.IsNaN(fv1) && !floats.EqualWithinULP(fv1, fv2, ulp) {
				return false
			}
		case reflect.Int:
			if f1.Int() != f2.Int() {
				return false
			}
		case reflect.Bool:
			if f1.Bool() != f2.Bool() {
				return false
			}
		case reflect.Ptr:
			if !equal(reflect.Indirect(f1), reflect.Indirect(f2), ulp) {
				return false
			}
		case reflect.String:
			if f1.String() != f2.String() {
				return false
			}
		case reflect.Slice:
			for i := 0; i < f1.Len(); i++ {
				if !floats.EqualWithinULP(f1.Index(i).Float(), f2.Index(i).Float(), ulp) {
					return false
				}
			}
		default:
			panic(fmt.Errorf("unsupported type %s", ft))
		}
	}
	return true
}
