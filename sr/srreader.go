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
	"fmt"
	"reflect"

	"bitbucket.org/ctessum/cdf"
	"github.com/ctessum/geom"
	"github.com/gonum/floats"
	"github.com/spatialmodel/inmap"
)

// Reader allows the interaction with a NetCDF-formatted source-receptor (SR) database.
type Reader struct {
	cdf.File
	d                 inmap.InMAP
	indices           map[*inmap.Cell]int
	layers            []int // layers are the vertical layers that are represented in the SR matrix.
	extraData         map[string][]float64
	nCellsGroundLevel int // number of cells in the lowest model layer
}

// NewReader creates a new SR reader from the netcdf database specified by r.
func NewReader(r cdf.ReaderWriterAt) (*Reader, error) {
	cf, err := cdf.Open(r)
	if err != nil {
		return nil, err
	}
	sr := &Reader{
		File: *cf,
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
	for _, cell := range cells {
		sr.d.InsertCell(cell)
	}

	// Read in extra data that wasn't able to be saved into the cells.
	sr.extraData = make(map[string][]float64)
	for _, v := range sr.File.Header.Variables() {
		if sr.File.Header.Dimensions(v)[0] != "allcells" {
			continue // We're only interested in the InMAP variables.
		}
		if _, ok := cellVarMap[v]; !ok {
			var err error
			sr.extraData[v], err = sr.readFullVar64(v)
			if err != nil {
				return nil, err
			}
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
	o := make(map[string][]float64)
	for _, name := range names {
		if d, ok := sr.extraData[name]; ok {
			o[name] = d[0:sr.nCellsGroundLevel] // only return ground-level data.
		} else {
			d, err := sr.d.Results(false, name)
			if err != nil {
				return nil, err
			}
			o[name] = d[name]
		}
	}
	return o, nil
}

// Concentrations returns the change in Total PM2.5 concentrations caused
// by the emissions specified by e, after accounting for plume rise.
//  As specified in the EmisRecord documentation
// emission units should be in μg/s.
func (sr *Reader) Concentrations(e *inmap.EmisRecord) ([]float64, error) {

	out := make([]float64, sr.nCellsGroundLevel)

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
			return nil, err
		}

		for i, layer := range layers {
			layerfrac := layerfracs[i]

			for i, emis := range []float64{e.NH3, e.NOx, e.SOx, e.VOC, e.PM25} {
				if emis != 0 {
					v, err := sr.Source(polNames[i], layer, index)
					if err != nil {
						return nil, err
					}
					floats.AddScaled(out, emis*frac*layerfrac, v)
				}
			}
		}
	}
	return out, nil
}

// polNames lists the pollutant names.
var polNames = []string{"pNH4", "pNO3", "pSO4", "SOA", "PrimaryPM25"}

// layerFracs interpolates the height of c among the layers in the
// SR matrix and returns a list of layers that should be used to represent
// the emissions in c and the weighting fraction of each layer.
func (sr *Reader) layerFracs(c *inmap.Cell, plumeHeight float64) ([]int, []float64, error) {

	layerHeights, _, err := sr.d.VerticalProfile("WindSpeed", c.Centroid())
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
		return nil, nil, fmt.Errorf("plume height (%g m) is above the top layer in the SR matrix", plumeHeight)
	}
	panic("problem in layerFracs")
}

// Source returns concentrations in μg m-3 for emissions in μg s-1 of
// pollutant pol in SR layer index 'layer' and horizontal grid cell index
// 'index'. If the layer and index are not known, use Concentrations instead.
func (sr *Reader) Source(pol string, layer, index int) ([]float64, error) {
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
