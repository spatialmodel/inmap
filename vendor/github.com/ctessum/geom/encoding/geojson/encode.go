package geojson

import (
	"encoding/json"
	"github.com/ctessum/geom"
	"reflect"
)

func pointCoordinates(point geom.Point) []float64 {
	return []float64{point.X, point.Y}
}

func pointsCoordinates(points []geom.Point) [][]float64 {
	coordinates := make([][]float64, len(points))
	for i, point := range points {
		coordinates[i] = pointCoordinates(point)
	}
	return coordinates
}

func pointssCoordinates(pointss [][]geom.Point) [][][]float64 {
	coordinates := make([][][]float64, len(pointss))
	for i, points := range pointss {
		coordinates[i] = pointsCoordinates(points)
	}
	return coordinates
}

func ToGeoJSON(g geom.Geom) (*Geometry, error) {
	switch g.(type) {
	case geom.Point:
		return &Geometry{
			Type:        "Point",
			Coordinates: pointCoordinates(g.(geom.Point)),
		}, nil
	case geom.LineString:
		return &Geometry{
			Type:        "LineString",
			Coordinates: pointsCoordinates(g.(geom.LineString)),
		}, nil
	case geom.Polygon:
		return &Geometry{
			Type:        "Polygon",
			Coordinates: pointssCoordinates(g.(geom.Polygon)),
		}, nil
	default:
		return nil, &UnsupportedGeometryError{reflect.TypeOf(g).String()}
	}
}

func Encode(g geom.Geom) ([]byte, error) {
	if object, err := ToGeoJSON(g); err == nil {
		return json.Marshal(object)
	} else {
		return nil, err
	}
}
