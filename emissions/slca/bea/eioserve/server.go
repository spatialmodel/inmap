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
// go get -u github.com/golang/protobuf/protoc-gen-go
// go get -u github.com/johanbrandhorst/protobuf/protoc-gen-gopherjs

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
	"gonum.org/v1/gonum/mat"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/palette/moreland"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg/draw"
	"gonum.org/v1/plot/vg/vgimg"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/spatialmodel/epi"
	"github.com/spatialmodel/inmap/emissions/slca"
	"github.com/spatialmodel/inmap/emissions/slca/bea"
	"github.com/spatialmodel/inmap/emissions/slca/bea/eioserve/proto/eioservepb"
)

type config struct {
	EIO bea.Config
	slca.CSTConfig
	AggregatorFile string
	SpatialRefFile string
	DefaultYear    bea.Year
}

// Server is a server for EIO LCA model simulation data.
type Server struct {
	spatial *bea.SpatialEIO
	ioAgg   *bea.Aggregator
	sccAgg  *bea.Aggregator

	defaultYear bea.Year

	geomCache, areaCache         *requestcache.Cache
	geomCacheOnce, areaCacheOnce sync.Once

	grpcServer   *grpcweb.WrappedGrpcServer
	staticServer http.Handler

	Log logrus.FieldLogger
}

type ServerConfig struct {
	bea.SpatialConfig

	// IOAggregatorFile is the path to the xlsx file containing IO sector
	// aggregation information.
	IOAggregatorFile string

	// SCCAggregatorFile is the path to the xlsx file containing SCC
	// aggregation information.
	SCCAggregatorFile string

	// PEMDir is the path to the director containing SSL information.
	PEMDir string

	// StaticDir is the path to the directory containing the static
	// assets for the website.
	StaticDir string

	// DefaultYear specifies the default analysis year.
	DefaultYear bea.Year
}

// NewServer creates a new EIO-LCA server.
func NewServer(c *ServerConfig) (*Server, error) {
	s, err := bea.NewSpatial(&c.SpatialConfig)
	if err != nil {
		return nil, err
	}

	ioa, err := s.EIO.NewIOAggregator(os.ExpandEnv(c.IOAggregatorFile))
	if err != nil {
		return nil, err
	}
	scca, err := s.NewSCCAggregator(os.ExpandEnv(c.SCCAggregatorFile))
	if err != nil {
		return nil, err
	}
	model := &Server{
		ioAgg:       ioa,
		sccAgg:      scca,
		spatial:     s,
		defaultYear: c.DefaultYear,
		Log:         logrus.StandardLogger(),
	}

	pemDir := os.ExpandEnv(c.PEMDir)
	creds, err := credentials.NewServerTLSFromFile(filepath.Join(pemDir, "cert.pem"), filepath.Join(pemDir, "key.pem"))
	if err != nil {
		return nil, err
	}
	opt := grpc.Creds(creds)
	grpcServer := grpc.NewServer(opt)
	eiopb.RegisterEIOServeServer(grpcServer, model)

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
	}
	fmt.Println(strings.ToLower(filepath.Ext(u.Path)))
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
  <script src="node_modules/jquery/dist/jquery.min.js"></script>
  <script src="node_modules/bootstrap/dist/js/bootstrap.min.js"></script>
  <script src="node_modules/leaflet/dist/leaflet.js"></script>
  <script src="glify.js"></script>
  <script src="gui.js"></script>
