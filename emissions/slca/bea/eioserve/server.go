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

package eioserve

// Install the code generation dependencies.
// go get github.com/golang/protobuf/protoc-gen-go
// go get github.com/johanbrandhorst/protobuf/protoc-gen-gopherjs

// Generate the gRPC client/server code. (Information at https://grpc.io/docs/quickstart/go.html)
//go:generate protoc -I proto/ eioserve.proto --go_out=plugins=grpc:proto/eioservepb --gopherjs_out=plugins=grpc:proto/eioclientpb

// Build the client javascript code.
//go:generate gopherjs build -m ./gui

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/gob"
	"fmt"
	"image/color"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"

	"github.com/ctessum/geom"
	"github.com/ctessum/geom/proj"
	"github.com/ctessum/requestcache"
	"github.com/gorilla/websocket"
	"github.com/improbable-eng/grpc-web/go/grpcweb"
	"github.com/sirupsen/logrus"
	"github.com/spatialmodel/epi"
	"github.com/spatialmodel/inmap/emissions/aep"
	"gonum.org/v1/gonum/mat"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/palette"
	"gonum.org/v1/plot/palette/moreland"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
	"gonum.org/v1/plot/vg/draw"
	"gonum.org/v1/plot/vg/vgimg"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/testdata"

	"github.com/spatialmodel/inmap/emissions/slca"
	"github.com/spatialmodel/inmap/emissions/slca/bea"
	"github.com/spatialmodel/inmap/emissions/slca/bea/eioserve/proto/eioservepb"
)

type config struct {
	EIO bea.Config
	slca.CSTConfig
	AggregatorFile string
	SpatialRefFile string
}

const year = 2014

// Server is a server for EIO LCA model simulation data.
type Server struct {
	spatial *bea.SpatialEIO
	agg     *bea.Aggregator

	sccDescriptions map[string]string

	geomCache, areaCache         *requestcache.Cache
	geomCacheOnce, areaCacheOnce sync.Once

	grpcServer   *grpcweb.WrappedGrpcServer
	staticServer http.Handler

	Log logrus.FieldLogger
}

// NewServer creates a new EIO-LCA server.
func NewServer() (*Server, error) {
	f, err := os.Open(os.ExpandEnv("${GOPATH}/src/github.com/spatialmodel/inmap/emissions/slca/bea/eioserve/config.toml"))
	if err != nil {
		return nil, err
	}

	s, err := bea.NewSpatial(f)
	if err != nil {
		return nil, err
	}
	f.Close()

	a, err := s.EIO.NewAggregator(os.ExpandEnv("${GOPATH}/src/github.com/spatialmodel/inmap/emissions/slca/bea/data/aggregates.xlsx"))
	if err != nil {
		return nil, err
	}

	r, err := os.Open(os.ExpandEnv("${GOPATH}/src/github.com/spatialmodel/inmap/emissions/aep/data/nei2014/sccdesc_2014platform_09sep2016_v0.txt"))
	if err != nil {
		return nil, err
	}
	sccDescriptions, err := aep.SCCDescription(r)
	if err != nil {
		return nil, err
	}

	model := &Server{
		agg:             a,
		spatial:         s,
		Log:             logrus.StandardLogger(),
		sccDescriptions: sccDescriptions,
	}

	creds, err := credentials.NewServerTLSFromFile(testdata.Path("server1.pem"), testdata.Path("server1.key"))
	if err != nil {
		return nil, err
	}
	opt := grpc.Creds(creds)
	grpcServer := grpc.NewServer(opt)
	eiopb.RegisterEIOServeServer(grpcServer, model)

	model.grpcServer = grpcweb.WrapServer(grpcServer, grpcweb.WithWebsockets(true))

	model.staticServer = http.FileServer(http.Dir(os.ExpandEnv("${GOPATH}/src/github.com/spatialmodel/inmap/emissions/slca/bea/eioserve")))

	return model, nil
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.ProtoMajor == 2 && strings.Contains(r.Header.Get("Content-Type"), "application/grpc") || websocket.IsWebSocketUpgrade(r) {
		s.Log.WithFields(logrus.Fields{
			"url":  r.URL.String(),
			"addr": r.RemoteAddr,
		}).Info("eioserve grpc request")
		s.grpcServer.ServeHTTP(w, r)
	} else {
		s.Log.WithFields(logrus.Fields{
			"url":  r.URL.String(),
			"addr": r.RemoteAddr,
		}).Info("eioserve static request")
		s.staticServer.ServeHTTP(w, r)
	}
}

