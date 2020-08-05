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

package eieio

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/gob"
	"fmt"
	"image/color"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/ctessum/geom"
	"github.com/ctessum/geom/proj"
	"github.com/ctessum/plotextra"
	"github.com/ctessum/requestcache"
	"github.com/gorilla/websocket"
	"github.com/improbable-eng/grpc-web/go/grpcweb"
	"github.com/sirupsen/logrus"
	"github.com/spatialmodel/inmap/emissions/slca/eieio/ces"
	"github.com/spatialmodel/inmap/internal/hash"
	"gonum.org/v1/gonum/mat"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/palette"
	"gonum.org/v1/plot/palette/moreland"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg/draw"
	"gonum.org/v1/plot/vg/vgimg"
	"google.golang.org/grpc"

	"github.com/spatialmodel/inmap/emissions/slca"
	"github.com/spatialmodel/inmap/emissions/slca/eieio/eieiorpc"
	"github.com/spatialmodel/inmap/epi"
)

type config struct {
	EIO Config
	slca.CSTConfig
	AggregatorFile string
	SpatialRefFile string
	DefaultYear    Year
}

// Server is a server for EIO LCA model simulation data.
type Server struct {
	*SpatialEIO
	*ces.CES
	ioAgg  *Aggregator
	sccAgg *Aggregator

	defaultYear Year

	geomCache, areaCache         *requestcache.Cache
	geomCacheOnce, areaCacheOnce sync.Once

	grpcServer   *grpcweb.WrappedGrpcServer
	staticServer http.Handler

	Log logrus.FieldLogger

	prefix string
}

type ServerConfig struct {
	SpatialConfig

	// IOAggregatorFile is the path to the xlsx file containing IO sector
	// aggregation information.
	IOAggregatorFile string

	// SCCAggregatorFile is the path to the xlsx file containing SCC
	// aggregation information.
	SCCAggregatorFile string

	// StaticDir is the path to the directory containing the static
	// assets for the website.
	StaticDir string

	// DefaultYear specifies the default analysis year.
	DefaultYear Year

	// CESDataDir is the path to the directory holding CES data,
	// e.g. ${INMAP_ROOT_DIR}/emissions/slca/eieio/ces/data
	CESDataDir string
}

// NewServer creates a new EIO-LCA server, where hr represents the hazard ratio
// functions to be used.
func NewServer(c *ServerConfig, prefix string, hr ...epi.HRer) (*Server, error) {
	s, err := NewSpatial(&c.SpatialConfig, hr...)
	if err != nil {
		return nil, fmt.Errorf("eioserve: creating server: %v", err)
	}

	ioa, err := s.EIO.NewIOAggregator(os.ExpandEnv(c.IOAggregatorFile))
	if err != nil {
		return nil, fmt.Errorf("eioserve: creating IO aggregator: %v", err)
	}
	scca, err := s.NewSCCAggregator(os.ExpandEnv(c.SCCAggregatorFile))
	if err != nil {
		return nil, fmt.Errorf("eioserve: creating SCC aggregator: %v", err)
	}
	model := &Server{
		ioAgg:       ioa,
		sccAgg:      scca,
		SpatialEIO:  s,
		defaultYear: c.DefaultYear,
		Log:         logrus.StandardLogger(),
		prefix:      prefix,
	}
	model.CES, err = ces.NewCES(model, c.CESDataDir)
	if err != nil {
		return nil, fmt.Errorf("eioserve: loading demographic consumption: %v", err)
	}

	grpcServer := grpc.NewServer(grpc.MaxSendMsgSize(3.0e9)) // Send messages up to 3 gb.
	eieiorpc.RegisterEIEIOrpcServer(grpcServer, model)

	model.grpcServer = grpcweb.WrapServer(grpcServer, grpcweb.WithWebsockets(true))

	model.staticServer = http.FileServer(http.Dir(os.ExpandEnv(c.StaticDir)))

	return model, nil
}

