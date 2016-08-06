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

// This file contains the methods to read CDF variable data.

package cdf

import (
	"encoding/binary"
	"errors"
	"io"
)

// A reader is an object that can read values from a CDF file.
type Reader interface {
	// Read reads len(values.([]T)) elements from the underlying file into values.
	//
	// Values must be a slice of int{8,16,32} or float{32,64},
	// corresponding to the type of the variable, with one
	// exception: A variable of NetCDF type CHAR must be read into
	// a []byte.  Read returns the number of elements actually
	// read.  if n < len(values.([]T)), err will be set.
	Read(values interface{}) (n int, err error)

	// Zero returns a slice of the appropriate type for Read
	// if n < 0, the slice will be of the length
	// that can be read contiguously.
	Zero(n int) interface{}
}

// A writer is an object that can write values to a CDF file.
type Writer interface {
	// Write writes len(values.([]T)) elements from values to the underlying file.
	//
	// Values must be a slice of int{8,16,32} or float{32,64} or a
	// string, according to the type of the variable.  if n <
	// len(values.([]T)), err will be set.
	Write(values interface{}) (n int, err error)
}

// Create a reader that starts at the corner begin, ends at end.  If begin is nil,
// it defaults to the origin (0, 0, ...).  If end is nil, it defaults
// to the f.Header.Lengths(v).
func (f *File) Reader(v string, begin, end []int) Reader { return f.strider(v, begin, end) }

// Create a writer that starts at the corner begin, ends at end.  If begin is nil,
// it defaults to the origin (0, 0, ...).  If end is nil and the variable is
// a record variable, writing can proceed past EOF and the underlying file will be extended.
func (f *File) Writer(v string, begin, end []int) Writer { return f.strider(v, begin, end) }

func (f *File) strider(v string, begin, end []int) interface {
	Reader
	Writer
} {
	vv := f.Header.varByName(v)
	if vv == nil {
		return nil
	}

	if begin != nil && len(begin) != len(vv.dim) {
		panic("invalid begin index vector")
	}

	if end != nil && len(end) != len(vv.dim) {
		panic("invalid end index vector")
	}

	var b, e, sz, sk int64

	if begin != nil {
		b = vv.offsetOf(begin)
	} else {
		b = vv.begin
	}

	if end != nil {
		e = vv.offsetOf(end) + int64(vv.dtype.storageSize())
	} else if !vv.isRecordVariable() {
		l := make([]int, len(vv.lengths))
		for i := range l {
			l[i] = vv.lengths[i] - 1
		}
		e = vv.offsetOf(l) + int64(vv.dtype.storageSize())
	}

	if vv.isRecordVariable() {
		sz = vv.strides[0] // vsize
		sk = vv.strides[1] // slabsize
	} else {
		sz = e - b
		sk = e - b
	}

	switch vv.dtype {
	case _BYTE, _CHAR:
		return &uint8strider{f.rw, b, e, sz, sk, b}
	case _SHORT:
		return &int16strider{f.rw, b, e, sz, sk, b}
	case _INT:
		return &int32strider{f.rw, b, e, sz, sk, b}
	case _FLOAT:
		return &float32strider{f.rw, b, e, sz, sk, b}
	case _DOUBLE:
		return &float64strider{f.rw, b, e, sz, sk, b}
	}
	panic("invalid variable data type")
}

type strider struct {
	rw                 ReaderWriterAt
	begin, end         int64
	stripesize, stride int64
	curr               int64
}

func (r *strider) relOffs(elemsz int) int64 {
	s := (r.curr - r.begin) / r.stride // stripe number
	e := (r.curr - r.begin) % r.stride // offset within stripe
	nn := (s * r.stripesize) + e
	nn /= int64(elemsz)
	return nn
}

func (r *strider) Read(p []byte) (n int, err error) {
	se := (r.curr - r.begin) / r.stride // stripe number
	se = r.begin + se*r.stride          // stripe begin
	se += r.stripesize                  // stripe end

	for len(p) > 0 {
		nn := int64(len(p))
		if r.curr+nn > se {
			nn = se - r.curr
		}
		if r.end > 0 && r.curr+nn > r.end {
			nn = r.end - r.curr
		}

		nr, err := r.rw.ReadAt(p[:nn], r.curr)
		r.curr += int64(nr)
		n += nr
		p = p[nr:]
		if r.curr == se {
			r.curr += r.stride - r.stripesize
			se += r.stride
		}
		if err != nil {
			return n, err
		}
		if r.curr >= r.end {
			return n, io.EOF
		}
	}

	return n, nil
}

