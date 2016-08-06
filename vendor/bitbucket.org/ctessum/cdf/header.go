// Copyright 2012 Luuk van Dijk. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// This file contains the header structure and related code.

package cdf

import (
	"bytes"
	"fmt"
	"io"
)

// A version of 1 indicates 32 bit offsets, a version of 2 indicates 64 bit offsets.
// All other versions, in particular V4 (which uses HDF as a backing store), are unsupported.
type version byte

const (
	_V1 version = iota + 1 // 32 bit offsets
	_V2                    // 64 bit offsets
)

// String renders v as "V1" or "V2" if valid, "<42>" if invalid.
func (v version) String() string {
	switch v {
	case _V1:
		return "V1"
	case _V2:
		return "V2"
	}
	return fmt.Sprintf("<%d>", byte(v))
}

// A datatype encodes the NetCDF data type of a variable or attribute.
type datatype int32

const (
	_BYTE datatype = iota + 1
	_CHAR
	_SHORT
	_INT
	_FLOAT
	_DOUBLE
)

// data type string and storage size tables.
var (
	dt2String      = [...]string{"", "BYTE", "CHAR", "SHORT", "INT", "FLOAT", "DOUBLE"}
	dt2StorageSize = [...]int{0, 1, 1, 2, 4, 4, 8}
)

// Valid returns whether d is one of the six defined types.
func (d datatype) valid() bool { return d >= _BYTE && d <= _DOUBLE }

// StorageSize returns the number of bytes occupied by an element of the datatype.
func (d datatype) storageSize() int {
	if d.valid() {
		return dt2StorageSize[d]
	}
	return 0
}

// Zero returns a slice of the proper type of length n,
// except for _CHAR, for which it returns the empty string.
func (d datatype) Zero(n int) interface{} {
	switch d {
	case _BYTE:
		return make([]uint8, n)
	case _CHAR:
		return ""
	case _SHORT:
		return make([]int16, n)
	case _INT:
		return make([]int32, n)
	case _FLOAT:
		return make([]float32, n)
	case _DOUBLE:
		return make([]float64, n)
	}
	return nil
}

// DataTypeFromValues maps the type of val to its corresponding datatype.
//
// The only valid dynamic types of val are 
// []uint8, string, []int16, []int32, []float32 or []float64.
// Any other type of val returns the zero (invalid) datatype).
func dataTypeFromValues(val interface{}) datatype {
	switch val.(type) {
	case []uint8:
		return _BYTE
	case string:
		return _CHAR
	case []int16:
		return _SHORT
	case []int32:
		return _INT
	case []float32:
		return _FLOAT
	case []float64:
		return _DOUBLE
	}
	return 0
}

// String renders the datatype as "BYTE", "CHAR", "SHORT", "INT", "FLOAT", "DOUBLE" or "<42>"
// if the type is invalid.
func (d datatype) String() string {
	if d.valid() {
		return dt2String[d]
	}
	return fmt.Sprintf("<%d>", int32(d))
}

// FillValue returns the data type's default fill value as per the spec.
func (d datatype) FillValue() interface{} {
	switch d {
	case _BYTE:
		return int8(-127)
	case _CHAR:
		return uint8(0)
	case _SHORT:
		return int16(-32767)
	case _INT:
		return int32(-2147483647)
	case _FLOAT:
		return float32(9.9692099683868690e+36) // \x7C \xF0 \x00 \x00 
	case _DOUBLE:
		return float64(9.9692099683868690e+36) // \x47 \x9E \x00 \x00 \x00 \x00
	}
	return nil
}

// round x up to the nearest multiple of 4.
func pad4(x int64) int64 { return (x + 3) &^ 3 }

// A NetCDF dimension as represented in the header
type dimension struct {
	name   string
	length int32
}

// An NetCDF global or variable attribute as represented in the header
type attribute struct {
	name   string
	dtype  datatype
	values interface{} // []uint8, string, []int16, []int32, []float32 or []float64
}

