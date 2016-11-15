// Copyright ©2013 The gonum Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package unit

import "fmt"

// Unit is a type a value with generic SI units. Most useful for
// translating between dimensions, for example, by multiplying
// an acceleration with a mass to get a force. Please see the
// package documentation for further explanation.
type Unit struct {
	dimensions Dimensions // Map for custom dimensions
	formatted  string
	value      float64
}

// New creates a new variable of type Unit which has the value
// specified by value and the dimensions specified by the
// base units struct. The value is always in SI Units.
//
// Example: To create an acceleration of 3 m/s^2, one could do
// myvar := CreateUnit(3.0, &Dimensions{unit.LengthDim: 1, unit.TimeDim: -2})
func New(value float64, d Dimensions) *Unit {
	u := &Unit{
		dimensions: make(map[Dimension]int),
		value:      value,
	}
	for key, val := range d {
		if val != 0 {
			u.dimensions[key] = val
		}
	}
	return u
}

// Clone makes a copy of a unit.
func (u *Unit) Clone() *Unit {
	o := &Unit{
		dimensions: make(map[Dimension]int),
		value:      u.value,
	}
	for key, val := range u.dimensions {
		if val != 0 {
			o.dimensions[key] = val
		}
	}
	return o
}

// DimensionsMatch checks if the dimensions of two *Units are the same.
func DimensionsMatch(a, b *Unit) bool {
	aUnit := a
	bUnit := b
	if len(aUnit.dimensions) != len(bUnit.dimensions) {
		return false
	}
	for key, val := range aUnit.dimensions {
		if bUnit.dimensions[key] != val {
			return false
		}
	}
	return true
}

// Matches checks if the two sets of dimensions are the same.
func (d Dimensions) Matches(d2 Dimensions) bool {
	if len(d) != len(d2) {
		return false
	}
	for key, val := range d {
		if d2[key] != val {
			return false
		}
	}
	return true
}

// operate loops through u and applies function f,
// ignoring nil values.
func operateIgnore(f func(*Unit, *Unit, int), u []*Unit) *Unit {
	if len(u) == 0 {
		return nil
	}
	var o *Unit
	for i := 0; i < len(u); i++ {
		if u[i] == nil {
			continue
		}
		if o == nil {
			o = u[i].Clone()
		} else {
			uu := u[i]
			f(o, uu, i)
		}
	}
	return o
}

// operateSubDiv loops through u and applies function f,
// panicing on nil values.
func operatePanic(f func(*Unit, *Unit, int), u []*Unit) *Unit {
	if len(u) == 0 {
		return nil
	}
	if u[0] == nil {
		panic("Argument 0 is nil")
	}
	o := u[0].Clone()
	for i := 1; i < len(u); i++ {
		if u[i] == nil {
			panic(fmt.Errorf("Argument %d is nil", i))
		}
		uu := u[i]
		f(o, uu, i)
	}
	return o
}

// Add adds the function arguments.
// It panics if the units of the arguments don't match.
// Any nil values are assumed to equal zero.
func Add(u ...*Unit) *Unit {
	return operateIgnore(func(o, uu *Unit, i int) {
		if !DimensionsMatch(o, uu) {
			panic(fmt.Errorf("Mismatched dimensions in addition: "+
				"the first non-nil argument has dimensions %s, whereas "+
				"argument %d has dimensions %s.",
				o.Dimensions(), i, uu.Dimensions()))
		}
		o.value += uu.value
	}, u)
}

// Add adds uu to u, modifying u instead of creating a copy.
func (u *Unit) Add(uu *Unit) {
	if !DimensionsMatch(u, uu) {
		panic(fmt.Errorf("Mismatched dimensions in addition: "+
			"the first argument has dimensions %s, whereas "+
			"the second argument has dimensions %s.",
			u.Dimensions(), uu.Dimensions()))
	}
	u.value += uu.value
}

// Sub subtracts the second-through-last function arguments
// from the first function argument.
// It panics if the units of the arguments don't match.
// Nil arguments cause a panic.
func Sub(u ...*Unit) *Unit {
	return operatePanic(func(o, uu *Unit, i int) {
		if !DimensionsMatch(o, uu) {
			panic(fmt.Errorf("Mismatched dimensions in subtraction: "+
				"the first non-nil argument has dimensions %s, whereas "+
				"argument %d has dimensions %s.",
				o.Dimensions(), i, uu.Dimensions()))
		}
		o.value -= uu.value
	}, u)
}

// Sub subtracts uu from u, modifying u instead of creating a copy.
func (u *Unit) Sub(uu *Unit) {
	if !DimensionsMatch(u, uu) {
		panic(fmt.Errorf("Mismatched dimensions in subtraction: "+
			"the first argument has dimensions %s, whereas "+
			"the second argument has dimensions %s.",
			u.Dimensions(), uu.Dimensions()))
	}
	u.value -= uu.value
}

// Negate multiplies the value by -1, returning
// a copy of the input argument.
func Negate(u *Unit) *Unit {
	uu := u.Clone()
	uu.value *= -1
	return uu
}