func (r *strider) Write(p []byte) (n int, err error) {
	se := (r.curr - r.begin) / r.stride // stripe number
	se = r.begin + se*r.stride          // stripe begin
	se += r.stripesize                  // stripe end

	for len(p) > 0 {
		nn := int64(len(p))
		if r.curr+nn > se {
			nn = se - r.curr
		}
		if r.end > 0 && r.curr+nn > r.end {
			nn = r.end - r.curr
		}

		nr, err := r.rw.WriteAt(p[:nn], r.curr)
		r.curr += int64(nr)
		n += nr
		p = p[nr:]
		if r.curr == se {
			r.curr += r.stride - r.stripesize
			se += r.stride
		}
		if err != nil {
			return n, err
		}
		if r.end > 0 && r.curr >= r.end {
			return n, io.EOF
		}
	}

	return n, nil
}

func (r *strider) readElems(elemsz int, values interface{}) (int, error) {
	nn := r.relOffs(elemsz)
	err := binary.Read(r, binary.BigEndian, values)
	return int(r.relOffs(elemsz) - nn), err
}

func (r *strider) writeElems(elemsz int, values interface{}) (int, error) {
	nn := r.relOffs(elemsz)
	err := binary.Write(r, binary.BigEndian, values)
	return int(r.relOffs(elemsz) - nn), err
}

var badValueType = errors.New("value type mismatch")

type uint8strider strider
type int16strider strider
type int32strider strider
type float32strider strider
type float64strider strider

func (r *uint8strider) Read(values interface{}) (n int, err error) {
	if _, ok := values.([]uint8); !ok {
		return 0, badValueType
	}
	return (*strider)(r).readElems(1, values)
}

func (r *int16strider) Read(values interface{}) (n int, err error) {
	if _, ok := values.([]int16); !ok {
		return 0, badValueType
	}
	return (*strider)(r).readElems(2, values)
}

func (r *int32strider) Read(values interface{}) (n int, err error) {
	if _, ok := values.([]int32); !ok {
		return 0, badValueType
	}
	return (*strider)(r).readElems(4, values)
}

func (r *float32strider) Read(values interface{}) (n int, err error) {
	if _, ok := values.([]float32); !ok {
		return 0, badValueType
	}
	return (*strider)(r).readElems(4, values)
}

func (r *float64strider) Read(values interface{}) (n int, err error) {
	if _, ok := values.([]float64); !ok {
		return 0, badValueType
	}
	return (*strider)(r).readElems(8, values)
}

func (r *uint8strider) Write(values interface{}) (n int, err error) {
	if _, ok := values.([]uint8); !ok {
		if _, ok := values.(string); !ok {
			return 0, badValueType
		}
	}
	if str, ok := values.(string); ok {
		return (*strider)(r).writeElems(1, []byte(str))
	} else {
		return (*strider)(r).writeElems(1, values)
	}
}

func (r *int16strider) Write(values interface{}) (n int, err error) {
	if _, ok := values.([]int16); !ok {
		return 0, badValueType
	}
	return (*strider)(r).writeElems(2, values)
}

func (r *int32strider) Write(values interface{}) (n int, err error) {
	if _, ok := values.([]int32); !ok {
		return 0, badValueType
	}
	return (*strider)(r).writeElems(4, values)
}

func (r *float32strider) Write(values interface{}) (n int, err error) {
	if _, ok := values.([]float32); !ok {
		return 0, badValueType
	}
	return (*strider)(r).writeElems(4, values)
}

func (r *float64strider) Write(values interface{}) (n int, err error) {
	if _, ok := values.([]float64); !ok {
		return 0, badValueType
	}
	return (*strider)(r).writeElems(8, values)
}

func (r *uint8strider) Zero(n int) interface{} {
	if n < 0 {
		n = int(r.stripesize)
	}
	return make([]uint8, n)
}

func (r *int16strider) Zero(n int) interface{} {
	if n < 0 {
		n = int(r.stripesize / 2)
	}
	return make([]int16, n)
}

func (r *int32strider) Zero(n int) interface{} {
	if n < 0 {
		n = int(r.stripesize / 4)
	}
	return make([]int32, n)
}

func (r *float32strider) Zero(n int) interface{} {
	if n < 0 {
		n = int(r.stripesize / 4)
	}
	return make([]float32, n)
}
func (r *float64strider) Zero(n int) interface{} {
	if n < 0 {
		n = int(r.stripesize / 8)
	}
	return make([]float64, n)
}