func isStatic(u *url.URL) bool {
	staticExtentions := map[string]struct{}{
		".js":     struct{}{},
		".css":    struct{}{},
		".png":    struct{}{},
		".gif":    struct{}{},
		".jpg":    struct{}{},
		".jpeg":   struct{}{},
		".js.map": struct{}{},
		".map":    struct{}{},
		".ico":    struct{}{},
	}
	_, ok := staticExtentions[strings.ToLower(filepath.Ext(u.Path))]
	return ok
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.ProtoMajor == 2 && strings.Contains(r.Header.Get("Content-Type"), "application/grpc") || websocket.IsWebSocketUpgrade(r) {
		s.Log.WithFields(logrus.Fields{
			"url":  r.URL.String(),
			"addr": r.RemoteAddr,
		}).Info("eioserve grpc request")
		s.grpcServer.ServeHTTP(w, r)
	} else if isStatic(r.URL) {
		s.Log.WithFields(logrus.Fields{
			"url":  r.URL.String(),
			"addr": r.RemoteAddr,
		}).Info("eioserve static request")
		r.URL.Path = r.URL.Path[len(s.prefix):]
		s.staticServer.ServeHTTP(w, r)
	} else {
		fmt.Fprint(w, `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta http-equiv="X-UA-Compatible" content="IE=edge">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <!-- The above 3 meta tags *must* come first in the head; any other head content must come *after* these tags -->

  <link href="css/gui.css" rel="stylesheet">
  <link href="node_modules/bootstrap/dist/css/bootstrap.min.css" rel="stylesheet">
  <link rel="stylesheet" href="node_modules/leaflet/dist/leaflet.css" />

  <!-- HTML5 shim and Respond.js for IE8 support of HTML5 elements and media queries -->
  <!--[if lt IE 9]>
      <script src="https://oss.maxcdn.com/html5shiv/3.7.3/html5shiv.min.js"></script>
      <script src="https://oss.maxcdn.com/respond/1.4.2/respond.min.js"></script>
  <![endif]-->
</head>
<body>
	<div id="loading" class="loading"></div>
  <script src="node_modules/jquery/dist/jquery.min.js"></script>
  <script src="node_modules/bootstrap/dist/js/bootstrap.min.js"></script>
  <script src="node_modules/leaflet/dist/leaflet.js"></script>
  <script src="glify.js"></script>
  <script src="gui.js"></script>
</body>
</html>`)
	}
}

// EndUseMask returns a mask that can be used to limit a FinalDemand vector
// to demand for end-uses in the end-use group defined by abbrev.
func (s *Server) EndUseMask(ctx context.Context, abbrev *eieiorpc.StringInput) (*eieiorpc.Mask, error) {
	return mask2rpc(s.ioAgg.CommodityMask(abbrev.String_)), nil
}

// EmitterMask returns a mask that can be used to limits impacts to only
// caused by emissions from the SCC codes defined by abbrev.
func (s *Server) EmitterMask(ctx context.Context, abbrev *eieiorpc.StringInput) (*eieiorpc.Mask, error) {
	return mask2rpc(s.sccAgg.IndustryMask(abbrev.String_)), nil
}

// EndUseGroupNames returns the names of the end-use groups.
func (s *Server) EndUseGroupNames(ctx context.Context, _ *eieiorpc.StringInput) (*eieiorpc.StringList, error) {
	return &eieiorpc.StringList{List: s.ioAgg.Names()}, nil
}

// EndUseGroupAbbrevs returns the abbreviations of the end-use groups.
func (s *Server) EndUseGroupAbbrevs(ctx context.Context, _ *eieiorpc.StringInput) (*eieiorpc.StringList, error) {
	return &eieiorpc.StringList{List: s.ioAgg.Abbreviations()}, nil
}

// EmitterGroupNames returns the names of the emitter groups.
func (s *Server) EmitterGroupNames(ctx context.Context, _ *eieiorpc.StringInput) (*eieiorpc.StringList, error) {
	return &eieiorpc.StringList{List: s.sccAgg.Names()}, nil
}

// EmitterGroupAbbrevs returns the abbreviations of the emitter groups.
func (s *Server) EmitterGroupAbbrevs(ctx context.Context, _ *eieiorpc.StringInput) (*eieiorpc.StringList, error) {
	return &eieiorpc.StringList{List: s.sccAgg.Abbreviations()}, nil
}

