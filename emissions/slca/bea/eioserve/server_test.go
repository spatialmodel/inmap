/*
Copyright © 2018 the InMAP authors.
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

package eioserve

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	eioservepb "github.com/spatialmodel/inmap/emissions/slca/bea/eioserve/proto/eioservepb"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/testdata"
)

func TestServer_grpc(t *testing.T) {
	s, err := NewServer()
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	go func() {
		http.ListenAndServeTLS(eioservepb.Address, testdata.Path("server1.pem"), testdata.Path("server1.key"), s)
	}()

	t.Run("index", func(t *testing.T) {
		r, err := http.Get("https://" + eioservepb.Address)
		if err != nil {
			t.Error(err)
		}
		fmt.Println(r)
	})

	c, err := NewClient()
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	ctx := context.Background()

	t.Run("DemandGroups", func(t *testing.T) {
		r, err := c.DemandGroups(ctx, &eioservepb.Selection{
			DemandGroup:      eioservepb.All,
			DemandSector:     eioservepb.All,
			ProductionGroup:  eioservepb.All,
			ProductionSector: eioservepb.All,
			ImpactType:       "health_total",
			DemandType:       "All",
		})
		if err != nil {
			t.Error(err)
		}
		fmt.Println(r)
	})

}

func NewClient() (eioservepb.EIOServeClient, error) {
	creds, err := credentials.NewClientTLSFromFile(testdata.Path("ca.pem"), "x.test.youtube.com")
	if err != nil {
		return nil, err
	}
	opt := grpc.WithTransportCredentials(creds)
	conn, err := grpc.Dial(eioservepb.Address, opt)
	if err != nil {
		return nil, err
	}
	return eioservepb.NewEIOServeClient(conn), nil
}
