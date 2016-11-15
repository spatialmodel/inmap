/*
Copyright (C) 2012-2014 Regents of the University of Minnesota.
This file is part of AEP.

AEP is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

AEP is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with AEP.  If not, see <http://www.gnu.org/licenses/>.
*/

package aep

import (
	"net"
	"net/http"
	"net/rpc"
)

var RPCport = "6061" // Port for RPC communications for distributed computing

// Set up a server to remotely calculate gridding surrogates.
func DistributedServer(c *Context) {
	srgGenWorker := new(srgGenWorker)
	rpc.Register(srgGenWorker)
	rpc.HandleHTTP()
	l, err := net.Listen("tcp", ":"+RPCport)
	if err != nil {
		panic(err)
	}
	http.Serve(l, nil)
}