func (s *Server) impactsMenu(ctx context.Context, selection *eieiorpc.Selection, commodityMask, industryMask *Mask) (*eieiorpc.Vector, error) {
	demand, err := s.SpatialEIO.EIO.FinalDemand(ctx, &eieiorpc.FinalDemandInput{
		FinalDemandType: selection.FinalDemandType,
		EndUseMask:      mask2rpc(commodityMask),
		Year:            selection.Year,
		Location:        eieiorpc.Location_Domestic,
	})
	if err != nil {
		return nil, fmt.Errorf("eioserve: calculating final demand: %v", err)
	}
	switch selection.ImpactType {
	case "health", "conc":
		return s.SpatialEIO.Health(ctx, &eieiorpc.HealthInput{
			Demand:      demand,
			EmitterMask: mask2rpc(industryMask),
			Pollutant:   selection.GetPollutant(),
			Population:  selection.Population,
			Year:        selection.Year,
			Location:    eieiorpc.Location_Domestic,
			HR:          "NasariACS",
			AQM:         selection.AQM,
		})
	case "emis":
		return s.SpatialEIO.Emissions(ctx, &eieiorpc.EmissionsInput{
			Demand:   demand,
			Emitters: mask2rpc(industryMask),
			Emission: selection.GetEmission(),
			Year:     selection.Year,
			Location: eieiorpc.Location_Domestic,
			AQM:      selection.AQM,
		})
	default:
		return nil, fmt.Errorf("invalid impact type request: %s", selection.ImpactType)
	}
}

func (s *Server) impactsMap(ctx context.Context, selection *eieiorpc.Selection, commodityMask, industryMask *Mask) (*eieiorpc.Vector, error) {
	demand, err := s.SpatialEIO.EIO.FinalDemand(ctx, &eieiorpc.FinalDemandInput{
		FinalDemandType: selection.FinalDemandType,
		EndUseMask:      mask2rpc(commodityMask),
		Year:            selection.Year,
		Location:        eieiorpc.Location_Domestic,
	})
	if err != nil {
		return nil, fmt.Errorf("eioserve: calculating final demand: %v", err)
	}
	switch selection.ImpactType {
	case "health":
		h, err := s.SpatialEIO.Health(ctx, &eieiorpc.HealthInput{
			Demand:      demand,
			EmitterMask: mask2rpc(industryMask),
			Pollutant:   selection.GetPollutant(),
			Population:  selection.Population,
			Year:        selection.Year,
			Location:    eieiorpc.Location_Domestic,
			HR:          "NasariACS",
			AQM:         selection.AQM,
		})
		return s.perArea(h, selection.AQM, err)
	case "conc":
		return s.SpatialEIO.Concentrations(ctx, &eieiorpc.ConcentrationInput{
			Demand:    demand,
			Emitters:  mask2rpc(industryMask),
			Pollutant: selection.GetPollutant(),
			Year:      selection.Year,
			Location:  eieiorpc.Location(Domestic),
			AQM:       selection.AQM,
		})
	case "emis":
		e, err := s.SpatialEIO.Emissions(ctx, &eieiorpc.EmissionsInput{
			Demand:   demand,
			Emitters: mask2rpc(industryMask),
			Emission: selection.GetEmission(),
			Year:     selection.Year,
			Location: eieiorpc.Location_Domestic,
			AQM:      selection.AQM,
		})
		return s.perArea(e, selection.AQM, err)
	default:
		return nil, fmt.Errorf("invalid impact type request: %s", selection.ImpactType)
	}
}

func (s *Server) perArea(v *eieiorpc.Vector, aqm string, err error) (*eieiorpc.Vector, error) {
	if err != nil {
		return nil, err
	}
	area, err := s.inverseArea(aqm)
	if err != nil {
		return nil, err
	}
	v2 := array2vec(v.Data)
	v2.MulElemVec(v2, area)
	return vec2rpc(v2), nil
}

