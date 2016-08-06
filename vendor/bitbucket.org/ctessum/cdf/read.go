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

// This file contains the code to deserialize cdf headers.

package cdf

import (
	"encoding/binary"
	"errors"
	"io"
	"log"
)

// The errors returned by readHeader that prevent reading the rest of the header.
var (
	badMagic         = errors.New("Invalid CDF Magic")
	badVersion       = errors.New("Unsupported CDF Version")
	badTag           = errors.New("Invalid tag")
	badLength        = errors.New("Invalid data length")
	badAttributeType = errors.New("Invalid attribute storage type")
)

// read an (int32, []byte) encoded string
func readString(r io.Reader) (s string, err error) {
	var nelems int32
	if err = binary.Read(r, binary.BigEndian, &nelems); err != nil {
		return "", err
	}
	if nelems < 0 {
		return "", badLength
	}
	buf := make([]byte, int(nelems+3)&^3) // pad to multiple of 4
	n, err := r.Read(buf)
	if n < len(buf) && err == nil {
		err = io.EOF
	}
	if err != nil {
		return "", err
	}

	return string(buf[:nelems]), nil
}

// used by readHeader
func (d *dimension) readFrom(r io.Reader) (err error) {
	if d.name, err = readString(r); err != nil {
		return err
	}
	return binary.Read(r, binary.BigEndian, &d.length)
}

// used by readHeader
func (a *attribute) readFrom(r io.Reader) (err error) {
	if a.name, err = readString(r); err != nil {
		return err
	}
	if err = binary.Read(r, binary.BigEndian, &a.dtype); err != nil {
		return err
	}
	if !a.dtype.valid() {
		return badAttributeType
	}

	if a.dtype == _CHAR {
		if a.values, err = readString(r); err != nil {
			return err
		}
		return nil
	}

	var nelems int32
	if err = binary.Read(r, binary.BigEndian, &nelems); err != nil {
		return err
	}
	if nelems < 0 {
		return badLength
	}

	switch a.dtype {
	case _BYTE:
		a.values = make([]uint8, int(nelems+3)&^3) // pad to multiple of 4 * 1
	case _SHORT:
		a.values = make([]int16, int(nelems+1)&^1) // pad to multiple of 2 * 2
	case _INT:
		a.values = make([]int32, nelems)
	case _FLOAT:
		a.values = make([]float32, nelems)
	case _DOUBLE:
		a.values = make([]float64, nelems)
	}

	if err = binary.Read(r, binary.BigEndian, a.values); err != nil {
		return err
	}

	switch a.dtype {
	case _BYTE:
		a.values = a.values.([]uint8)[:nelems]
	case _SHORT:
		a.values = a.values.([]int16)[:nelems]
	}

	return nil
}

// used by readHeader
func (v *variable) readFrom(r io.Reader, offs64 bool) (err error) {
	var tag, nelems int32

	if v.name, err = readString(r); err != nil {
		return err
	}

	if err = binary.Read(r, binary.BigEndian, &nelems); err != nil {
		return err
	}
	if nelems < 0 {
		return badLength
	}
	v.dim = make([]int32, nelems)
	if err = binary.Read(r, binary.BigEndian, v.dim); err != nil {
		return err
	}
	if err = binary.Read(r, binary.BigEndian, &tag); err != nil {
		return err
	}
	if err = binary.Read(r, binary.BigEndian, &nelems); err != nil {
		return err
	}

	switch tag {
	case 0:
		if nelems != 0 {
			return badLength
		}
	case 0xC:
		v.att = make([]attribute, nelems)
		for i := range v.att {
			if err = v.att[i].readFrom(r); err != nil {
				return err
			}
		}
	default:
		return badTag
	}

	if err = binary.Read(r, binary.BigEndian, &v.dtype); err != nil {
		return err
	}

	if err = binary.Read(r, binary.BigEndian, &v.vsize); err != nil {
		return err
	}

	if !offs64 {
		var b32 int32
		if err = binary.Read(r, binary.BigEndian, &b32); err != nil {
			return err
		}
		v.begin = int64(b32)
		return nil
	}

	return binary.Read(r, binary.BigEndian, &v.begin)
}

// readHeader decodes the CDF header from the io.Reader at the current position.
// On success readHeader returns a header struct and a nil error.
// If an error occurs that prevents further reading, the reader is left at the
// error position and err is set to badMagic, badVersion, badTag, badLenght or badAttributeType,
// or the error from the underlying call to binary.Read.
// The returned header is immutable, meaning it may not be modified with AddVariable or AddAttribute.
func ReadHeader(r io.Reader) (*Header, error) {
	var (
		magic       [3]byte
		version     version
		tag, nelems int32
	)

	if err := binary.Read(r, binary.BigEndian, &magic); err != nil {
		return nil, err
	}

	if magic != [3]byte{'C', 'D', 'F'} {
		return nil, badMagic
	}

	if err := binary.Read(r, binary.BigEndian, &version); err != nil {
		return nil, err
	}

	if version != _V1 && version != _V2 {
		return nil, badVersion
	}

	h := &Header{version: version}

	var numrecs int32 // ignored
	if err := binary.Read(r, binary.BigEndian, &numrecs); err != nil {
		return nil, err
	}

	for ii := 0; ii < 3; ii++ {

		if err := binary.Read(r, binary.BigEndian, &tag); err != nil {
			return nil, err
		}
		if err := binary.Read(r, binary.BigEndian, &nelems); err != nil {
			return nil, err
		}
		if nelems < 0 {
			return nil, badLength
		}

		switch tag {
		case 0:
			if nelems != 0 {
				return nil, badLength
			}

		case 0xA: // list of dimensions
			if ii != 0 {
				log.Printf("Dimension section out of order: %d", ii)
			}

			h.dim = make([]dimension, nelems)
			for i := range h.dim {
				if err := h.dim[i].readFrom(r); err != nil {
					return nil, err
				}
			}

		case 0xB: // list of variables
			if ii != 2 {
				log.Printf("Variable section out of order: %d", ii)
			}

			h.vars = make([]variable, nelems)
			for i := range h.vars {
				if err := h.vars[i].readFrom(r, h.version == _V2); err != nil {
					return nil, err
				}
				h.vars[i].setComputed(h.dim)
			}

		case 0xC: // list of attributes
			if ii != 1 {
				log.Printf("Global attribute section out of order: %d", ii)
			}

			h.att = make([]attribute, nelems)
			for i := range h.att {
				if err := h.att[i].readFrom(r); err != nil {
					return nil, err
				}
			}
		default:
			return nil, badTag
		}
	}

	h.fixRecordStrides()

	return h, nil
}
