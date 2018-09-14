/*
Copyright © 2017 the InMAP authors.
This file is part of InMAP.

InMAP is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

InMAP is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with InMAP.  If not, see <http://www.gnu.org/licenses/>.*/

// Package eieio implements an Extended InMAP Economic Input-Output (EIEIO)
// life cycle assessment
// model based on the US Bureau of Economic Analysis (BEA)
// Annual Input-Output Accounts Data from
// https://www.gov/industry/io_annual.htm.
package eieio

// Install the code generation dependencies.
// go get -u github.com/golang/protobuf/protoc-gen-go
// go get -u github.com/johanbrandhorst/protobuf/protoc-gen-gopherjs

// Generate the gRPC client/server code. (Information at https://grpc.io/docs/quickstart/go.html)
//go:generate protoc eieio.proto --go_out=plugins=grpc:eieiorpc --gopherjs_out=plugins=grpc:eieiorpc/eieiorpcjs

// Build the client javascript code.
//go:generate gopherjs build -m ./gui