// EndUseGroups returns the available demand groups.
func (s *Server) EndUseGroups(ctx context.Context, in *eieiorpc.Selection) (*eieiorpc.Selectors, error) {
	s.Log.WithFields(logrus.Fields{
		"DemandGroup":      in.EndUseGroup,
		"DemandSector":     in.EndUseSector,
		"ProductionGroup":  in.EmitterGroup,
		"ProductionSector": in.EmitterSector,
		"ImpactType":       in.ImpactType,
		"FinalDemandType":  in.FinalDemandType,
	}).Info("eioserve generating EndUseGroups")
	productionMask, err := s.productionMask(in.EmitterGroup, in.EmitterSector)
	if err != nil {
		return nil, err
	}
	out := &eieiorpc.Selectors{
		Names:  make([]string, len(s.ioAgg.Names())+1),
		Values: make([]float32, len(s.ioAgg.Names())+1),
	}
	out.Names[0] = All
	// impacts produced by selected sectors owing to all sectors.
	impacts, err := s.impactsMenu(ctx, in, nil, productionMask)
	if err != nil {
		return nil, err
	}
	out.Values[0] = float32(mat.Sum(array2vec(impacts.Data)))
	i := 1
	for _, g := range s.ioAgg.Names() {
		out.Names[i] = g
		// impacts produced by selected sectors owing to consumption in this group
		// of sectors.
		mask, err := s.demandMask(g, All)
		if err != nil {
			return nil, err
		}
		impacts, err := s.impactsMenu(ctx, in, mask, productionMask)
		if err != nil {
			return nil, err
		}
		out.Values[i] = float32(mat.Sum(array2vec(impacts.Data)))
		i++
	}
	sorter := selectorSorter(*out)
	sort.Sort(&sorter)
	s.Log.WithFields(logrus.Fields{
		"DemandGroup":      in.EndUseGroup,
		"DemandSector":     in.EndUseSector,
		"ProductionGroup":  in.EmitterGroup,
		"ProductionSector": in.EmitterSector,
		"ImpactType":       in.ImpactType,
		"FinalDemandType":  in.FinalDemandType,
	}).Info("eioserve finished generating EndUseGroups")
	return out, nil
}

func commodityGroup(e *EIO, m *Mask) []string {
	var o []string
	v := (mat.VecDense)(*m)
	for i, c := range e.commodities {
		if v.At(i, 0) != 0 {
			o = append(o, c)
		}
	}
	return o
}

func sccGroup(s *SpatialEIO, m *Mask) (codes, descriptions []string) {
	v := (mat.VecDense)(*m)
	for i, c := range s.SCCs {
		if v.At(i, 0) != 0 {
			d, err := s.SCCDescription(i)
			if err != nil {
				panic(err)
			}
			descriptions = append(descriptions, d)
			codes = append(codes, string(c))
		}
	}
	return
}

// EndUseSectors returns the available demand sectors.
func (s *Server) EndUseSectors(ctx context.Context, in *eieiorpc.Selection) (*eieiorpc.Selectors, error) {
	s.Log.WithFields(logrus.Fields{
		"DemandGroup":      in.EndUseGroup,
		"DemandSector":     in.EndUseSector,
		"ProductionGroup":  in.EmitterGroup,
		"ProductionSector": in.EmitterSector,
		"ImpactType":       in.ImpactType,
		"DemandType":       in.FinalDemandType,
	}).Info("eioserve generating EndUseSectors")
	out := &eieiorpc.Selectors{Names: []string{All}}
	productionMask, err := s.productionMask(in.EmitterGroup, in.EmitterSector)
	if err != nil {
		return nil, err
	}
	if in.EndUseGroup == All {
		// impacts produced by all sectors owing to all sectors.
		impacts, err := s.impactsMenu(ctx, in, nil, productionMask)
		if err != nil {
			return nil, err
		}
		out.Values = []float32{float32(mat.Sum(array2vec(impacts.Data)))}
		return out, nil
	}
	mask, err := s.demandMask(in.EndUseGroup, All)
	if err != nil {
		return nil, err
	}
	impacts, err := s.impactsMenu(ctx, in, mask, productionMask)
	if err != nil {
		return nil, err
	}
	out.Values = []float32{float32(mat.Sum(array2vec(impacts.Data)))}

	sectors := commodityGroup(&s.SpatialEIO.EIO, mask)
	out.Names = append(out.Names, sectors...)
	temp := make([]float32, len(sectors))
	out.Values = append(out.Values, temp...)
	for i, sector := range sectors {
		// impacts produced by all sectors owing to consumption in this sector.
		mask, err := s.demandMask(in.EndUseGroup, sector)
		if err != nil {
			return nil, err
		}
		impacts, err := s.impactsMenu(ctx, in, mask, productionMask)
		if err != nil {
			return nil, err
		}
		out.Values[i+1] = float32(mat.Sum(array2vec(impacts.Data)))
	}
	sorter := selectorSorter(*out)
	sort.Sort(&sorter)
	s.Log.WithFields(logrus.Fields{
		"DemandGroup":      in.EndUseGroup,
		"DemandSector":     in.EndUseSector,
		"ProductionGroup":  in.EmitterGroup,
		"ProductionSector": in.EmitterSector,
		"ImpactType":       in.ImpactType,
		"FinalDemandType":  in.FinalDemandType,
	}).Info("eioserve finished generating EndUseSectors")
	return out, nil
}

