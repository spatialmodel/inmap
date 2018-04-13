//+build !js

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

package main

// Build the client javascript code.
//go:generate gopherjs build

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image/color"
	"log"
	"net/http"
	"net/rpc"
	"net/rpc/jsonrpc"
	"os"
	"runtime"
	"sort"
	"sync"

	"github.com/ctessum/geom"
	"github.com/ctessum/geom/carto"
	"github.com/ctessum/geom/encoding/geojson"
	"github.com/ctessum/geom/proj"
	"github.com/ctessum/requestcache"
	"github.com/spatialmodel/epi"
	"gonum.org/v1/gonum/mat"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/palette"
	"gonum.org/v1/plot/palette/moreland"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
	"gonum.org/v1/plot/vg/draw"
	"gonum.org/v1/plot/vg/vgimg"

	"golang.org/x/net/websocket"

	"github.com/spatialmodel/inmap/emissions/slca"
	"github.com/spatialmodel/inmap/emissions/slca/bea"
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

	geomCache, areaCache         *requestcache.Cache
	geomCacheOnce, areaCacheOnce sync.Once
}

func main() {
	log.Println("Starting up...")
	f, err := os.Open(os.ExpandEnv("${GOPATH}/src/bitbucket.org/ctessum/slca/bea/data/example_config.toml"))
	if err != nil {
		log.Fatal(err)
	}

	s, err := bea.NewSpatial(f)
	if err != nil {
		log.Fatal(err)
	}
	f.Close()

	a, err := s.EIO.NewAggregator(os.ExpandEnv("${GOPATH}/src/bitbucket.org/ctessum/slca/bea/data/aggregates.xlsx"))
	if err != nil {
		log.Fatal(err)
	}

	model := &Server{agg: a, spatial: s}

	rpc.Register(model)

	http.Handle("/ws-rpc", websocket.Handler(func(conn *websocket.Conn) {
		jsonrpc.ServeConn(conn)
	}))

	// Serve the static files.
	http.Handle("/", http.FileServer(http.Dir(".")))

	log.Println("Ready!")
	panic(http.ListenAndServe(port, nil))
}