// Negate multiplies u by -1, modifying u instead of creating a copy.
func (u *Unit) Negate() {
	u.value *= -1
}

// Mul multiplies the function arguments, calculating
// the proper units for the result.
// Nil arguments cause a panic.
func Mul(u ...*Unit) *Unit {
	return operatePanic(func(o, uu *Unit, i int) {
		for key, val := range uu.dimensions {
			if d := o.dimensions[key]; d == -val {
				delete(o.dimensions, key)
			} else {
				o.dimensions[key] = d + val
			}
		}
		o.value *= uu.value
		o.formatted = ""
	}, u)
}

// Mul multiplies u by uu, modifying u instead of creating a copy.
func (u *Unit) Mul(uu *Unit) {
	for key, val := range uu.dimensions {
		if d := u.dimensions[key]; d == -val {
			delete(u.dimensions, key)
		} else {
			u.dimensions[key] = d + val
		}
	}
	u.value *= uu.value
	u.formatted = ""
}

// Div divides the function arguments with the first arguement
// as the numerator and the rest as the denominator, calculating
// the proper units for the result.
// Nil arguments cause a panic.
func Div(u ...*Unit) *Unit {
	return operatePanic(func(o, uu *Unit, i int) {
		for key, val := range uu.dimensions {
			if d := o.dimensions[key]; d == val {
				delete(o.dimensions, key)
			} else {
				o.dimensions[key] = d - val
			}
		}
		o.value /= uu.value
		o.formatted = ""
	}, u)
}

// Div divides u by uu, modifying u instead of creating a copy.
func (u *Unit) Div(uu *Unit) {
	for key, val := range uu.dimensions {
		if d := u.dimensions[key]; d == val {
			delete(u.dimensions, key)
		} else {
			u.dimensions[key] = d - val
		}
	}
	u.value /= uu.value
	u.formatted = ""
}

// Max returns the maximum among function arguments.
// It panics if the units of the arguments don't match.
// Any nil values are ignored.
func Max(u ...*Unit) *Unit {
	return operateIgnore(func(o, uu *Unit, i int) {
		if !DimensionsMatch(o, uu) {
			panic(fmt.Errorf("Mismatched dimensions in addition: "+
				"argument 0 has dimensions %s, whereas argument %d has dimensions %s.",
				o.Dimensions(), i, uu.Dimensions()))
		}
		if uu.value > o.value {
			o.value = uu.value
		}
	}, u)
}

// Min returns the minimum among function arguments.
// It panics if the units of the arguments don't match.
// Any nil values are ignored.
func Min(u ...*Unit) *Unit {
	return operateIgnore(func(o, uu *Unit, i int) {
		if !DimensionsMatch(o, uu) {
			panic(fmt.Errorf("Mismatched dimensions in addition: "+
				"argument 0 has dimensions %s, whereas argument %d has dimensions %s.",
				o.Dimensions(), i, uu.Dimensions()))
		}
		if uu.value < o.value {
			o.value = uu.value
		}
	}, u)
}

// Value return the raw value of the unit as a float64. Use of this
// method is, in general, not recommended, though it can be useful
// for printing. Instead, the From type of a specific dimension
// should be used to guarantee dimension consistency.
func (u *Unit) Value() float64 {
	return u.value
}

// Dimensions returns the dimenstions associated with the unit.
func (u *Unit) Dimensions() Dimensions {
	return u.dimensions
}

// Format makes Unit satisfy the fmt.Formatter interface. The unit is formatted
// with dimensions appended. If the power if the dimension is not zero or one,
// symbol^power is appended, if the power is one, just the symbol is appended
// and if the power is zero, nothing is appended. Dimensions are appended
// in order by symbol name with positive powers ahead of negative powers.
func (u *Unit) Format(fs fmt.State, c rune) {
	if u == nil {
		fmt.Fprint(fs, "<nil>")
	}
	switch c {
	case 'v':
		if fs.Flag('#') {
			fmt.Fprintf(fs, "&%#v", *u)
			return
		}
		fallthrough
	case 'e', 'E', 'f', 'F', 'g', 'G':
		s := "%"
		w, wOk := fs.Width()
		if wOk {
			s += fmt.Sprintf("%d", w)
		}
		p, pOk := fs.Precision()
		if pOk {
			s += fmt.Sprintf(".%d", p)
		}
		fmt.Fprintf(fs, s+string(c), u.value)
	default:
		fmt.Fprintf(fs, "%%!%c(*Unit=%g)", c, u)
		return
	}
	if u.formatted == "" && len(u.dimensions) > 0 {
		u.formatted = u.dimensions.String()
	}
	fmt.Fprintf(fs, " %s", u.formatted)
}

// Check checks whether u's dimensions match d, and returns an error if they
// don't.
func (u *Unit) Check(d Dimensions) error {
	if !u.Dimensions().Matches(d) {
		return fmt.Errorf("unit dimensions (%s) should be %s",
			u.Dimensions().String(), d.String())
	}
	return nil
}