func (s *Server) impactsMenu(ctx context.Context, selection *eiopb.Selection, commodityMask, industryMask *bea.Mask) (*mat.VecDense, error) {
	demand, err := s.spatial.EIO.FinalDemand(bea.FinalDemand(selection.DemandType), commodityMask, year, bea.Domestic)
	if err != nil {
		return nil, err
	}
	switch selection.ImpactType {
	case "health_total", "conc_totalPM25":
		return s.spatial.Health(ctx, demand, industryMask, bea.TotalPM25, "all", year, bea.Domestic, epi.NasariACS)
	case "health_white":
		return s.spatial.Health(ctx, demand, industryMask, bea.TotalPM25, "white", year, bea.Domestic, epi.NasariACS)
	case "health_black":
		return s.spatial.Health(ctx, demand, industryMask, bea.TotalPM25, "black", year, bea.Domestic, epi.NasariACS)
	case "health_native":
		return s.spatial.Health(ctx, demand, industryMask, bea.TotalPM25, "native", year, bea.Domestic, epi.NasariACS)
	case "health_asian":
		return s.spatial.Health(ctx, demand, industryMask, bea.TotalPM25, "asian", year, bea.Domestic, epi.NasariACS)
	case "health_latino":
		return s.spatial.Health(ctx, demand, industryMask, bea.TotalPM25, "latino", year, bea.Domestic, epi.NasariACS)
	case "conc_PNH4":
		return s.spatial.Health(ctx, demand, industryMask, bea.PNH4, "all", year, bea.Domestic, epi.NasariACS)
	case "conc_PNO3":
		return s.spatial.Health(ctx, demand, industryMask, bea.PNO3, "all", year, bea.Domestic, epi.NasariACS)
	case "conc_PSO4":
		return s.spatial.Health(ctx, demand, industryMask, bea.PSO4, "all", year, bea.Domestic, epi.NasariACS)
	case "conc_SOA":
		return s.spatial.Health(ctx, demand, industryMask, bea.SOA, "all", year, bea.Domestic, epi.NasariACS)
	case "conc_PrimaryPM25":
		return s.spatial.Health(ctx, demand, industryMask, bea.PrimaryPM25, "all", year, bea.Domestic, epi.NasariACS)
	case "emis_PM25":
		return s.spatial.Emissions(ctx, demand, industryMask, slca.PM25, year, bea.Domestic)
	case "emis_NH3":
		return s.spatial.Emissions(ctx, demand, industryMask, slca.NH3, year, bea.Domestic)
	case "emis_NOx":
		return s.spatial.Emissions(ctx, demand, industryMask, slca.NOx, year, bea.Domestic)
	case "emis_SOx":
		return s.spatial.Emissions(ctx, demand, industryMask, slca.SOx, year, bea.Domestic)
	case "emis_VOC":
		return s.spatial.Emissions(ctx, demand, industryMask, slca.VOC, year, bea.Domestic)
	default:
		return nil, fmt.Errorf("invalid impact type request: %s", selection.ImpactType)
	}
}