func (s *Server) impactsMenu(ctx context.Context, selection *Selection, commodityMask, industryMask *bea.Mask) (*mat.VecDense, error) {
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

func (s *Server) impactsMap(ctx context.Context, selection *Selection, commodityMask, industryMask *bea.Mask) (*mat.VecDense, error) {
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
func (s *Server) DemandGroups(in *Selection, out *Selectors) error {
	ctx := context.Background()
	out.Names = make([]string, len(s.agg.Names())+1)
	out.Values = make([]float64, len(s.agg.Names())+1)
	out.Names[0] = All
	// impacts produced by all sectors owing to all sectors.
	impacts, err := s.impactsMenu(ctx, in, nil, nil)
	if err != nil {
		return err
	}
	out.Values[0] = mat.Sum(impacts)
	i := 1
	for _, g := range s.agg.Names() {
		out.Names[i] = g

		// impacts produced by all sectors owing to consumption in this group
		// of sectors.
		mask, err := s.demandMask(g, All)
		if err != nil {
			return err
		}
		impacts, err := s.impactsMenu(ctx, in, mask, nil)
		if err != nil {
			return err
		}
		out.Values[i] = mat.Sum(impacts)
		i++
	}
	sort.Sort(out)
	return nil
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
func (s *Server) DemandSectors(in *Selection, out *Selectors) error {
	ctx := context.Background()
	out.Names = []string{All}
	if in.DemandGroup == All {
		// impacts produced by all sectors owing to all sectors.
		impacts, err := s.impactsMenu(ctx, in, nil, nil)
		if err != nil {
			return err
		}
		out.Values = []float64{mat.Sum(impacts)}
		return nil
	}
	mask, err := s.demandMask(in.DemandGroup, All)
	if err != nil {
		return err
	}
	impacts, err := s.impactsMenu(ctx, in, mask, nil)
	if err != nil {
		return err
	}
	out.Values = []float64{mat.Sum(impacts)}

	sectors := commodityGroup(&s.spatial.EIO, mask)
	out.Names = append(out.Names, sectors...)
	temp := make([]float64, len(sectors))
	out.Values = append(out.Values, temp...)
	for i, sector := range sectors {
		// impacts produced by all sectors owing to consumption in this sector.
		mask, err := s.demandMask(in.DemandGroup, sector)
		if err != nil {
			return err
		}
		impacts, err := s.impactsMenu(ctx, in, mask, nil)
		if err != nil {
			return err
		}
		out.Values[i+1] = mat.Sum(impacts)
	}
	sort.Sort(out)
	return nil
}

// demandMask returns a commodity mask corresponding to the
// given selection.
func (s *Server) demandMask(demandGroup, demandSector string) (*bea.Mask, error) {
	if demandGroup == All {
		return nil, nil
	} else if demandSector == All {
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
	if productionGroup == All {
		return nil, nil
	} else if productionSector == All {
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
func (s *Server) ProdGroups(in *Selection, out *Selectors) error {
	ctx := context.Background()
	demandMask, err := s.demandMask(in.DemandGroup, in.DemandSector)
	if err != nil {
		return err
	}
	out.Names = make([]string, len(s.agg.Names())+1)
	out.Values = make([]float64, len(s.agg.Names())+1)
	out.Names[0] = All
	v, err := s.impactsMenu(ctx, in, demandMask, nil)
	if err != nil {
		return err
	}
	out.Values[0] = mat.Sum(v)
	i := 1
	for _, g := range s.agg.Names() {
		out.Names[i] = g
		mask, err := s.productionMask(g, All)
		if err != nil {
			return err
		}
		v, err := s.impactsMenu(ctx, in, demandMask, mask)
		if err != nil {
			return err
		}
		out.Values[i] = mat.Sum(v)
		i++
	}
	sort.Sort(out)
	return nil
}

// ProdSectors returns the available production sectors.
func (s *Server) ProdSectors(in *Selection, out *Selectors) error {
	ctx := context.Background()
	demandMask, err := s.demandMask(in.DemandGroup, in.DemandSector)
	if err != nil {
		return err
	}

	out.Names = []string{All}
	if in.ProductionGroup == All {
		v, err2 := s.impactsMenu(ctx, in, demandMask, nil)
		if err2 != nil {
			return err2
		}
		out.Values = []float64{mat.Sum(v)}
		return nil
	}
	mask, err := s.productionMask(in.ProductionGroup, All)
	if err != nil {
		return err
	}
	v, err := s.impactsMenu(ctx, in, demandMask, mask)
	if err != nil {
		return err
	}
	out.Values = []float64{mat.Sum(v)}
	sectors := industryGroup(&s.spatial.EIO, mask)
	out.Names = append(out.Names, sectors...)
	temp := make([]float64, len(sectors))
	out.Values = append(out.Values, temp...)
	for i, sector := range sectors {
		mask, err := s.productionMask(in.ProductionGroup, sector)
		if err != nil {
			return err
		}
		v, err := s.impactsMenu(ctx, in, demandMask, mask)
		if err != nil {
			return err
		}
		out.Values[i+1] = mat.Sum(v)
	}
	sort.Sort(out)
	return nil
}

// MapInfo returns the grid cell colors and a legend for the given selection.
func (s *Server) MapInfo(in *Selection, out *MapInfo) error {
	ctx := context.Background()

	commodityMask, err := s.demandMask(in.DemandGroup, in.DemandSector)
	if err != nil {
		return err
	}
	industryMask, err := s.productionMask(in.ProductionGroup, in.ProductionSector)
	if err != nil {
		return err
	}
	impacts, err := s.impactsMap(ctx, in, commodityMask, industryMask)
	if err != nil {
		return err
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
	(*out).Color = make([]RGB, rows)
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
		(*out).Color[i] = RGB{R: float64(col.R) / 255, G: float64(col.G) / 255, B: float64(col.B) / 255}
	}
	out.Legend = legend(cm, cm2, cutpt, max)
	return nil
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

func (s *Server) getGeometry(ctx context.Context, requestPayload interface{}) (resultPayload interface{}, err error) {
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
	o := new(carto.GeoJSON)
	o.Type = "FeatureCollection"
	o.Features = make([]*carto.GeoJSONfeature, len(g))
	var ggg geom.Geom
	for i, gg := range g {
		ggg, err = gg.Transform(ct)
		if err != nil {
			return nil, err
		}
		var geojsonGeom *geojson.Geometry
		geojsonGeom, err = geojson.ToGeoJSON(ggg)
		if err != nil {
			return nil, err
		}
		o.Features[i] = &carto.GeoJSONfeature{
			Type:     "Feature",
			Geometry: geojsonGeom,
		}
	}
	b, err := json.Marshal(o)
	if err != nil {
		return nil, err
	}
	return b, nil
}

// Geometry returns the InMAP grid geometry in the Google mercator projection.
func (s *Server) Geometry(in *Empty, out *[]byte) error {
	s.geomCacheOnce.Do(func() {
		if s.spatial.SpatialCache == "" {
			s.geomCache = requestcache.NewCache(s.getGeometry, runtime.GOMAXPROCS(-1),
				requestcache.Deduplicate(), requestcache.Memory(1))
		} else {
			s.geomCache = requestcache.NewCache(s.getGeometry, runtime.GOMAXPROCS(-1),
				requestcache.Deduplicate(), requestcache.Memory(1),
				requestcache.Disk(s.spatial.SpatialCache,
					func(i interface{}) ([]byte, error) {
						i2 := i.(*interface{})
						return (*i2).([]byte), nil
					},
					func(b []byte) (interface{}, error) { return b, nil },
				),
			)
		}
	})
	req := s.geomCache.NewRequest(context.Background(), in, "geometry")
	iface, err := req.Result()
	if err != nil {
		return err
	}
	(*out) = iface.([]byte)
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
		if s.spatial.SpatialCache == "" {
			s.areaCache = requestcache.NewCache(f, runtime.GOMAXPROCS(-1),
				requestcache.Deduplicate(), requestcache.Memory(1))
		} else {
			s.areaCache = requestcache.NewCache(f, runtime.GOMAXPROCS(-1),
				requestcache.Deduplicate(), requestcache.Memory(1),
				requestcache.Disk(s.spatial.SpatialCache, vectorMarshal, vectorUnmarshal),
			)
		}
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