// demandMask returns a commodity mask corresponding to the
// given selection.
func (s *Server) demandMask(demandGroup, demandSector string) (*Mask, error) {
	if demandGroup == All {
		return nil, nil
	} else if demandSector == All {
		// demand from a group of sectors.
		abbrev, err := s.ioAgg.Abbreviation(demandGroup)
		if err != nil {
			return nil, fmt.Errorf("eioserve: getting abbreviation for demand mask: %v", err)
		}
		return s.ioAgg.CommodityMask(abbrev), nil
	}
	// demand from a single sector.
	return s.SpatialEIO.EIO.CommodityMask(demandSector)
}

// productionMask returns a commodity mask corresponding to the
// given selection.
func (s *Server) productionMask(productionGroup, productionSector string) (*Mask, error) {
	if productionGroup == All {
		return nil, nil
	} else if productionSector == All {
		// demand from a group of sectors.
		abbrev, err := s.sccAgg.Abbreviation(productionGroup)
		if err != nil {
			return nil, fmt.Errorf("eioserve: getting abbreviation for production mask: %v", err)
		}
		return s.sccAgg.IndustryMask(abbrev), nil
	}
	// demand from a single sector.
	return s.SpatialEIO.SCCMask(slca.SCC(productionSector))
}

// EmitterGroups returns the available emitter groups.
func (s *Server) EmitterGroups(ctx context.Context, in *eieiorpc.Selection) (*eieiorpc.Selectors, error) {
	s.Log.WithFields(logrus.Fields{
		"DemandGroup":      in.EndUseGroup,
		"DemandSector":     in.EndUseSector,
		"ProductionGroup":  in.EmitterGroup,
		"ProductionSector": in.EmitterSector,
		"ImpactType":       in.ImpactType,
		"DemandType":       in.FinalDemandType,
	}).Info("eioserve generating EmitterGroups")
	demandMask, err := s.demandMask(in.EndUseGroup, in.EndUseSector)
	if err != nil {
		return nil, err
	}
	out := &eieiorpc.Selectors{
		Names:  make([]string, len(s.sccAgg.Names())+1),
		Values: make([]float32, len(s.sccAgg.Names())+1),
	}
	out.Names[0] = All
	v, err := s.impactsMenu(ctx, in, demandMask, nil)
	if err != nil {
		return nil, err
	}
	out.Values[0] = float32(mat.Sum(array2vec(v.Data)))
	i := 1
	for _, g := range s.sccAgg.Names() {
		out.Names[i] = g
		mask, err := s.productionMask(g, All)
		if err != nil {
			return nil, err
		}
		v, err := s.impactsMenu(ctx, in, demandMask, mask)
		if err != nil {
			return nil, err
		}
		out.Values[i] = float32(mat.Sum(array2vec(v.Data)))
		i++
	}
	sorter := selectorSorter(*out)
	sort.Sort(&sorter)
	s.Log.WithFields(logrus.Fields{
		"DemandGroup":      in.EndUseGroup,
		"DemandSector":     in.EndUseSector,
		"ProductionGroup":  in.EmitterGroup,
		"ProductionSector": in.EmitterSector,
		"ImpactType":       in.ImpactType,
		"FinalDemandType":  in.FinalDemandType,
	}).Info("eioserve finished generating EmitterGroups")
	return out, nil
}