// Fprint writes a debug representation of the attribute in the form "[var]:name type = val"
// to w.  Long strings are truncated and suffixed with "...".
func (a *attribute) Fprint(w io.Writer, pfx string) {
	fmt.Fprintf(w, "%s:%s %s = ", pfx, a.name, a.dtype)
	switch a.dtype {
	case _CHAR:
		s := a.values.(string)
		if len(s) > 40 {
			s = s[:40] + "..."
		}
		fmt.Fprintf(w, "%#v", s)
	default:
		fmt.Fprintf(w, "%#v", a.values)
	}
}

// An NetCDF variable as represented in the header
type variable struct {
	// stored
	name  string
	dim   []int32 // indices into header.dim
	att   []attribute
	dtype datatype
	vsize int32 // set as per spec but not used by this library
	begin int64

	// computed
	lengths []int // header.dim[v.dim[i]].length

	// for a non-record variable, this is { nz*ny*nx*dsz, ny*nx*dsz, nx*dsz, dsz}
	// for a record variable this is { ny*nx*dsz, slabsize, nx*dsz, dsz }
	strides []int64
}

func (v *variable) isRecordVariable() bool { return len(v.lengths) > 0 && v.lengths[0] == 0 }
func (v *variable) vSize() int64           { return v.strides[0] }

func (v *variable) setComputed(dims []dimension) {
	v.lengths = make([]int, len(v.dim))
	for i, d := range v.dim {
		if d >= 0 && d < int32(len(dims)) {
			v.lengths[i] = int(dims[d].length)
		}
	}

	v.strides = make([]int64, len(v.dim)+1)
	v.strides[len(v.dim)] = int64(v.dtype.storageSize())
	for i := len(v.dim) - 1; i >= 0; i-- {
		v.strides[i] = int64(v.lengths[i]) * v.strides[i+1]
	}

	vsize := v.strides[0]
	if vsize == 0 && len(v.strides) > 1 {
		vsize = v.strides[1]
	}
	vsize = pad4(vsize)
	// the spec is ambiguous, it says s > 1<<32-4...
	// but the grammar says NON_NEG is a positive INT
	// and INT is a signed 32 bit big endian, so we'll set it it 0xffff/-1 if 
	if vsize > (1<<31 - 4) {
		v.vsize = -1
	} else {
		v.vsize = int32(vsize)
	}
}

func (v *variable) offsetOf(idx []int) int64 {
	o := v.begin
	for i, x := range idx {
		o += int64(x) * v.strides[i+1]
	}
	return o
}

// If the variable has a scalar attribute '_FillValue' of the same data type as the variable,
// it will be returned, otherwise the type's default fill value will be returned
func (v *variable) fillValue() interface{} {
	for i := range v.att {
		if v.att[i].name != "_FillValue" {
			continue
		}
		if v.att[i].dtype != v.dtype {
			break
		}
		switch vv := v.att[i].values.(type) {
		case []uint8:
			if len(vv) == 1 {
				return vv[0]
			}
		case string:
			if len(vv) == 1 {
				return vv[0]
			}
		case []int16:
			if len(vv) == 1 {
				return vv[0]
			}
		case []int32:
			if len(vv) == 1 {
				return vv[0]
			}
		case []float32:
			if len(vv) == 1 {
				return vv[0]
			}
		case []float64:
			if len(vv) == 1 {
				return vv[0]
			}
		default:
			panic("invalid attribute value type")
		}
		break // length != 1
	}
	return v.dtype.FillValue()
}

// A CDF file contains a header and a data section.
// The header defines the layout of the data section.
//
// The serialized header layout is specified by "The NetCDF Classic Format Specification"
// 	http://www.unidata.ucar.edu/software/netcdf/docs/classic_format_spec.html
//
// A header read with ReadHeader can not be modified.  A header created with NewHeader
// can be modified with AddVariable and AddAttribute until the call to Define.
//
// The NetCDF defined 'numrecs' field is ignored on reading and set to -1
// ('STREAMING') on writing of the header, but can be read and written
// separately.
type Header struct {
	version version
	dim     []dimension
	att     []attribute
	vars    []variable
}

