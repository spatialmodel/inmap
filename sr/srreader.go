/*
Copyright © 2013 the InMAP authors.
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
along with InMAP.  If not, see <http://www.gnu.org/licenses/>.
*/

package sr

import (
	"context"
	"fmt"
	"reflect"
	"runtime"
	"strings"
	"sync"

	"github.com/Knetic/govaluate"
	"github.com/ctessum/cdf"
	"github.com/ctessum/geom"
	"github.com/ctessum/geom/proj"
	"github.com/ctessum/requestcache"
	"github.com/gonum/floats"
	"github.com/spatialmodel/inmap"
	"github.com/spatialmodel/inmap/science/chem/simplechem"
)

// Reader allows the interaction with a NetCDF-formatted source-receptor (SR) database.
type Reader struct {
	cdf.File
	d                 inmap.InMAP
	indices           map[*inmap.Cell]int
	layers            []int // layers are the vertical layers that are represented in the SR matrix.
	nCellsGroundLevel int   // number of cells in the lowest model layer

	// CacheSize specifies the number of records to be held in the memory cache.
	// Larger numbers lead to faster operation but greater memory use.
	// If the SR matrix is created from a version of InMAP with 50,000 grid cells
	// in each vertical layer, then a CacheSize of 50,000 would be the equivalent
	// of storing an entire vertical layer in the cache. The default is 100.
	// CacheSize can only be changed before the Reader has been used to read
	// concentrations for the first time.
	CacheSize int

	// sourceCache is a cache for SR records.
	sourceCache *requestcache.Cache
	// sourceInit is used to initialize sourceCache.
	sourceInit sync.Once
}

// NewReader creates a new SR reader from the netcdf database specified by r.
func NewReader(r cdf.ReaderWriterAt) (*Reader, error) {
	cf, err := cdf.Open(r)
	if err != nil {
		return nil, err
	}
	sr := &Reader{
		File:      *cf,
		CacheSize: 100,
	}
	nCells := sr.Header.Lengths("N")[0] // number of InMAP cells.
	cells := make([]*inmap.Cell, nCells)
	sr.nCellsGroundLevel = sr.Header.Lengths("PrimaryPM25")[1]

	// Get the grid cell geometry
	g := make([][]float64, 4)
	for i, dir := range []string{"N", "S", "E", "W"} {
		g[i], err = sr.readFullVar64(dir)
		if err != nil {
			return nil, err
		}
	}
	for i := range cells {
		cells[i] = new(inmap.Cell)
		cells[i].Polygonal = geom.Polygon{{
			geom.Point{X: g[3][i], Y: g[1][i]}, // W, S
			geom.Point{X: g[2][i], Y: g[1][i]}, // E, S
			geom.Point{X: g[2][i], Y: g[0][i]}, // E, N
			geom.Point{X: g[3][i], Y: g[0][i]}, // W, N
			geom.Point{X: g[3][i], Y: g[1][i]}, // W, S
		}}
	}

	// Get the included layers.
	rr := sr.File.Reader("layers", nil, nil)
	buf := rr.Zero(-1)
	if _, err = rr.Read(buf); err != nil {
		return nil, err
	}
	l := buf.([]int32)
	sr.layers = make([]int, len(l))
	for i, ll := range l {
		sr.layers[i] = int(ll)
	}

	// Get InMAP data
	varMap := make(map[string]string)
	for _, v := range sr.File.Header.Variables() {
		varMap[v] = ""
	}
	cellVarMap := make(map[string]string)
	cVal := reflect.ValueOf(cells[0]).Elem()
	cType := cVal.Type()
	for i := 0; i < cVal.NumField(); i++ {
		fieldName := cType.Field(i).Name
		if _, ok := varMap[fieldName]; ok {
			cellVarMap[fieldName] = ""
			data, err := sr.readFullVar64(fieldName)
			if err != nil {
				return nil, err
			}
			for j, c := range cells {
				field := reflect.ValueOf(c).Elem().Field(i)
				switch field.Type().Kind() {
				case reflect.Float64:
					field.SetFloat(data[j])
				case reflect.Int:
					field.SetInt(int64(data[j]))
				default:
					panic(fmt.Errorf("unsupported field type %v", field.Type().Kind()))
				}
			}
		}
	}
	var m simplechem.Mechanism
	for _, cell := range cells {
		sr.d.InsertCell(cell, m)
	}

	// Read in extra data that wasn't yet able to be saved into the cells,
	// and save it as PopData.
	sr.d.PopIndices = make(map[string]int)
	var popI int
	for _, v := range sr.File.Header.Variables() {
		if sr.File.Header.Dimensions(v)[0] != "allcells" {
			continue // We're only interested in the InMAP variables.
		}
		if _, ok := cellVarMap[v]; !ok {
			sr.d.PopIndices[v] = popI
			popI++
		}
	}
	for _, c := range cells {
		c.PopData = make([]float64, len(sr.d.PopIndices))
	}

	for v, popI := range sr.d.PopIndices {
		data, err := sr.readFullVar64(v)
		if err != nil {
			return nil, err
		}
		for i, c := range cells {
			c.PopData[popI] = data[i]
		}
	}

	// Add cell indices for easy searching later.
	sr.indices = make(map[*inmap.Cell]int)
	ii := 0
	prevLayer := 0
	for _, c := range sr.d.Cells() {
		if c.Layer != prevLayer {
			ii = 0
		}
		sr.indices[c] = ii
		ii++
		prevLayer = c.Layer
	}

	return sr, nil
}