func (s *Server) impactsMap(ctx context.Context, selection *eiopb.Selection, commodityMask, industryMask *bea.Mask) (*mat.VecDense, error) {
	demand, err := s.spatial.EIO.FinalDemand(bea.FinalDemand(selection.DemandType), commodityMask, year, bea.Domestic)
	if err != nil {
		return nil, err
	}
	switch selection.ImpactType {
	case "health_total":
		return s.perArea(s.spatial.Health(ctx, demand, industryMask, bea.TotalPM25, "all", year, bea.Domestic, epi.NasariACS))
	case "health_white":
		return s.perArea(s.spatial.Health(ctx, demand, industryMask, bea.TotalPM25, "white", year, bea.Domestic, epi.NasariACS))
	case "health_black":
		return s.perArea(s.spatial.Health(ctx, demand, industryMask, bea.TotalPM25, "black", year, bea.Domestic, epi.NasariACS))
	case "health_native":
		return s.perArea(s.spatial.Health(ctx, demand, industryMask, bea.TotalPM25, "native", year, bea.Domestic, epi.NasariACS))
	case "health_asian":
		return s.perArea(s.spatial.Health(ctx, demand, industryMask, bea.TotalPM25, "asian", year, bea.Domestic, epi.NasariACS))
	case "health_latino":
		return s.perArea(s.spatial.Health(ctx, demand, industryMask, bea.TotalPM25, "latino", year, bea.Domestic, epi.NasariACS))
	case "conc_totalPM25":
		return s.spatial.Concentrations(ctx, demand, industryMask, bea.TotalPM25, year, bea.Domestic)
	case "conc_PNH4":
		return s.spatial.Concentrations(ctx, demand, industryMask, bea.PNH4, year, bea.Domestic)
	case "conc_PNO3":
		return s.spatial.Concentrations(ctx, demand, industryMask, bea.PNO3, year, bea.Domestic)
	case "conc_PSO4":
		return s.spatial.Concentrations(ctx, demand, industryMask, bea.PSO4, year, bea.Domestic)
	case "conc_SOA":
		return s.spatial.Concentrations(ctx, demand, industryMask, bea.SOA, year, bea.Domestic)
	case "conc_PrimaryPM25":
		return s.spatial.Concentrations(ctx, demand, industryMask, bea.PrimaryPM25, year, bea.Domestic)
	case "emis_PM25":
		return s.perArea(s.spatial.Emissions(ctx, demand, industryMask, slca.PM25, year, bea.Domestic))
	case "emis_NH3":
		return s.perArea(s.spatial.Emissions(ctx, demand, industryMask, slca.NH3, year, bea.Domestic))
	case "emis_NOx":
		return s.perArea(s.spatial.Emissions(ctx, demand, industryMask, slca.NOx, year, bea.Domestic))
	case "emis_SOx":
		return s.perArea(s.spatial.Emissions(ctx, demand, industryMask, slca.SOx, year, bea.Domestic))
	case "emis_VOC":
		return s.perArea(s.spatial.Emissions(ctx, demand, industryMask, slca.VOC, year, bea.Domestic))
	default:
		return nil, fmt.Errorf("invalid impact type request: %s", selection.ImpactType)
	}
}

func (s *Server) perArea(v *mat.VecDense, err error) (*mat.VecDense, error) {
	if err != nil {
		return nil, err
	}
	area, err := s.inverseArea()
	if err != nil {
		return nil, err
	}
	v.MulElemVec(v, area)
	return v, nil
}