// EmitterSectors returns the available production sectors.
func (s *Server) EmitterSectors(ctx context.Context, in *eieiorpc.Selection) (*eieiorpc.Selectors, error) {
	s.Log.WithFields(logrus.Fields{
		"DemandGroup":      in.EndUseGroup,
		"DemandSector":     in.EndUseSector,
		"ProductionGroup":  in.EmitterGroup,
		"ProductionSector": in.EmitterSector,
		"ImpactType":       in.ImpactType,
		"FinailDemandType": in.FinalDemandType,
	}).Info("eioserve generating EmitterSectors")
	demandMask, err := s.demandMask(in.EndUseGroup, in.EndUseSector)
	if err != nil {
		return nil, err
	}

	out := &eieiorpc.Selectors{Names: []string{All}, Codes: []string{All}}
	if in.EmitterGroup == All {
		v, err2 := s.impactsMenu(ctx, in, demandMask, nil)
		if err2 != nil {
			return nil, err2
		}
		out.Values = []float32{float32(mat.Sum(array2vec(v.Data)))}
		return out, nil
	}
	mask, err := s.productionMask(in.EmitterGroup, All)
	if err != nil {
		return nil, err
	}
	v, err := s.impactsMenu(ctx, in, demandMask, mask)
	if err != nil {
		return nil, err
	}
	out.Values = []float32{float32(mat.Sum(array2vec(v.Data)))}
	sectors, descriptions := sccGroup(s.SpatialEIO, mask)
	out.Names = append(out.Names, descriptions...)
	out.Codes = append(out.Codes, sectors...)
	temp := make([]float32, len(sectors))
	out.Values = append(out.Values, temp...)
	for i, sector := range sectors {
		mask, err := s.productionMask(in.EmitterGroup, sector)
		if err != nil {
			return nil, err
		}
		v, err := s.impactsMenu(ctx, in, demandMask, mask)
		if err != nil {
			return nil, err
		}
		out.Values[i+1] = float32(mat.Sum(array2vec(v.Data)))
	}
	sorter := selectorSorter(*out)
	sort.Sort(&sorter)
	s.Log.WithFields(logrus.Fields{
		"DemandGroup":      in.EndUseGroup,
		"DemandSector":     in.EndUseSector,
		"ProductionGroup":  in.EmitterGroup,
		"ProductionSector": in.EmitterSector,
		"ImpactType":       in.ImpactType,
		"FinalDemandType":  in.FinalDemandType,
	}).Info("eioserve finished generating EmitterSectors")
	return out, nil
}

// MapInfo returns the grid cell colors and a legend for the given selection.
func (s *Server) MapInfo(ctx context.Context, in *eieiorpc.Selection) (*eieiorpc.ColorInfo, error) {
	s.Log.WithFields(logrus.Fields{
		"DemandGroup":      in.EndUseGroup,
		"DemandSector":     in.EndUseSector,
		"ProductionGroup":  in.EmitterGroup,
		"ProductionSector": in.EmitterSector,
		"ImpactType":       in.ImpactType,
		"DemandType":       in.FinalDemandType,
	}).Info("eioserve generating MapInfo")
	out := new(eieiorpc.ColorInfo)
	commodityMask, err := s.demandMask(in.EndUseGroup, in.EndUseSector)
	if err != nil {
		return nil, err
	}
	industryMask, err := s.productionMask(in.EmitterGroup, in.EmitterSector)
	if err != nil {
		return nil, err
	}
	impactsRPC, err := s.impactsMap(ctx, in, commodityMask, industryMask)
	if err != nil {
		return nil, err
	}
	impacts := array2vec(impactsRPC.Data)

	cm1 := moreland.ExtendedBlackBody()
	cm2, err := moreland.NewLuminance([]color.Color{
		color.NRGBA{G: 176, A: 255},
		color.NRGBA{G: 255, A: 255},
	})
	if err != nil {
		log.Panic(err)
	}
	cm := &plotextra.BrokenColorMap{
		Base:     cm1,
		OverFlow: palette.Reverse(cm2),
	}
	cm.SetMin(mat.Min(impacts))
	cm.SetMax(mat.Max(impacts))
	cutpt := percentile(impacts, 0.999)
	cm.SetHighCut(cutpt)
	out.Legend = legend(cm, cutpt)

	rows, _ := impacts.Dims()
	out.RGB = make([][]byte, rows)
	for i := 0; i < rows; i++ {
		v := impacts.At(i, 0)
		c, err := cm.At(v)
		if err != nil {
			return nil, fmt.Errorf("eioserve: creating map legend: %v", err)
		}
		col := color.NRGBAModel.Convert(c).(color.NRGBA)
		out.RGB[i] = []byte{col.R, col.G, col.B}
	}
	s.Log.WithFields(logrus.Fields{
		"DemandGroup":      in.EndUseGroup,
		"DemandSector":     in.EndUseSector,
		"ProductionGroup":  in.EmitterGroup,
		"ProductionSector": in.EmitterSector,
		"ImpactType":       in.ImpactType,
		"FinalDemandType":  in.FinalDemandType,
	}).Info("eioserve finished generating MapInfo")
	return out, nil
}