// Find the index of the dimension named v, or return -1.
// Linear scan but unlikely to matter. 
func (h *Header) dimByName(v string) int {
	for i := range h.dim {
		if h.dim[i].name == v {
			return i
		}
	}
	return -1
}

// Find the the variable named v, or return nil.
// The returned pointer may be invalidated by a call to header.AddVariable.
// Linear scan but unlikely to matter. 
func (h *Header) varByName(v string) *variable {
	for i := range h.vars {
		if h.vars[i].name == v {
			return &h.vars[i]
		}
	}
	return nil
}

// Find the attribute named a in the variable named v
// or in the global variables if v == "". returns nil
// if there is no such attibute.
// Linear scan but unlikely to matter. 
func (h *Header) attrByName(v, a string) *attribute {
	attr := &h.att
	if v != "" {
		vv := h.varByName(v)
		if vv == nil {
			return nil
		}
		attr = &vv.att
	}
	for i := range *attr {
		if (*attr)[i].name == a {
			return &(*attr)[i]
		}
	}
	return nil
}

// Dimensions returns a slice with the names of the dimensions for variable v,
// all dimensions if v == "", or nil if v is not a valid variable.
//
// May panic on un-Check-ed headers.
func (h *Header) Dimensions(v string) []string {
	if v == "" {
		r := make([]string, len(h.dim))
		for i := range h.dim {
			r[i] = h.dim[i].name
		}
		return r
	}

	vv := h.varByName(v)
	if vv == nil {
		return nil
	}
	r := make([]string, len(vv.dim))
	for j, d := range vv.dim {
		r[j] = h.dim[d].name
	}
	return r
}

// Lengths returns a slice with the lengths of the dimensions for variable v,
// all dimensions if v == "", or nil if v is not a valid variable.
//
// May panic on un-Check-ed headers.
func (h *Header) Lengths(v string) []int {
	if v == "" {
		r := make([]int, len(h.dim))
		for i := range h.dim {
			r[i] = int(h.dim[i].length)
		}
		return r
	}

	vv := h.varByName(v)
	if vv == nil {
		return nil
	}
	return vv.lengths
}

// ZeroValue returns a zeroed slice of the type of the variable v of length n.
// If the named variable does not exist in h, Zero returns nil.
// For type CHAR, Zero returns an empty string.
func (h *Header) ZeroValue(v string, n int) interface{} {
	vv := h.varByName(v)
	if vv == nil {
		return nil
	}
	return vv.dtype.Zero(n)
}

// Return the fill value for the variable v. 
// If the variable has a scalar attribute '_FillValue' of the same data type as the variable,
// it will be used, otherwise the type's default fill value will be used.
func (h *Header) FillValue(v string) interface{} {
	vv := h.varByName(v)
	if vv == nil {
		return nil
	}
	return vv.fillValue()
}

// IsRecordVariable returns true iff a variable named v exists and its outermost dimension
// is the header's (unique) record dimension.
func (h *Header) IsRecordVariable(v string) bool {
	vv := h.varByName(v)
	if vv == nil {
		return false
	}
	return vv.isRecordVariable()
}

// Variables returns a slice with the names of all variables defined in the header.
func (h *Header) Variables() []string {
	r := make([]string, len(h.vars))
	for i := range h.vars {
		r[i] = h.vars[i].name
	}
	return r
}

// Variables returns a slice with the names of all attributes defined in the header,
// for variable v.  If v is the empty string, returns all global attributes.
func (h *Header) Attributes(v string) []string {
	attr := &h.att
	if v != "" {
		vv := h.varByName(v)
		if vv == nil {
			return nil
		}
		attr = &vv.att
	}
	r := make([]string, len(*attr))
	for i := range *attr {
		r[i] = (*attr)[i].name
	}
	return r
}