// DemandGroups returns the available demand groups.
func (s *Server) DemandGroups(ctx context.Context, in *eiopb.Selection) (*eiopb.Selectors, error) {
	s.Log.WithFields(logrus.Fields{
		"DemandGroup":      in.DemandGroup,
		"DemandSector":     in.DemandSector,
		"ProductionGroup":  in.ProductionGroup,
		"ProductionSector": in.ProductionSector,
		"ImpactType":       in.ImpactType,
		"DemandType":       in.DemandType,
	}).Info("eioserve generating DemandGroups")

	out := &eiopb.Selectors{
		Names:  make([]string, len(s.agg.Names())+1),
		Values: make([]float32, len(s.agg.Names())+1),
	}
	out.Names[0] = eiopb.All
	// impacts produced by all sectors owing to all sectors.
	impacts, err := s.impactsMenu(ctx, in, nil, nil)
	if err != nil {
		return nil, err
	}
	out.Values[0] = float32(mat.Sum(impacts))
	i := 1
	for _, g := range s.agg.Names() {
		out.Names[i] = g

		// impacts produced by all sectors owing to consumption in this group
		// of sectors.
		mask, err := s.demandMask(g, eiopb.All)
		if err != nil {
			return nil, err
		}
		impacts, err := s.impactsMenu(ctx, in, mask, nil)
		if err != nil {
			return nil, err
		}
		out.Values[i] = float32(mat.Sum(impacts))
		i++
	}
	sort.Sort(out)
	s.Log.WithFields(logrus.Fields{
		"DemandGroup":      in.DemandGroup,
		"DemandSector":     in.DemandSector,
		"ProductionGroup":  in.ProductionGroup,
		"ProductionSector": in.ProductionSector,
		"ImpactType":       in.ImpactType,
		"DemandType":       in.DemandType,
	}).Info("eioserve finished generating DemandGroups")
	return out, nil
}

func commodityGroup(e *bea.EIO, m *bea.Mask) []string {
	var o []string
	v := (mat.VecDense)(*m)
	for i, c := range e.Commodities {
		if v.At(i, 0) != 0 {
			o = append(o, c)
		}
	}
	return o
}

func industryGroup(e *bea.EIO, m *bea.Mask) []string {
	var o []string
	v := (mat.VecDense)(*m)
	for i, c := range e.Industries {
		if v.At(i, 0) != 0 {
			o = append(o, c)
		}
	}
	return o
}

// DemandSectors returns the available demand sectors.
func (s *Server) DemandSectors(ctx context.Context, in *eiopb.Selection) (*eiopb.Selectors, error) {
	s.Log.WithFields(logrus.Fields{
		"DemandGroup":      in.DemandGroup,
		"DemandSector":     in.DemandSector,
		"ProductionGroup":  in.ProductionGroup,
		"ProductionSector": in.ProductionSector,
		"ImpactType":       in.ImpactType,
		"DemandType":       in.DemandType,
	}).Info("eioserve generating DemandSectors")
	out := &eiopb.Selectors{Names: []string{eiopb.All}}
	if in.DemandGroup == eiopb.All {
		// impacts produced by all sectors owing to all sectors.
		impacts, err := s.impactsMenu(ctx, in, nil, nil)
		if err != nil {
			return nil, err
		}
		out.Values = []float32{float32(mat.Sum(impacts))}
		return out, nil
	}
	mask, err := s.demandMask(in.DemandGroup, eiopb.All)
	if err != nil {
		return nil, err
	}
	impacts, err := s.impactsMenu(ctx, in, mask, nil)
	if err != nil {
		return nil, err
	}
	out.Values = []float32{float32(mat.Sum(impacts))}

	sectors := commodityGroup(&s.spatial.EIO, mask)
	out.Names = append(out.Names, sectors...)
	temp := make([]float32, len(sectors))
	out.Values = append(out.Values, temp...)
	for i, sector := range sectors {
		// impacts produced by all sectors owing to consumption in this sector.
		mask, err := s.demandMask(in.DemandGroup, sector)
		if err != nil {
			return nil, err
		}
		impacts, err := s.impactsMenu(ctx, in, mask, nil)
		if err != nil {
			return nil, err
		}
		out.Values[i+1] = float32(mat.Sum(impacts))
	}
	sort.Sort(out)
	s.Log.WithFields(logrus.Fields{
		"DemandGroup":      in.DemandGroup,
		"DemandSector":     in.DemandSector,
		"ProductionGroup":  in.ProductionGroup,
		"ProductionSector": in.ProductionSector,
		"ImpactType":       in.ImpactType,
		"DemandType":       in.DemandType,
	}).Info("eioserve finished generating DemandSectors")
	return out, nil
}

