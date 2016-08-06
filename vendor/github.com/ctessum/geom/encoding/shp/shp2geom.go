package shp

import (
	"fmt"
	"math"
	"reflect"

	"github.com/ctessum/geom"
	"github.com/ctessum/geom/op"
	"github.com/jonas-p/go-shp"
)

// FixOrientation specifies whether to automatically check and fix the
// orientation of polygons imported from shapefiles.
var FixOrientation = false

// Shp2Geom converts a shapefile shape to a geometry
// object that can be used with other packages.
// This function can be used to wrap the go-shp "Shape()" method.
func shp2Geom(n int, s shp.Shape) (int, geom.Geom, error) {
	switch t := reflect.TypeOf(s); {
	case t == reflect.TypeOf(&shp.Point{}):
		return n, point2geom(*s.(*shp.Point)), nil
	case t == reflect.TypeOf(&shp.PointM{}):
		return n, pointM2geom(*s.(*shp.PointM)), nil
	case t == reflect.TypeOf(&shp.PointZ{}):
		return n, pointZ2geom(*s.(*shp.PointZ)), nil
	case t == reflect.TypeOf(&shp.Polygon{}):
		return n, polygon2geom(*s.(*shp.Polygon)), nil
	case t == reflect.TypeOf(&shp.PolygonM{}):
		return n, polygonM2geom(*s.(*shp.PolygonM)), nil
	case t == reflect.TypeOf(&shp.PolygonZ{}):
		return n, polygonZ2geom(*s.(*shp.PolygonZ)), nil
	case t == reflect.TypeOf(&shp.PolyLine{}):
		return n, polyLine2geom(*s.(*shp.PolyLine)), nil
	case t == reflect.TypeOf(&shp.PolyLineM{}):
		return n, polyLineM2geom(*s.(*shp.PolyLineM)), nil
	case t == reflect.TypeOf(&shp.PolyLineZ{}):
		return n, polyLineZ2geom(*s.(*shp.PolyLineZ)), nil
	//case t == "MultiPatch": // not yet supported
	case t == reflect.TypeOf(&shp.MultiPoint{}):
		return n, multiPoint2geom(*s.(*shp.MultiPoint)), nil
	case t == reflect.TypeOf(&shp.MultiPointM{}):
		return n, multiPointM2geom(*s.(*shp.MultiPointM)), nil
	case t == reflect.TypeOf(&shp.MultiPointZ{}):
		return n, multiPointZ2geom(*s.(*shp.MultiPointZ)), nil
	case t == reflect.TypeOf(&shp.Null{}):
		return n, nil, nil
	default:
		return n, nil, fmt.Errorf("Unsupported shape type: %v", t)
	}
}

// Functions for converting shp to geom

func point2geom(s shp.Point) geom.Geom {
	return geom.Point(s)
}
func pointM2geom(s shp.PointM) geom.Geom {
	return geom.Point{s.X, s.Y}
}
func pointZ2geom(s shp.PointZ) geom.Geom {
	return geom.Point{s.X, s.Y}
}
func getStartEnd(parts []int32, points []shp.Point, i int) (start, end int) {
	start = int(parts[i])
	if i == len(parts)-1 {
		end = len(points)
	} else {
		end = int(parts[i+1])
	}
	return
}
func polygon2geom(s shp.Polygon) geom.Geom {
	var pg geom.Polygon = make([][]geom.Point, len(s.Parts))
	for i := 0; i < len(s.Parts); i++ {
		start, end := getStartEnd(s.Parts, s.Points, i)
		pg[i] = make([]geom.Point, end-start)
		// Go backwards around the rings to switch to OGC format
		for j := end - 1; j >= start; j-- {
			pg[i][j-start] = geom.Point(s.Points[j])
		}
	}
	// Make sure the winding direction is correct
	if FixOrientation {
		op.FixOrientation(pg)
	}
	return pg
}
func polygonM2geom(s shp.PolygonM) geom.Geom {
	var pg geom.Polygon = make([][]geom.Point, len(s.Parts))
	jj := 0
	for i := 0; i < len(s.Parts); i++ {
		start, end := getStartEnd(s.Parts, s.Points, i)
		jj += end - start
		pg[i] = make([]geom.Point, end-start)
		// Go backwards around the rings to switch to OGC format
		for j := end - 1; j >= start; j-- {
			ss := s.Points[j]
			pg[i][j-start] = geom.Point{ss.X, ss.Y} //, s.MArray[jj]}
			jj--
		}
	}
	// Make sure the winding direction is correct
	op.FixOrientation(pg)
	return pg
}

