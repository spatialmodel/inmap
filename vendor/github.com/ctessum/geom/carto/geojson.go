package carto

import (
	"encoding/json"
	"github.com/ctessum/geom"
	"github.com/ctessum/geom/encoding/geojson"
	"io"
)

type GeoJSONfeature struct {
	Type       string             `json:"type"`
	Geometry   *geojson.Geometry  `json:"geometry"`
	Properties map[string]float64 `json:"properties"`
}
type GeoJSON struct {
	Type     string `json:"type"`
	CRS      Crs    `json:"crs"`
	Features []*GeoJSONfeature
}

// Coordinate reference system. Used for GeoJSON
type Crs struct {
	Type       string   `json:"type"`
	Properties CrsProps `json:"properties"`
}

// Coordinate reference system properties.
type CrsProps struct {
	Name string `json:"name"`
}

func LoadGeoJSON(r io.Reader) (*GeoJSON, error) {
	out := new(GeoJSON)
	d := json.NewDecoder(r)
	err := d.Decode(&out)
	return out, err
}

func (g *GeoJSON) Sum(propertyName string) float64 {
	sum := 0.
	for _, f := range g.Features {
		sum += f.Properties[propertyName]
	}
	return sum
}

func (g *GeoJSON) GetProperty(propertyName string) []float64 {
	out := make([]float64, len(g.Features))
	for i, f := range g.Features {
		out[i] = f.Properties[propertyName]
	}
	return out
}

func (g *GeoJSON) GetGeometry() ([]geom.Geom, error) {
	var err error
	out := make([]geom.Geom, len(g.Features))
	for i, f := range g.Features {
		out[i], err = geojson.FromGeoJSON(f.Geometry)
		if err != nil {
			return nil, err
		}
	}
	return out, nil
}

// Convert map data to GeoJSON, where value name is a
// name for the data values being output.
func (m *MapData) ToGeoJSON(valueName string) (*GeoJSON, error) {
	var err error
	g := new(GeoJSON)
	g.Type = "FeatureCollection"
	g.CRS = Crs{"name", CrsProps{"EPSG:3857"}}
	g.Features = make([]*GeoJSONfeature, len(m.Shapes))
	for i, shape := range m.Shapes {
		f := new(GeoJSONfeature)
		f.Type = "Feature"
		f.Geometry, err = geojson.ToGeoJSON(shape)
		if err != nil {
			return nil, err
		}
		f.Properties = map[string]float64{valueName: m.Data[i]}
		g.Features[i] = f
	}
	return g, nil
}