// demandMask returns a commodity mask corresponding to the
// given selection.
func (s *Server) demandMask(demandGroup, demandSector string) (*bea.Mask, error) {
	if demandGroup == eiopb.All {
		return nil, nil
	} else if demandSector == eiopb.All {
		// demand from a group of sectors.
		abbrev, err := s.agg.Abbreviation(demandGroup)
		if err != nil {
			return nil, err
		}
		return s.agg.CommodityMask(abbrev), nil
	}
	// demand from a single sector.
	return s.spatial.EIO.CommodityMask(demandSector)
}

// productionMask returns a commodity mask corresponding to the
// given selection.
func (s *Server) productionMask(productionGroup, productionSector string) (*bea.Mask, error) {
	if productionGroup == eiopb.All {
		return nil, nil
	} else if productionSector == eiopb.All {
		// demand from a group of sectors.
		abbrev, err := s.agg.Abbreviation(productionGroup)
		if err != nil {
			return nil, err
		}
		return s.agg.IndustryMask(abbrev), nil
	}
	// demand from a single sector.
	return s.spatial.EIO.IndustryMask(productionSector)
}

// ProdGroups returns the available production groups.
func (s *Server) ProdGroups(ctx context.Context, in *eiopb.Selection) (*eiopb.Selectors, error) {
	s.Log.WithFields(logrus.Fields{
		"DemandGroup":      in.DemandGroup,
		"DemandSector":     in.DemandSector,
		"ProductionGroup":  in.ProductionGroup,
		"ProductionSector": in.ProductionSector,
		"ImpactType":       in.ImpactType,
		"DemandType":       in.DemandType,
	}).Info("eioserve generating ProdGroups")
	demandMask, err := s.demandMask(in.DemandGroup, in.DemandSector)
	if err != nil {
		return nil, err
	}
	out := &eiopb.Selectors{
		Names:  make([]string, len(s.agg.Names())+1),
		Values: make([]float32, len(s.agg.Names())+1),
	}
	out.Names[0] = eiopb.All
	v, err := s.impactsMenu(ctx, in, demandMask, nil)
	if err != nil {
		return nil, err
	}
	out.Values[0] = float32(mat.Sum(v))
	i := 1
	for _, g := range s.agg.Names() {
		out.Names[i] = g
		mask, err := s.productionMask(g, eiopb.All)
		if err != nil {
			return nil, err
		}
		v, err := s.impactsMenu(ctx, in, demandMask, mask)
		if err != nil {
			return nil, err
		}
		out.Values[i] = float32(mat.Sum(v))
		i++
	}
	sort.Sort(out)
	s.Log.WithFields(logrus.Fields{
		"DemandGroup":      in.DemandGroup,
		"DemandSector":     in.DemandSector,
		"ProductionGroup":  in.ProductionGroup,
		"ProductionSector": in.ProductionSector,
		"ImpactType":       in.ImpactType,
		"DemandType":       in.DemandType,
	}).Info("eioserve finished generating ProdGroups")
	return out, nil
}

// ProdSectors returns the available production sectors.
func (s *Server) ProdSectors(ctx context.Context, in *eiopb.Selection) (*eiopb.Selectors, error) {
	s.Log.WithFields(logrus.Fields{
		"DemandGroup":      in.DemandGroup,
		"DemandSector":     in.DemandSector,
		"ProductionGroup":  in.ProductionGroup,
		"ProductionSector": in.ProductionSector,
		"ImpactType":       in.ImpactType,
		"DemandType":       in.DemandType,
	}).Info("eioserve generating ProdSectors")
	demandMask, err := s.demandMask(in.DemandGroup, in.DemandSector)
	if err != nil {
		return nil, err
	}

	out := &eiopb.Selectors{Names: []string{eiopb.All}}
	if in.ProductionGroup == eiopb.All {
		v, err2 := s.impactsMenu(ctx, in, demandMask, nil)
		if err2 != nil {
			return nil, err2
		}
		out.Values = []float32{float32(mat.Sum(v))}
		return out, nil
	}
	mask, err := s.productionMask(in.ProductionGroup, eiopb.All)
	if err != nil {
		return nil, err
	}
	v, err := s.impactsMenu(ctx, in, demandMask, mask)
	if err != nil {
		return nil, err
	}
	out.Values = []float32{float32(mat.Sum(v))}
	sectors := industryGroup(&s.spatial.EIO, mask)
	out.Names = append(out.Names, sectors...)
	temp := make([]float32, len(sectors))
	out.Values = append(out.Values, temp...)
	for i, sector := range sectors {
		mask, err := s.productionMask(in.ProductionGroup, sector)
		if err != nil {
			return nil, err
		}
		v, err := s.impactsMenu(ctx, in, demandMask, mask)
		if err != nil {
			return nil, err
		}
		out.Values[i+1] = float32(mat.Sum(v))
	}
	sort.Sort(out)
	s.Log.WithFields(logrus.Fields{
		"DemandGroup":      in.DemandGroup,
		"DemandSector":     in.DemandSector,
		"ProductionGroup":  in.ProductionGroup,
		"ProductionSector": in.ProductionSector,
		"ImpactType":       in.ImpactType,
		"DemandType":       in.DemandType,
	}).Info("eioserve finished generating ProdSectors")
	return out, nil
}

