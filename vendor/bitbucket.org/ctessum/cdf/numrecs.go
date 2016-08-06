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

// this file contains the code to deal with the numrecs field in cdf headers.

package cdf

import (
	"io"
	"os"
)

const _STREAMING = int32(-1) // value of numrecs meaning 'indeterminate'

const _NumRecsOffset = 4 // position of the bigendian int32 in the header

func readNumRecs(r io.ReaderAt) (int64, error) {
	var buf [4]byte
	_, err := r.ReadAt(buf[:], _NumRecsOffset)
	if err != nil {
		return 0, err
	}
	return int64(buf[0])<<24 + int64(buf[1])<<16 + int64(buf[2])<<8 + int64(buf[3]), nil
}

func writeNumRecs(w io.WriterAt, numrecs int64) error {
	if numrecs >= (1<<31) || numrecs < 0 {
		numrecs = -1
	}
	buf := [4]byte{byte(numrecs >> 24), byte(numrecs >> 16), byte(numrecs >> 8), byte(numrecs)}
	_, err := w.WriteAt(buf[:], _NumRecsOffset)
	return err
}

// UpdateNumRecs determines the number of record from the file size and
// writes it into the file's header as the 'numrecs' field.
//
// Any incomplete trailing record will not be included in the count.
//
// Only valid headers will be updated.
// After a succesful call f's filepointer will be left at the end of the file.
//
// This library does not use the numrecs header field but updating it
// enables full bit for bit compatibility with other libraries.  There
// is no need to call this function until after all updates by the program,
// and it is rather costly because it reads, parses and checks the entire header.
func UpdateNumRecs(f *os.File) error {
	fi, err := f.Stat()
	if err != nil {
		return err
	}

	if _, err = f.Seek(0, 0); err != nil {
		return err
	}

	h, err := ReadHeader(f)
	if err != nil {
		return err
	}

	// seek to EOF to avoid hard to find clobbering errors
	if _, err = f.Seek(0, 2); err != nil {
		return err
	}

	if errs := h.Check(); errs != nil {
		return errs[0] // only room for the first
	}

	if err = writeNumRecs(f, h.NumRecs(fi.Size())); err != nil {
		return err
	}

	return nil
}