func polygonZ2geom(s shp.PolygonZ) geom.Geom {
	var pg geom.Polygon = make([][]geom.Point, len(s.Parts))
	jj := -1
	for i := 0; i < len(s.Parts); i++ {
		start, end := getStartEnd(s.Parts, s.Points, i)
		jj += end - start
		pg[i] = make([]geom.Point, end-start)
		// Go backwards around the rings to switch to OGC format
		for j := end - 1; j >= start; j-- {
			ss := s.Points[j]
			pg[i][j-start] = geom.Point{ss.X, ss.Y} //, s.ZArray[jj], s.MArray[jj]}
			jj--
		}
	}
	// Make sure the winding direction is correct
	op.FixOrientation(pg)
	return pg
}
func polyLine2geom(s shp.PolyLine) geom.Geom {
	var pl geom.MultiLineString = make([]geom.LineString, len(s.Parts))
	for i := 0; i < len(s.Parts); i++ {
		start, end := getStartEnd(s.Parts, s.Points, i)
		pl[i] = make([]geom.Point, end-start)
		for j := start; j < end; j++ {
			pl[i][j-start] = geom.Point(s.Points[j])
		}
	}
	return pl
}
func polyLineM2geom(s shp.PolyLineM) geom.Geom {
	var pl geom.MultiLineString = make([]geom.LineString, len(s.Parts))
	jj := 0
	for i := 0; i < len(s.Parts); i++ {
		start, end := getStartEnd(s.Parts, s.Points, i)
		pl[i] = make([]geom.Point, end-start)
		for j := start; j < end; j++ {
			ss := s.Points[j]
			pl[i][j-start] =
				geom.Point{ss.X, ss.Y} //, s.MArray[jj]}
			jj++
		}
	}
	return pl
}
func polyLineZ2geom(s shp.PolyLineZ) geom.Geom {
	var pl geom.MultiLineString = make([]geom.LineString, len(s.Parts))
	jj := 0
	for i := 0; i < len(s.Parts); i++ {
		start, end := getStartEnd(s.Parts, s.Points, i)
		pl[i] = make([]geom.Point, end-start)
		for j := start; j < end; j++ {
			ss := s.Points[j]
			pl[i][j-start] =
				geom.Point{ss.X, ss.Y} //, s.ZArray[jj], s.MArray[jj]}
			jj++
		}
	}
	return pl
}
func multiPoint2geom(s shp.MultiPoint) geom.Geom {
	var mp geom.MultiPoint = make([]geom.Point, len(s.Points))
	for i, p := range s.Points {
		mp[i] = geom.Point(p)
	}
	return mp
}
func multiPointM2geom(s shp.MultiPointM) geom.Geom {
	var mp geom.MultiPoint = make([]geom.Point, len(s.Points))
	for i, p := range s.Points {
		mp[i] = geom.Point{p.X, p.Y} //, s.MArray[i]}
	}
	return mp
}
func multiPointZ2geom(s shp.MultiPointZ) geom.Geom {
	var mp geom.MultiPoint = make([]geom.Point, len(s.Points))
	for i, p := range s.Points {
		mp[i] = geom.Point{p.X, p.Y} //, s.ZArray[i], s.MArray[i]}
	}
	return mp
}

// Geom2Shp converts a geometry object to a shapefile shape.
func geom2Shp(g geom.Geom) (shp.Shape, error) {
	if g == nil {
		return &shp.Null{}, nil
	}
	switch t := g.(type) {
	case geom.Point:
		return geom2point(g.(geom.Point)), nil
	case geom.Polygon:
		return geom2polygon(g.(geom.Polygon)), nil
	case geom.MultiLineString:
		return geom2polyLine(g.(geom.MultiLineString)), nil
	//case t == "MultiPatch": // not yet supported
	case geom.MultiPoint:
		return geom2multiPoint(g.(geom.MultiPoint)), nil
	default:
		return nil, fmt.Errorf("Unsupported geom type: %v", t)
	}
}

// Functions for converting geom to shp

func geom2point(g geom.Point) shp.Shape {
	p := shp.Point(g)
	return &p
}
func geom2polygon(g geom.Polygon) shp.Shape {
	parts := make([][]shp.Point, len(g))
	for i, r := range g {
		parts[i] = make([]shp.Point, len(r))
		// switch the winding direction
		for j := len(r) - 1; j >= 0; j-- {
			parts[i][j] = shp.Point(r[j])
		}
	}
	p := shp.Polygon(*shp.NewPolyLine(parts))
	return &p
}
func valrange(a []float64) [2]float64 {
	out := [2]float64{math.Inf(1), math.Inf(-1)}
	for _, val := range a {
		if val < out[0] {
			out[0] = val
		}
		if val > out[1] {
			out[1] = val
		}
	}
	return out
}
func geom2polyLine(g geom.MultiLineString) shp.Shape {
	parts := make([][]shp.Point, len(g))
	for i, r := range g {
		parts[i] = make([]shp.Point, len(r))
		for j, l := range r {
			parts[i][j] = shp.Point(l)
		}
	}
	return shp.NewPolyLine(parts)
}
func geom2multiPoint(g geom.MultiPoint) shp.Shape {
	mp := new(shp.MultiPoint)
	mp.Box = bounds2box(g)
	mp.NumPoints = int32(len(g))
	mp.Points = make([]shp.Point, len(g))
	for i, p := range g {
		mp.Points[i] = shp.Point(p)
	}
	return mp
}
func bounds2box(g geom.Geom) shp.Box {
	b := g.Bounds()
	return shp.Box{b.Min.X, b.Min.Y, b.Max.X, b.Max.Y}
}
