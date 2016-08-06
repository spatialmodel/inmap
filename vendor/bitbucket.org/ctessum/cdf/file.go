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

// This file contains the File type.

package cdf

import (
	"bytes"
	"encoding/binary"
	"io"
)

// A ReaderWriterAt is the underlying storage for a NetCDF file,
// providing {Read,Write}At([]byte, int64) methods.
// Since {Read,Write}At are required to not modify the underlying
// file pointer, one instance may be shared by multiple Files, although
// the documentation of io.WriterAt specifies that it only has to 
// guarantee non-concurrent calls succeed.
type ReaderWriterAt interface {
	io.ReaderAt
	io.WriterAt
}

type File struct {
	rw     ReaderWriterAt
	Header *Header
}

// Open reads the header from an existing storage rw and returns a File
// usable for reading or writing (if the underlying rw permits).
func Open(rw ReaderWriterAt) (*File, error) {
	h, err := ReadHeader(io.NewSectionReader(rw, 0, 1<<31))
	if err != nil {
		return nil, err
	}
	return &File{rw: rw, Header: h}, nil
}

// Create writes the header to a storage rw and returns a File
// usable for reading and writing.
//
// The header should not be mutable, and may be shared by multiple
// Files.  Note that at every Create the headers numrec
// field will be reset to -1 (STREAMING).
func Create(rw ReaderWriterAt, h *Header) (*File, error) {
	if h.isMutable() {
		panic("Create must be called with a fully defined header")
	}
	var buf bytes.Buffer
	err := h.WriteHeader(&buf)
	if err != nil {
		return nil, err
	}
	if _, err := rw.WriteAt(buf.Bytes(), 0); err != nil {
		return nil, err
	}
	return &File{rw: rw, Header: h}, nil
}

func fill(w io.WriterAt, begin, end int64, val interface{}, dtype datatype) error {
	var buf bytes.Buffer
	binary.Write(&buf, binary.BigEndian, val)
	if buf.Len() != dtype.storageSize() {
		panic("invalid fill value")
	}
	d := int64(buf.Len())
	for ; begin < end; begin += d {
		if _, err := w.WriteAt(buf.Bytes(), begin); err != nil {
			return err
		}
	}
	return nil
}

// Fill overwrites the data for non-record variable named v with its fill value.
// Fill panics if v does not name a non-record variable.
// If the variable has a scalar attribute '_FillValue' of the same data type as the variable,
// it will be used, otherwise the type's default fill value will be used.
func (f *File) Fill(v string) error {
	vv := f.Header.varByName(v)
	if vv == nil || vv.isRecordVariable() {
		panic("Fill for non-record variable")
	}
	return fill(f.rw, vv.begin, vv.begin+pad4(vv.vSize()), vv.fillValue(), vv.dtype)
}

// FillRecord overwrites the data for all record variables in the r'th slab with their fill values.
func (f *File) FillRecord(r int) error {
	_, slabsize := f.Header.slabs()
	for i := range f.Header.vars {
		vv := &f.Header.vars[i]
		if !vv.isRecordVariable() {
			continue
		}
		begin := vv.begin + int64(r)*slabsize
		end := begin + pad4(vv.vSize())
		if err := fill(f.rw, begin, end, vv.fillValue(), vv.dtype); err != nil {
			return err
		}
	}
	return nil
}