func (s *Server) SCCs(in *eiopb.Selection, stream eiopb.EIOServe_SCCsServer) error {
	s.Log.WithFields(logrus.Fields{
		"ProductionSector": in.ProductionSector,
	}).Info("eioserve generating SCCs")
	if in.ProductionSector == eiopb.All {
		return nil
	}
	spatialRefs, ok := s.spatial.SpatialRefs[year]
	if !ok {
		return fmt.Errorf("SCCs: mission SpatialRefs for year %d", year)
	}
	i, err := s.spatial.EIO.IndustryIndex(in.ProductionSector)
	if err != nil {
		return err
	}
	spatialRef := spatialRefs[i]
	for i, scc := range spatialRef.SCCs {
		desc, ok := s.sccDescriptions[string(scc)]
		if !ok {
			return fmt.Errorf("missing description for SCC %s", scc)
		}
		err := stream.Send(&eiopb.SCCInfo{
			SCC:  string(scc),
			Frac: float32(spatialRef.SCCFractions[i]),
			Desc: desc,
		})
		if err != nil {
			return err
		}
	}
	s.Log.WithFields(logrus.Fields{
		"ProductionSector": in.ProductionSector,
	}).Info("eioserve finished generating SCCs")
	return nil
}

// MapInfo returns the grid cell colors and a legend for the given selection.
func (s *Server) MapInfo(ctx context.Context, in *eiopb.Selection) (*eiopb.ColorInfo, error) {
	s.Log.WithFields(logrus.Fields{
		"DemandGroup":      in.DemandGroup,
		"DemandSector":     in.DemandSector,
		"ProductionGroup":  in.ProductionGroup,
		"ProductionSector": in.ProductionSector,
		"ImpactType":       in.ImpactType,
		"DemandType":       in.DemandType,
	}).Info("eioserve generating MapInfo")
	out := new(eiopb.ColorInfo)
	commodityMask, err := s.demandMask(in.DemandGroup, in.DemandSector)
	if err != nil {
		return nil, err
	}
	industryMask, err := s.productionMask(in.ProductionGroup, in.ProductionSector)
	if err != nil {
		return nil, err
	}
	impacts, err := s.impactsMap(ctx, in, commodityMask, industryMask)
	if err != nil {
		return nil, err
	}

	cm := moreland.ExtendedBlackBody()
	min := mat.Min(impacts)
	max := mat.Max(impacts)
	cutpt := percentile(impacts, 0.999)
	cm.SetMin(min)
	cm.SetMax(cutpt)

	cm2, err := moreland.NewLuminance([]color.Color{
		color.NRGBA{G: 176, A: 255},
		color.NRGBA{G: 255, A: 255},
	})
	if err != nil {
		log.Panic(err)
	}
	cm2.SetMin(cutpt)
	cm2.SetMax(max)

	rows, _ := impacts.Dims()
	out.RGB = make([][]byte, rows)
	for i := 0; i < rows; i++ {
		v := impacts.At(i, 0)
		c, err := cm.At(v)
		if err != nil {
			if err == palette.ErrOverflow {
				c, err = cm2.At(v)
				if err != nil {
					panic(err)
				}
			} else {
				panic(err)
			}
		}
		col := color.NRGBAModel.Convert(c).(color.NRGBA)
		out.RGB[i] = []byte{col.R, col.G, col.B}
	}
	out.Legend = legend(cm, cm2, cutpt, max)
	s.Log.WithFields(logrus.Fields{
		"DemandGroup":      in.DemandGroup,
		"DemandSector":     in.DemandSector,
		"ProductionGroup":  in.ProductionGroup,
		"ProductionSector": in.ProductionSector,
		"ImpactType":       in.ImpactType,
		"DemandType":       in.DemandType,
	}).Info("eioserve finished generating MapInfo")
	return out, nil
}