func legend(cm *plotextra.BrokenColorMap, highcut float64) string {
	p, err := plot.New()
	if err != nil {
		log.Panic(err)
	}
	l := &plotter.ColorBar{
		ColorMap: cm,
	}
	p.X.Scale = plotextra.BrokenScale{
		HighCut:         highcut,
		HighCutFraction: 0.9,
	}
	p.X.Tick.Marker = plotextra.BrokenTicks{
		HighCut: highcut,
	}
	p.Add(l)
	p.HideY()
	p.X.Padding = 0

	img := vgimg.New(300, 40)
	dc := draw.New(img)
	p.Draw(dc)
	b := new(bytes.Buffer)
	png := vgimg.PngCanvas{Canvas: img}
	if _, err := png.WriteTo(b); err != nil {
		panic(err)
	}
	return base64.StdEncoding.EncodeToString(b.Bytes())
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
	gob.Register([]*eieiorpc.Rectangle{})
	gob.Register(geom.Polygon{})
}

func (s *Server) getGeometry(ctx context.Context, inputI interface{}) (interface{}, error) {
	const inProj = "+proj=lcc +lat_1=33.000000 +lat_2=45.000000 +lat_0=40.000000 +lon_0=-97.000000 +x_0=0 +y_0=0 +a=6370997.000000 +b=6370997.000000 +to_meter=1"
	input := inputI.(*eieiorpc.GeometryInput)
	inSR, err := proj.Parse(inProj)
	if err != nil {
		return nil, fmt.Errorf("eioserve: getting geometry: %v", err)
	}
	outSR, err := proj.Parse(input.SpatialReference)
	if err != nil {
		return nil, fmt.Errorf("eioserve: getting geometry: %v", err)
	}
	ct, err := inSR.NewTransform(outSR)
	if err != nil {
		return nil, fmt.Errorf("eioserve: getting geometry: %v", err)
	}

	g, err := s.SpatialEIO.CSTConfig.Geometry(input.AQM)
	if err != nil {
		return nil, fmt.Errorf("eioserve: getting geometry: %v", err)
	}
	o := make([]*eieiorpc.Rectangle, len(g))
	for i, gg := range g {
		gT, err := gg.Transform(ct)
		if err != nil {
			return nil, fmt.Errorf("eioserve: getting geometry: %v", err)
		}
		gr := gT.(geom.Polygon)[0]
		o[i] = &eieiorpc.Rectangle{
			LL: &eieiorpc.Point{X: float32(gr[0].X), Y: float32(gr[0].Y)},
			LR: &eieiorpc.Point{X: float32(gr[1].X), Y: float32(gr[1].Y)},
			UR: &eieiorpc.Point{X: float32(gr[2].X), Y: float32(gr[2].Y)},
			UL: &eieiorpc.Point{X: float32(gr[3].X), Y: float32(gr[3].Y)},
		}
	}
	return o, nil
}

// Geometry returns the InMAP grid geometry, ,
// where SpatialReference specifies the desired projection in WKT or PROJ4
// format.
func (s *Server) Geometry(ctx context.Context, input *eieiorpc.GeometryInput) (*eieiorpc.Rectangles, error) {
	s.Log.Info("eioserve generating Geometry")
	s.geomCacheOnce.Do(func() {
		s.geomCache = loadCacheOnce(s.getGeometry, 1, 1, s.SpatialEIO.EIEIOCache,
			requestcache.MarshalGob, requestcache.UnmarshalGob)
	})
	key := fmt.Sprintf("geometry_%s", hash.Hash(input))
	req := s.geomCache.NewRequest(ctx, input, key)
	iface, err := req.Result()
	if err != nil {
		s.Log.WithError(err).Errorf("generating/retrieving geometry")
		return nil, err
	}
	out := iface.([]*eieiorpc.Rectangle)
	s.Log.Info("eioserve finished generating Geometry")
	return &eieiorpc.Rectangles{Rectangles: out}, nil
}

