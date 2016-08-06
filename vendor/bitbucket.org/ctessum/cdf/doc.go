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

// Package CDF provides facilities to read and write files in NetCDF 'classic' (V1 or V2) format.
// The HDF based NetCDF-4 format is not supported.
//
// The data model and the classic file format are documented at
//	http://www.unidata.ucar.edu/software/netcdf/docs/tutorial.html
// 	http://www.unidata.ucar.edu/software/netcdf/docs/classic_format_spec.html
//
// A NetCDF file contains an immutable header (this library does not support modifying it)
// that defines the layout of the data section and contains metadata.  The data can be read,
// written and, if there exists a record dimension, appended to.
//
// To create a new file, first create a header, e.g.:
//
//      h := cdf.NewHeader([]string{"time", "x", "y", "z"}, []int{0, 10, 10, 10})
//      h.AddVariable("psi", []string{"time", "x"}, float32(0))
//      h.AddAttribute("", "comment", "This is a test file")
//      h.AddAttribute("psi", "description", "The value of psi as a function of time and x")
//      h.AddAttribute("psi", "interesting_value", float32(42))
//      h.Define()
//      ff, _ := os.Create("/path/to/file")
//      f, _ := cdf.Create(ff, h)   // writes the header to ff
//
// To use an existing file:
//
//	ff, _ := os.Open("/path/to/file")
//	f, _ := cdf.Open(ff)
//
//
// The Header field of f is now usable for inspection of dimensions, variables and attributes, but
// should not be modified (obvious ways of doing this will cause panics).
//
// To read data from the file, use 
//	r := f.Reader("psi", nil, nil)
//	buf := r.Zero(100)      // a []T of the right T for the variable.
//	n, err := r.Read(buf)   // similar to io.Read, but reads T's instead of bytes.
//
// And similar for writing.
//
package cdf