// GetAttribute returns the value of the attribute a of variable
// v or the global attribute a if v == "".  The returned
// value is of type  []uint8, string, []int16, []int32,
// []float32 or []float64 and should not be modified by the caller,
// as it is shared by all callers.
func (h *Header) GetAttribute(v, a string) interface{} {
	attr := h.attrByName(v, a)
	if attr == nil {
		return nil
	}
	return attr.values
}

// Newheader constructs a new CDF header.
//
// dims and lengths specify the names and lengths of the dimensions.
// Invalid dimension or size specifications, repeated dimension names,
// as well as the occurence of more than 1 record dimension (size == 0) lead to panics.
//
// Until the call to h.Define() the version of the header will not be set, and the header will mutable,
// meaning it can be modified by AddAttribute or AddVariable.
func NewHeader(dims []string, lengths []int) *Header { return newHeader(0, dims, lengths) }

// newheader constructs a new CDF header of the specified version.
func newHeader(v version, dims []string, lengths []int) *Header {
	if len(dims) != len(lengths) {
		panic("dims and sizes should be of same length")
	}

	recdim := -1
	for i, s := range dims {
		if lengths[i] < 0 {
			panic("invalid dimension length")
		}
		if lengths[i] == 0 {
			if recdim == -1 {
				recdim = i
			} else {
				panic("multiple record dimensions")
			}
		}
		for j, t := range dims {
			if i != j && s == t {
				panic("duplicate dimension name: " + s)
			}
		}
	}

	h := &Header{version: v, dim: make([]dimension, len(dims))}
	for i, v := range dims {
		h.dim[i] = dimension{name: v, length: int32(lengths[i])}
	}

	return h
}

// AddVariable adds a variable of given type with the named dimensions to the header.
//
// Use of an existing variable name, or a nonexistent dimension name leads to a panic,
// as does use of the record dimension for any other than the first.
//
// The datatype is determined from the dynamic type of val, which may be
// one of []uint8, string, []int16, []int32, []float32 or []float64.  Any
// other type will lead to a panic.  The contents of val are ignored.
//
// The header must be mutable, i.e. created by NewHeader, not by ReadHeader.
func (h *Header) AddVariable(v string, dims []string, val interface{}) {
	if !h.isMutable() {
		panic("cannot call AddVariable on an immutable header")
	}

	if h.varByName(v) != nil {
		panic("repeated add of variable " + v)
	}

	d := dataTypeFromValues(val)
	if !d.valid() {
		panic("invalid attribute value type")
	}

	dim := make([]int32, len(dims))
	for i, dd := range dims {
		d := h.dimByName(dd)
		if d < 0 {
			panic("invalid dimension")
		}
		if h.dim[d].length == 0 && i != 0 {
			panic("record dimension not outermost")
		}

		dim[i] = int32(d)
	}

	h.vars = append(h.vars, variable{name: v, dim: dim, dtype: d})
	h.vars[len(h.vars)-1].setComputed(h.dim)
}

// AddAttribute adds an attribute named a to a variable named v, or to the global attributes
// if v is the empty string.
//
// Use of a nonexistent variable name or an existent attribute name leads to a panic.
// The value can be of type []uint8, string, []int16, []int32, []float32 or []float64, and will be stored
// as NetCDF type  BYTE, CHAR, SHORT, INT, FLOAT, DOUBLE resp.
// The header must be mutable, i.e. created by NewHeader, not by ReadHeader.
func (h *Header) AddAttribute(v, a string, val interface{}) {
	if !h.isMutable() {
		panic("cannot call AddAttribute on an immutable header")
	}

	att := &h.att
	if v != "" {
		vv := h.varByName(v)
		if vv == nil {
			panic("no such variable")
		}
		att = &vv.att
	}
	for _, aa := range *att {
		if aa.name == a {
			panic("repeated add of attribute " + v + ":" + a)
		}
	}
	d := dataTypeFromValues(val)
	if !d.valid() {
		panic("invalid attribute value type")
	}
	*att = append(*att, attribute{name: a, dtype: d, values: val})
}