func legend(cm, cm2 palette.ColorMap, cutpt, max float64) string {
	p, err := plot.New()
	if err != nil {
		log.Panic(err)
	}
	l := &plotter.ColorBar{
		ColorMap: cm,
	}
	p.Add(l)
	p.HideY()
	p.X.Padding = 0

	p2, err := plot.New()
	if err != nil {
		log.Panic(err)
	}
	l2 := &plotter.ColorBar{
		ColorMap: cm2,
	}
	p2.Add(l2)
	p2.HideY()
	p2.X.Tick.Marker = minMax{}

	p2.X.Padding = 0

	img := vgimg.New(300, 40)
	dc := draw.New(img)
	dc1, dc2 := splitHorizontal(dc, vg.Points(265))
	p.Draw(dc1)
	p2.Draw(dc2)
	b := new(bytes.Buffer)
	png := vgimg.PngCanvas{Canvas: img}
	if _, err := png.WriteTo(b); err != nil {
		panic(err)
	}
	return base64.StdEncoding.EncodeToString(b.Bytes())
}

type minMax struct{}

func (m minMax) Ticks(min, max float64) []plot.Tick {
	return []plot.Tick{
		plot.Tick{
			Value: min,
			Label: fmt.Sprintf("%.3g", min),
		},
		plot.Tick{
			Value: max,
			Label: fmt.Sprintf("%.3g", max),
		},
	}
}

// splitHorizontal splits c at x
func splitHorizontal(c draw.Canvas, x vg.Length) (left, right draw.Canvas) {
	return draw.Crop(c, 0, c.Min.X-c.Max.X+x, 0, 0), draw.Crop(c, x, 0, 0, 0)
}

// percentile returns percentile p (range [0,1]) of the given data.
func percentile(data *mat.VecDense, p float64) float64 {
	rows, _ := data.Dims()
	tmp := make([]float64, rows)
	for i := 0; i < rows; i++ {
		tmp[i] = data.At(i, 0)
	}
	sort.Float64s(tmp)
	return tmp[roundInt(p*float64(rows))-1]
}

// roundInt rounds a float to an integer
func roundInt(x float64) int {
	return int(x + 0.5)
}

func init() {
	gob.Register([]*eiopb.Rectangle{})
}

func (s *Server) getGeometry(ctx context.Context, _ interface{}) (interface{}, error) {
	const (
		inProj  = "+proj=lcc +lat_1=33.000000 +lat_2=45.000000 +lat_0=40.000000 +lon_0=-97.000000 +x_0=0 +y_0=0 +a=6370997.000000 +b=6370997.000000 +to_meter=1"
		outProj = "+proj=longlat"
	)
	inSR, err := proj.Parse(inProj)
	if err != nil {
		return nil, err
	}
	outSR, err := proj.Parse(outProj)
	if err != nil {
		return nil, err
	}
	ct, err := inSR.NewTransform(outSR)
	if err != nil {
		return nil, err
	}

	g, err := s.spatial.CSTConfig.Geometry()
	if err != nil {
		return nil, err
	}
	o := make([]*eiopb.Rectangle, len(g))
	for i, gg := range g {
		gT, err := gg.Transform(ct)
		if err != nil {
			return nil, err
		}
		gr := gT.(geom.Polygon)[0]
		o[i] = &eiopb.Rectangle{
			LL: &eiopb.Point{X: float32(gr[0].X), Y: float32(gr[0].Y)},
			LR: &eiopb.Point{X: float32(gr[1].X), Y: float32(gr[1].Y)},
			UR: &eiopb.Point{X: float32(gr[2].X), Y: float32(gr[2].Y)},
			UL: &eiopb.Point{X: float32(gr[3].X), Y: float32(gr[3].Y)},
		}
	}
	return o, nil
}