// readFullVar reads a full float64 variable and returns it as a
// []float64.
func (sr *Reader) readFullVar64(varName string) ([]float64, error) {
	r := sr.File.Reader(varName, nil, nil)
	buf := r.Zero(-1)
	_, err := r.Read(buf)
	if err != nil {
		return nil, err
	}
	return buf.([]float64), nil
}

// Geometry returns the SR matrix grid geometry in the native grid projection.
func (sr *Reader) Geometry() []geom.Polygonal {
	return sr.d.GetGeometry(0, false)
}

// Variables returns the data for the InMAP variables named by names. Any
// changes to the returned data may also alter the underlying data.
func (sr *Reader) Variables(names ...string) (map[string][]float64, error) {
	r := make(map[string][]float64)
	var m simplechem.Mechanism
	for _, name := range names {
		n := make(map[string]string)
		n[name] = name
		if i, ok := sr.d.PopIndices[name]; ok {
			o := make([]float64, sr.nCellsGroundLevel)
			cells := sr.d.Cells()
			for j := 0; j < sr.nCellsGroundLevel; j++ {
				o[j] = cells[j].PopData[i]
			}
			r[name] = o // only return ground-level data.
		} else {
			o, err := inmap.NewOutputter("", false, n, nil, m)
			if err != nil {
				return nil, err
			}
			d, err := sr.d.Results(o)
			if err != nil {
				return nil, err
			}
			r[name] = d[name]
		}
	}
	return r, nil
}

// Concentrations holds pollutant concentration information [μg/m3]
// for the PM2.5 subspecies.
type Concentrations struct {
	PNH4, PNO3, PSO4, SOA, PrimaryPM25 []float64
}

// TotalPM25 returns total combined PM2.5 concentration [μg/m3].
func (c *Concentrations) TotalPM25() []float64 {
	o := make([]float64, len(c.PNH4))
	floats.Add(o, c.PNH4)
	floats.Add(o, c.PNO3)
	floats.Add(o, c.PSO4)
	floats.Add(o, c.SOA)
	floats.Add(o, c.PrimaryPM25)
	return o
}