</body>
</html>`)
	}
}

func (s *Server) impactsMenu(ctx context.Context, selection *eiopb.Selection, commodityMask, industryMask *bea.Mask) (*mat.VecDense, error) {
	demand, err := s.spatial.EIO.FinalDemand(bea.FinalDemand(selection.DemandType), commodityMask, bea.Year(selection.Year), bea.Domestic)
	if err != nil {
		return nil, err
	}
	switch selection.ImpactType {
	case "health", "conc":
		return s.spatial.Health(ctx, demand, industryMask, bea.Pollutant(selection.Pollutant), selection.Population, bea.Year(selection.Year), bea.Domestic, epi.NasariACS)
	case "emis":
		return s.spatial.Emissions(ctx, demand, industryMask, slca.Pollutant(selection.Pollutant), bea.Year(selection.Year), bea.Domestic)
	default:
		return nil, fmt.Errorf("invalid impact type request: %s", selection.ImpactType)
	}
}

func (s *Server) impactsMap(ctx context.Context, selection *eiopb.Selection, commodityMask, industryMask *bea.Mask) (*mat.VecDense, error) {
	demand, err := s.spatial.EIO.FinalDemand(bea.FinalDemand(selection.DemandType), commodityMask, bea.Year(selection.Year), bea.Domestic)
	if err != nil {
		return nil, err
	}
	switch selection.ImpactType {
	case "health":
		return s.perArea(s.spatial.Health(ctx, demand, industryMask, bea.Pollutant(selection.Pollutant), selection.Population, bea.Year(selection.Year), bea.Domestic, epi.NasariACS))
	case "conc":
		return s.spatial.Concentrations(ctx, demand, industryMask, bea.Pollutant(selection.Pollutant), bea.Year(selection.Year), bea.Domestic)
	case "emis":
		return s.perArea(s.spatial.Emissions(ctx, demand, industryMask, slca.Pollutant(selection.Pollutant), bea.Year(selection.Year), bea.Domestic))
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
		Names:  make([]string, len(s.ioAgg.Names())+1),
		Values: make([]float32, len(s.ioAgg.Names())+1),
	}
	out.Names[0] = eiopb.All
	// impacts produced by all sectors owing to all sectors.
	impacts, err := s.impactsMenu(ctx, in, nil, nil)
	if err != nil {
		return nil, err
	}
	out.Values[0] = float32(mat.Sum(impacts))
	i := 1
	for _, g := range s.ioAgg.Names() {
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

func sccGroup(s *bea.SpatialEIO, m *bea.Mask) (codes, descriptions []string) {
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
		abbrev, err := s.ioAgg.Abbreviation(demandGroup)
		if err != nil {
			return nil, err
		}
		return s.ioAgg.CommodityMask(abbrev), nil
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
		abbrev, err := s.sccAgg.Abbreviation(productionGroup)
		if err != nil {
			return nil, err
		}
		return s.sccAgg.IndustryMask(abbrev), nil
	}
	// demand from a single sector.
	return s.spatial.SCCMask(slca.SCC(productionSector))
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
		Names:  make([]string, len(s.sccAgg.Names())+1),
		Values: make([]float32, len(s.sccAgg.Names())+1),
	}
	out.Names[0] = eiopb.All
	v, err := s.impactsMenu(ctx, in, demandMask, nil)
	if err != nil {
		return nil, err
	}
	out.Values[0] = float32(mat.Sum(v))
	i := 1
	for _, g := range s.sccAgg.Names() {
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

	out := &eiopb.Selectors{Names: []string{eiopb.All}, Codes: []string{eiopb.All}}
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
	sectors, descriptions := sccGroup(s.spatial, mask)
	out.Names = append(out.Names, descriptions...)
	out.Codes = append(out.Codes, sectors...)
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
		OverFlow: cm2,
	}
	cm.SetMin(mat.Min(impacts))
	cm.SetMax(mat.Max(impacts))
	cutpt := percentile(impacts, 0.999)
	cm.SetHighCut(cutpt)

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
	out.Legend = legend(cm, cutpt)
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
	gob.Register([]*eiopb.Rectangle{})
	gob.Register(geom.Polygon{})
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
	} else if strings.HasPrefix(cacheLoc, "gs://") {
		loc, err := url.Parse(cacheLoc)
		if err != nil {
			panic(err)
		}
		cf, err := requestcache.GoogleCloudStorage(context.TODO(), loc.Host, strings.TrimLeft(loc.Path, "/"), marshal, unmarshal)
		if err != nil {
			panic(err)
		}
		return requestcache.NewCache(f, workers, requestcache.Deduplicate(),
			requestcache.Memory(memCacheSize), cf)
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

func (s *Server) DefaultSelection(ctx context.Context, in *eiopb.Selection) (*eiopb.Selection, error) {
	return &eiopb.Selection{
		DemandGroup:      eiopb.All,
		DemandSector:     eiopb.All,
		ProductionGroup:  eiopb.All,
		ProductionSector: eiopb.All,
		ImpactType:       "conc",
		DemandType:       eiopb.All,
		Year:             int32(s.defaultYear),
		Population:       s.spatial.CSTConfig.CensusPopColumns[0],
		Pollutant:        int32(bea.TotalPM25),
	}, nil
}

func (s *Server) Populations(ctx context.Context, _ *eiopb.Selection) (*eiopb.Selectors, error) {
	return &eiopb.Selectors{Names: s.spatial.CSTConfig.CensusPopColumns}, nil
}

func (s *Server) Years(ctx context.Context, _ *eiopb.Selection) (*eiopb.Year, error) {
	o := &eiopb.Year{Years: make([]int32, len(s.spatial.EIO.Years()))}
	for i, y := range s.spatial.EIO.Years() {
		o.Years[i] = int32(y)
	}
	return o, nil
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