// loadCacheOnce inititalizes a request cache.
func loadCacheOnce(f requestcache.ProcessFunc, workers, memCacheSize int, cacheLoc string, marshal func(interface{}) ([]byte, error), unmarshal func([]byte) (interface{}, error)) *requestcache.Cache {
	if cacheLoc == "" {
		return requestcache.NewCache(f, workers, requestcache.Deduplicate(),
			requestcache.Memory(memCacheSize))
	} else if strings.HasPrefix(cacheLoc, "http") {
		return requestcache.NewCache(f, workers, requestcache.Deduplicate(),
			requestcache.Memory(memCacheSize), requestcache.HTTP(cacheLoc, unmarshal))
	}
	return requestcache.NewCache(f, workers, requestcache.Deduplicate(),
		requestcache.Memory(memCacheSize), requestcache.Disk(cacheLoc, marshal, unmarshal))
}

// Geometry returns the InMAP grid geometry in the Google mercator projection.
func (s *Server) Geometry(_ *eiopb.Selection, stream eiopb.EIOServe_GeometryServer) error {
	s.Log.Info("eioserve generating Geometry")
	s.geomCacheOnce.Do(func() {
		s.geomCache = loadCacheOnce(s.getGeometry, 1, 1, s.spatial.SpatialCache,
			requestcache.MarshalGob, requestcache.UnmarshalGob)
	})
	req := s.geomCache.NewRequest(context.Background(), struct{}{}, "geometry")
	iface, err := req.Result()
	if err != nil {
		s.Log.WithError(err).Errorf("generating/retrieving geometry")
		return err
	}
	out := iface.([]*eiopb.Rectangle)
	for _, r := range out {
		if err := stream.Send(r); err != nil {
			s.Log.WithError(err).Errorf("sending geometry")
			return err
		}
	}
	s.Log.Info("eioserve finished generating Geometry")
	return nil
}

// inverseArea returns the inverse of the area of each grid cell in km^-2.
func (s *Server) inverseArea() (*mat.VecDense, error) {
	f := func(ctx context.Context, requestPayload interface{}) (resultPayload interface{}, err error) {
		g, err := s.spatial.CSTConfig.Geometry()
		if err != nil {
			return nil, err
		}
		area := mat.NewVecDense(len(g), nil)
		for i, c := range g {
			area.SetVec(i, 1e6/c.Area()) // convert m^2 to km^-2
		}
		return area, nil
	}
	s.areaCacheOnce.Do(func() {
		s.areaCache = loadCacheOnce(f, 1, 1, s.spatial.SpatialCache,
			vectorMarshal, vectorUnmarshal)
	})
	req := s.areaCache.NewRequest(context.Background(), nil, "grid_area")
	iface, err := req.Result()
	if err != nil {
		return nil, err
	}
	return iface.(*mat.VecDense), nil
}

// vectorMarshal converts a vector to a byte array for storing in a cache.
func vectorMarshal(data interface{}) ([]byte, error) {
	i := data.(*interface{})
	m := (*i).(*mat.VecDense)
	return m.MarshalBinary()
}

// vectorUnmarshal converts a byte array to a vector after storing it in a cache.
func vectorUnmarshal(b []byte) (interface{}, error) {
	m := new(mat.VecDense)
	err := m.UnmarshalBinary(b)
	return m, err
}