// Concentrations returns the change in Total PM2.5 concentrations caused
// by the emissions specified by e, after accounting for plume rise.
// If the emission plume height is above the highest layer in the SR
// matrix, the function will allocate the emissions to the top layer
// and an error of type AboveTopErr will be returned. In some cases it
// may be appropriate to ignore errors of this type.
// As specified in the EmisRecord documentation,
// emission units should be in μg/s.
func (sr *Reader) Concentrations(emis ...*inmap.EmisRecord) (*Concentrations, error) {

	out := &Concentrations{
		PNH4:        make([]float64, sr.nCellsGroundLevel),
		PNO3:        make([]float64, sr.nCellsGroundLevel),
		PSO4:        make([]float64, sr.nCellsGroundLevel),
		SOA:         make([]float64, sr.nCellsGroundLevel),
		PrimaryPM25: make([]float64, sr.nCellsGroundLevel),
	}

	// stickyErr is used for errors that shouldn't immediately
	// cause the function to fail but should be returned with the
	// result anyway.
	var stickyErr error

	for _, e := range emis {
		cells, fractions := sr.d.CellIntersections(e.Geom)
		for i, c := range cells {
			// Figure out if this cell is the right layer.
			var plumeHeight float64
			if e.Height != 0 {
				var in bool
				var err error
				in, plumeHeight, err = c.IsPlumeIn(e.Height, e.Diam, e.Temp, e.Velocity)
				if err != nil {
					return nil, err
				}
				if !in {
					continue
				}
			} else { // ground-level emissions
				if c.Layer != 0 {
					continue
				}
			}
			frac := fractions[i]
			index := sr.indices[c]

			layers, layerfracs, err := sr.layerFracs(c, plumeHeight)
			if err != nil {
				switch err.(type) {
				case AboveTopErr:
					stickyErr = err
				default:
					return nil, err
				}
			}

			for i, layer := range layers {
				layerfrac := layerfracs[i]

				for i, emis := range []float64{e.NH3, e.NOx, e.SOx, e.VOC, e.PM25} {
					if emis != 0 {
						v, err := sr.Source(polNames[i], layer, index)
						if err != nil {
							return nil, err
						}
						switch polNames[i] {
						case "pNH4":
							floats.AddScaled(out.PNH4, emis*frac*layerfrac, v)
						case "pNO3":
							floats.AddScaled(out.PNO3, emis*frac*layerfrac, v)
						case "pSO4":
							floats.AddScaled(out.PSO4, emis*frac*layerfrac, v)
						case "SOA":
							floats.AddScaled(out.SOA, emis*frac*layerfrac, v)
						case "PrimaryPM25":
							floats.AddScaled(out.PrimaryPM25, emis*frac*layerfrac, v)
						default:
							panic(fmt.Errorf("invalid pollutant %s", polNames[i]))
						}
					}
				}
			}
		}
	}
	return out, stickyErr
}

// SetConcentrations set the `Cf` concentration field of the underlying
// InMAP data structure to the specified values. This is not
// concurrency-safe.
func (sr *Reader) SetConcentrations(c *Concentrations) error {
	conversion := map[string]float64{
		"pnh4":        simplechem.NtoNH4,
		"pso4":        simplechem.StoSO4,
		"pno3":        simplechem.NtoNO3,
		"primarypm25": 1,
		"soa":         1,
	}
	m := simplechem.Mechanism{}
	nSpec := m.Len()
	speciesIndex := make(map[string]int)
	for i, s := range m.Species() {
		sl := strings.ToLower(s)
		if _, ok := speciesIndex[sl]; ok {
			return fmt.Errorf("sr: there is more than one (case-insensitive) instance of species `%s` in this mechanism", sl)
		}
		speciesIndex[sl] = i
	}
	cVal := reflect.ValueOf(c).Elem()
	cType := cVal.Type()
	cells := sr.d.Cells()
	for i := 0; i < cVal.NumField(); i++ {
		fieldT := cType.Field(i)
		fieldV := cVal.Field(i)
		iPol, ok := speciesIndex[strings.ToLower(fieldT.Name)]
		if !ok {
			return fmt.Errorf("sr: this mechanism does not contain case-insensitive species `%s`", fieldT.Name)
		}
		conv, ok := conversion[strings.ToLower(fieldT.Name)]
		if !ok {
			panic("missing conversion for " + strings.ToLower(fieldT.Name))
		}
		for i := 0; i < fieldV.Len(); i++ {
			c := cells[i]
			if len(c.Cf) != nSpec {
				c.Cf = make([]float64, nSpec)
			}
			c.Cf[iPol] = fieldV.Index(i).Float() / conv
		}
	}
	return nil
}

// Output writes out the results specified by variables.
// See the documenation for inmap.Outputter for more information.
// This function assumes that concentrations have already been set using
// SetConcentrations.
// Note that because the SR matrix does not save gas-phase concentrations,
// attempts to output gas-phase equations will result in all zeros.
func (sr *Reader) Output(shapefilePath string, variables map[string]string, funcs map[string]govaluate.ExpressionFunction, sRef *proj.SR) error {
	m := simplechem.Mechanism{}
	o, err := inmap.NewOutputter(shapefilePath, false, variables, funcs, m)
	if err != nil {
		return err
	}
	if err := o.CheckOutputVars(m)(&sr.d); err != nil {
		return err
	}
	if err := o.Output(sRef)(&sr.d); err != nil {
		return err
	}
	return nil
}

// polNames lists the pollutant names.
var polNames = []string{"pNH4", "pNO3", "pSO4", "SOA", "PrimaryPM25"}