// inverseArea returns the inverse of the area of each grid cell in km^-2.
func (s *Server) inverseArea(aqm string) (*mat.VecDense, error) {
	f := func(ctx context.Context, aqmI interface{}) (resultPayload interface{}, err error) {
		g, err := s.SpatialEIO.CSTConfig.Geometry(aqmI.(string))
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
		s.areaCache = loadCacheOnce(f, 1, 1, s.SpatialEIO.EIEIOCache,
			vectorMarshal, vectorUnmarshal)
	})
	req := s.areaCache.NewRequest(context.Background(), aqm, "grid_area_"+aqm)
	iface, err := req.Result()
	if err != nil {
		return nil, err
	}
	return iface.(*mat.VecDense), nil
}

func (s *Server) DefaultSelection(ctx context.Context, in *eieiorpc.Selection) (*eieiorpc.Selection, error) {
	return &eieiorpc.Selection{
		EndUseGroup:     All,
		EndUseSector:    All,
		EmitterGroup:    All,
		EmitterSector:   All,
		ImpactType:      "conc",
		FinalDemandType: eieiorpc.FinalDemandType_AllDemand,
		Year:            int32(s.defaultYear),
		Population:      s.SpatialEIO.CSTConfig.CensusPopColumns[0],
		Pol:             &eieiorpc.Selection_Pollutant{eieiorpc.Pollutant_TotalPM25},
		AQM:             "isrm",
	}, nil
}

func (s *Server) Populations(ctx context.Context, _ *eieiorpc.Selection) (*eieiorpc.Selectors, error) {
	return &eieiorpc.Selectors{Names: s.SpatialEIO.CSTConfig.CensusPopColumns}, nil
}

func (s *Server) Years(ctx context.Context, _ *eieiorpc.Selection) (*eieiorpc.Year, error) {
	o := &eieiorpc.Year{Years: make([]int32, len(s.SpatialEIO.EIO.Years()))}
	for i, y := range s.SpatialEIO.EIO.Years() {
		o.Years[i] = int32(y)
	}
	return o, nil
}

// All specifies that all sectors are to be considered
const All = "All"

type selectorSorter eieiorpc.Selectors

// Len fulfils sort.Sort.
func (s *selectorSorter) Len() int { return len(s.Names) }

// Less fulfils sort.Sort.
func (s *selectorSorter) Less(i, j int) bool {
	if s.Names[i] == All {
		return true
	}
	if s.Names[j] == All {
		return false
	}
	return s.Values[i] > s.Values[j]
}

// Swap fulfills sort.Sort.
func (s *selectorSorter) Swap(i, j int) {
	s.Names[i], s.Names[j] = s.Names[j], s.Names[i]
	s.Values[i], s.Values[j] = s.Values[j], s.Values[i]
	if len(s.Codes) == len(s.Names) {
		s.Codes[i], s.Codes[j] = s.Codes[j], s.Codes[i]
	}
}

func array2vec(d []float64) *mat.VecDense {
	if len(d) == 0 {
		return nil
	}
	return mat.NewVecDense(len(d), d)
}

func rpc2vec(d *eieiorpc.Vector) *mat.VecDense {
	if d == nil {
		return nil
	}
	return array2vec(d.Data)
}

func vec2array(v *mat.VecDense) []float64 {
	if v == nil {
		return nil
	}
	return v.RawVector().Data
}

func mask2rpc(m *Mask) *eieiorpc.Mask {
	if m == nil {
		return nil
	}
	return &eieiorpc.Mask{Data: vec2array((*mat.VecDense)(m))}
}

func rpc2mask(m *eieiorpc.Mask) *Mask {
	if m == nil {
		return nil
	}
	return (*Mask)(array2vec(m.Data))
}

func vec2rpc(v *mat.VecDense) *eieiorpc.Vector {
	return &eieiorpc.Vector{Data: vec2array(v)}
}

func mat2rpc(m *mat.Dense) *eieiorpc.Matrix {
	r, c := m.Dims()
	return &eieiorpc.Matrix{Rows: int32(r), Cols: int32(c), Data: m.RawMatrix().Data}
}

func rpc2mat(m *eieiorpc.Matrix) *mat.Dense {
	return mat.NewDense(int(m.Rows), int(m.Cols), m.Data)
}