// String returns a summary dump of the header, suitable for debugging.
func (h *Header) String() string {
	var b bytes.Buffer
	fmt.Fprintf(&b, "version:%v\ndimensions:\n", h.version)
	for i := range h.dim {
		if h.dim[i].length == 0 { // the record dimension
			fmt.Fprintf(&b, "\t%s = UNLIMITED ;\n", h.dim[i].name)
		} else {
			fmt.Fprintf(&b, "\t%s = %d ;\n", h.dim[i].name, h.dim[i].length)
		}
	}

	fmt.Fprintf(&b, "variables:\n")
	for i := range h.vars {
		fmt.Fprintf(&b, "\t%s %s[", h.vars[i].name, h.vars[i].dtype)
		for j, d := range h.vars[i].dim {
			if j > 0 {
				fmt.Fprintf(&b, ", ")
			}
			if d < 0 || int(d) >= len(h.dim) {
				fmt.Fprintf(&b, "<invalid %d>", d)
				continue
			}
			fmt.Fprintf(&b, "%s", h.dim[d].name)
			if h.dim[d].length == 0 {
				fmt.Fprintf(&b, "*")
			}
		}
		fmt.Fprintf(&b, "] vsize:%d begin:%d\n", h.vars[i].vsize, h.vars[i].begin)
		for j := range h.vars[i].att {
			fmt.Fprintf(&b, "\t\t")
			h.vars[i].att[j].Fprint(&b, h.vars[i].name)
			fmt.Fprintf(&b, "\n")
		}
	}

	// global attributes
	for j := range h.att {
		fmt.Fprintf(&b, "\t")
		h.att[j].Fprint(&b, "")
		fmt.Fprintf(&b, "\n")
	}

	return b.String()
}

// Check verifies the integrity of the header:
//
// - at most one record dimension
//
// - no duplicate dimension names
//
// - no duplicate attribute names
//
// - no duplicate variable names
//
// - variable dimensions valid
//
// - only the first dimension can be the record dimension
//
// - offsets of non-variable records increasing, large enough and all before variable records
//
// - offset of variable records also increasing, large enough
func (h *Header) Check() (errs []error) {
	var x []string
	for i := range h.dim {
		if h.dim[i].length == 0 {
			x = append(x, h.dim[i].name)
		}
	}
	if len(x) > 1 {
		errs = append(errs, fmt.Errorf("multiple record dimensions: %v", x))
	}

	for i := range h.dim {
		for j := range h.dim {
			if i != j && h.dim[i].name == h.dim[j].name {
				errs = append(errs, fmt.Errorf("repeated dimension: %v", h.dim[i].name))
			}
		}
	}

	for i := range h.vars {
		for j := range h.vars {
			if i != j && h.vars[i].name == h.vars[j].name {
				errs = append(errs, fmt.Errorf("repeated variable: %s", h.vars[i].name))
			}
		}
	}

	for i := range h.att {
		for j := range h.att {
			if i != j && h.att[i].name == h.att[j].name {
				errs = append(errs, fmt.Errorf("repeated attribute :%s", h.att[i].name))
			}
		}
	}

	for v := range h.vars {
		for i := range h.vars[v].att {
			for j := range h.vars[v].att {
				if i != j && h.vars[v].att[i].name == h.vars[v].att[j].name {
					errs = append(errs, fmt.Errorf("repeated attribute %s:%s", h.vars[v].name, h.vars[v].att[i].name))
				}
			}
		}
	}

	d := int32(len(h.dim))
	for v := range h.vars {
		for i, x := range h.vars[v].dim {
			if x < 0 || x > d {
				errs = append(errs, fmt.Errorf("invalid dimension %s[%d] = %d", h.vars[v].name, i, x))
			}
			if h.dim[x].length == 0 && i != 0 {
				errs = append(errs, fmt.Errorf("non-outer record dimension %s[%d]", h.vars[v].name, i))
			}
		}
	}

	// check offsets increase in the right order and fit vsizes
	offs := pad4(h.size())

	for i := range h.vars {
		if !h.vars[i].isRecordVariable() {
			if h.vars[i].begin&3 != 0 || h.vars[i].begin < offs {
				errs = append(errs, fmt.Errorf("variable %s offset %d invalid", h.vars[i].name, h.vars[i].begin))
			}
			offs = h.vars[i].begin
			offs += pad4(h.vars[i].strides[0])
		}
	}

	for i := range h.vars {
		if h.vars[i].isRecordVariable() {
			if h.vars[i].begin&3 != 0 || h.vars[i].begin < offs {
				errs = append(errs, fmt.Errorf("variable %s offset %d invalid", h.vars[i].name, h.vars[i].begin))
			}
			offs = h.vars[i].begin
			offs += pad4(h.vars[i].strides[0])
		}
	}

	return
}

