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

// This file contains the code to serialize cdf headers.

package cdf

import (
	"encoding/binary"
	"io"
)

var padding [4]byte // zeroes

// write an (int32, []byte) encoded string
func writeString(w io.Writer, s string) error {
	if err := binary.Write(w, binary.BigEndian, int32(len(s))); err != nil {
		return err
	}
	if _, err := w.Write([]byte(s)); err != nil {
		return err
	}
	p := 4 - len(s)&3
	if p < 4 {
		_, err := w.Write(padding[:p])
		return err
	}
	return nil
}

// used by writeHeader
func (d *dimension) writeTo(w io.Writer) error {
	if err := writeString(w, d.name); err != nil {
		return err
	}
	return binary.Write(w, binary.BigEndian, d.length)
}

// used by writeHeader
func (a *attribute) writeTo(w io.Writer) error {
	if err := writeString(w, a.name); err != nil {
		return err
	}

	// instead dtype field directly use a.values.(type) 
	if err := binary.Write(w, binary.BigEndian, dataTypeFromValues(a.values)); err != nil {
		return err
	}

	if !a.dtype.valid() {
		panic("invalid attribute data type for " + a.name)
	}

	if a.dtype == _CHAR {
		return writeString(w, a.values.(string))
	}

	var nelems int
	p := 4
	switch a.dtype {
	case _BYTE:
		nelems = len(a.values.([]uint8))
		p = 4 - nelems&3
	case _SHORT:
		nelems = len(a.values.([]int16))
		p = 4 - (2*nelems)&3
	case _INT:
		nelems = len(a.values.([]int32))
	case _FLOAT:
		nelems = len(a.values.([]float32))
	case _DOUBLE:
		nelems = len(a.values.([]float64))
	}
	if err := binary.Write(w, binary.BigEndian, int32(nelems)); err != nil {
		return err
	}
	if err := binary.Write(w, binary.BigEndian, a.values); err != nil {
		return err
	}

	if p < 4 {
		_, err := w.Write(padding[:p])
		return err
	}
	return nil
}

// used by writeHeader
func (v *variable) writeTo(w io.Writer, offs64 bool) error {
	if err := writeString(w, v.name); err != nil {
		return err
	}

	if err := binary.Write(w, binary.BigEndian, int32(len(v.dim))); err != nil {
		return err
	}
	for _, d := range v.dim {
		if err := binary.Write(w, binary.BigEndian, int32(d)); err != nil {
			return err
		}
	}
	var tag int32
	if len(v.att) > 0 {
		tag = 0xC
	}
	if err := binary.Write(w, binary.BigEndian, tag); err != nil {
		return err
	}
	if err := binary.Write(w, binary.BigEndian, int32(len(v.att))); err != nil {
		return err
	}

	for i := range v.att {
		if err := v.att[i].writeTo(w); err != nil {
			return err
		}
	}

	if !v.dtype.valid() {
		panic("invalid variable data type for " + v.name)
	}

	if err := binary.Write(w, binary.BigEndian, v.dtype); err != nil {
		return err
	}

	if err := binary.Write(w, binary.BigEndian, v.vsize); err != nil {
		return err
	}

	if !offs64 {
		return binary.Write(w, binary.BigEndian, int32(v.begin))
	}

	return binary.Write(w, binary.BigEndian, &v.begin)
}

// writeHeader encodes the CDF header to the io.Writer at the current position.
// If an error occurs that prevents further writing, the writer is left at the
// erroring position and err is set to the error from the underlying call to binary.Write. 
func (h *Header) WriteHeader(w io.Writer) error {

	if err := binary.Write(w, binary.BigEndian, [4]byte{'C', 'D', 'F', byte(h.version)}); err != nil {
		return err
	}

	var numrecs int32 = _STREAMING // ignored on reading
	if err := binary.Write(w, binary.BigEndian, numrecs); err != nil {
		return err
	}

	if len(h.dim) == 0 {
		if err := binary.Write(w, binary.BigEndian, [2]int32{0, 0}); err != nil {
			return err
		}
	} else {
		if err := binary.Write(w, binary.BigEndian, [2]int32{0xA, int32(len(h.dim))}); err != nil {
			return err
		}
		for i := range h.dim {
			if err := h.dim[i].writeTo(w); err != nil {
				return err
			}
		}
	}

	if len(h.att) == 0 {
		if err := binary.Write(w, binary.BigEndian, [2]int32{0, 0}); err != nil {
			return err
		}
	} else {
		if err := binary.Write(w, binary.BigEndian, [2]int32{0xC, int32(len(h.att))}); err != nil {
			return err
		}
		for i := range h.att {
			if err := h.att[i].writeTo(w); err != nil {
				return err
			}
		}
	}

	if len(h.vars) == 0 {
		if err := binary.Write(w, binary.BigEndian, [2]int32{0, 0}); err != nil {
			return err
		}
	} else {
		if err := binary.Write(w, binary.BigEndian, [2]int32{0xB, int32(len(h.vars))}); err != nil {
			return err
		}
		for i := range h.vars {
			if err := h.vars[i].writeTo(w, h.version == _V2); err != nil {
				return err
			}
		}
	}

	return nil
}

// a nullWriter discards all data but keeps track of the number of bytes written.
type nullWriter int64

func (w *nullWriter) Write(p []byte) (int, error) {
	*(*int64)(w) += int64(len(p))
	return len(p), nil
}

// return the size in bytes of the serialized header
func (h *Header) size() int64 {
	var nw nullWriter
	h.WriteHeader(&nw)
	return int64(nw)
}