// layerFracs interpolates the height of c among the layers in the
// SR matrix and returns a list of layers that should be used to represent
// the emissions in c and the weighting fraction of each layer.
func (sr *Reader) layerFracs(c *inmap.Cell, plumeHeight float64) ([]int, []float64, error) {
	var m simplechem.Mechanism
	layerHeights, _, err := sr.d.VerticalProfile("WindSpeed", c.Centroid(), m)
	if err != nil {
		return nil, nil, err
	}

	for i := 0; i < len(sr.layers); i++ {
		if c.Layer == sr.layers[i] {
			return []int{i}, []float64{1.}, nil
		}
	}

	for i := 0; i < len(sr.layers)-1; i++ {
		if sr.layers[i] < c.Layer && sr.layers[i+1] > c.Layer {
			below := layerHeights[sr.layers[i]]
			above := layerHeights[sr.layers[i+1]]
			frac := (plumeHeight - below) / (above - below)
			return []int{i, i + 1}, []float64{frac, 1 - frac}, nil
		}
	}

	if c.Layer > sr.layers[len(sr.layers)-1] {
		return []int{len(sr.layers) - 1}, []float64{1.}, AboveTopErr{PlumeHeight: plumeHeight}
	}
	panic("problem in layerFracs")
}

// AboveTopErr is returned when the plume height of an emissions
// source is above the top layer in the SR matrix.
type AboveTopErr struct {
	PlumeHeight float64
}

func (e AboveTopErr) Error() string {
	return fmt.Sprintf("plume height (%g m) is above the top layer in the SR matrix", e.PlumeHeight)
}

// Source returns concentrations in μg m-3 for emissions in μg s-1 of
// pollutant pol in SR layer index 'layer' and horizontal grid cell index
// 'index'. This function uses a cache with the size specified by
// the CacheSize attribute of the receiver to speed up repeated requests
// and is concurrency-safe. Users desiring to make changes to the returned
// values should make a copy first to avoid inadvertently editing the cached results
// which could cause subsequent results from this function to be incorrect.
// If the layer and index are not known, use the Concentrations method instead.
func (sr *Reader) Source(pol string, layer, index int) ([]float64, error) {
	sr.sourceInit.Do(func() {
		sr.sourceCache = requestcache.NewCache(func(ctx context.Context, request interface{}) (interface{}, error) {
			r := request.(sourceRequest)
			return sr.source(r.pol, r.layer, r.index)
		}, runtime.GOMAXPROCS(-1),
			requestcache.Deduplicate(), requestcache.Memory(sr.CacheSize))
	})
	req := sr.sourceCache.NewRequest(context.TODO(),
		sourceRequest{pol: pol, layer: layer, index: index},
		fmt.Sprintf("%s_%d_%d", pol, layer, index),
	)
	result, err := req.Result()
	return result.([]float64), err
}

type sourceRequest struct {
	pol          string
	layer, index int
}

// source returns concentrations in μg m-3 for emissions in μg s-1 of
// pollutant pol in SR layer index 'layer' and horizontal grid cell index
// 'index'.
func (sr *Reader) source(pol string, layer, index int) ([]float64, error) {
	if layer >= len(sr.layers) {
		return nil, fmt.Errorf("sr: requested layer %d >= number of layers (%d)", layer, len(sr.layers))
	}
	if index >= sr.nCellsGroundLevel {
		return nil, fmt.Errorf("sr: requested index %d >= number of grid cells (%d)", index, sr.nCellsGroundLevel)
	}
	foundPol := false
	for _, p := range polNames {
		if p == pol {
			foundPol = true
			break
		}
	}
	if !foundPol {
		return nil, fmt.Errorf("sr: requested pollutant %s not one of valid pollutants (%+v)", pol, polNames)
	}
	start := []int{layer, index, 0}
	end := []int{layer, index, sr.nCellsGroundLevel - 1}
	return sr.get(pol, start, end)
}

// get returns data from a variable starting and ending at the given indices.
func (sr *Reader) get(pol string, start, end []int) ([]float64, error) {
	// indices: layer, source, receptor.
	r := sr.File.Reader(pol, start, end)
	buf := r.Zero(-1)
	_, err := r.Read(buf)
	if err != nil {
		return nil, err
	}
	dat32 := buf.([]float32)
	dat64 := make([]float64, len(dat32))
	for i, v := range dat32 {
		dat64[i] = float64(v)
	}
	return dat64, err
}