func (h *Header) fixRecordStrides() {
	recvars := 0
	var slabsize int64

	for i := range h.vars {
		if h.vars[i].strides[0] == 0 && len(h.vars[i].strides) > 1 {
			recvars++
			slabsize = h.vars[i].strides[1]
		}
	}

	// if there was just 1 recvar, slabsize has been set above, and does not require padding
	// otherwise recompute based on all of the vsizes of the record variables
	if recvars > 1 {
		slabsize = 0
		for i := range h.vars {
			if h.vars[i].strides[0] == 0 { // is record variable
				slabsize += pad4(h.vars[i].strides[1])
			}
		}
	}

	for i := range h.vars {
		if h.vars[i].strides[0] == 0 {
			// save the vsize in the [0] entry which is not used for indexing anyway
			h.vars[i].strides[0] = h.vars[i].strides[1]
			h.vars[i].strides[1] = slabsize
		}
	}
}

// DataStart returns the offset of the first variable.
func (h *Header) dataStart() int64 {
	if h.isMutable() {
		return pad4(h.size())
	}

	ds := h.vars[0].begin

	for i := range h.vars {
		if !h.vars[i].isRecordVariable() {
			ds = h.vars[i].begin
			break
		}
	}

	return ds
}

// Set the vars[*].begin fields starting at max(start, h.size)
// returns the values of the first and last offset.
// if there are no variables, last will be zero
func (h *Header) setOffsets(start int64) (first, last int64) {
	offs := h.size()
	if offs < start {
		offs = start
	}

	offs = pad4(offs)
	first = offs

	for i := range h.vars {
		if !h.vars[i].isRecordVariable() {
			h.vars[i].begin = offs
			last = offs
			offs += pad4(h.vars[i].vSize())
		}
	}

	for i := range h.vars {
		if h.vars[i].isRecordVariable() {
			h.vars[i].begin = offs
			last = offs
			offs += pad4(h.vars[i].vSize())
		}
	}

	return
}

// as long as the version is not set, this is a mutable header
func (h *Header) isMutable() bool { return h.version == 0 }

// Define makes a mutable header immutable by calculating the variable offsets and setting
// the version number to V1 or V2, depending on whether the layout requires 64-bit offsets or not.
func (h *Header) Define() {
	if !h.isMutable() {
		panic("cannot Define an immutable header")
	}

	h.fixRecordStrides()

	// version must be set before call to dataStart/setOffsets.  Theoretically
	// writing 64 bit offsets instead of 32 bit can affect the value of dataStart.
	h.version = _V2
	if _, last := h.setOffsets(h.dataStart()); last < (1 << 31) {
		h.version = _V1
	}
}

func (h *Header) slabs() (offs, size int64) {

	for i := range h.vars {
		if h.vars[i].isRecordVariable() {
			offs = h.vars[i].begin
			size = h.vars[i].strides[1] // slabsize
			break
		}
	}

	return
}

// numRecs computes the number or records from the filesize, returns the real number of records.
// For fsize < 0, returns -1.
func (h *Header) NumRecs(fsize int64) int64 {
	if fsize < 0 {
		return -1
	}

	offs, size := h.slabs()

	if size == 0 || fsize < offs {
		return 0
	}

	nr := (fsize - offs) / size
	return nr
}
