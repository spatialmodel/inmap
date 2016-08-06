package geom

import "github.com/ctessum/geom/proj"

// Transform shifts the coordinates of p according to t.
func (p Point) Transform(t proj.Transformer) (Geom, error) {
	var err error
	p2 := Point{}
	p2.X, p2.Y, err = t(p.X, p.Y)
	return p2, err
}

// Transform shifts the coordinates of mp according to t.
func (mp MultiPoint) Transform(t proj.Transformer) (Geom, error) {
	mp2 := make(MultiPoint, len(mp))
	for i, p := range mp {
		g, err := p.Transform(t)
		if err != nil {
			return nil, err
		}
		mp2[i] = g.(Point)
	}
	return mp2, nil
}

// Transform shifts the coordinates of l according to t.
func (l LineString) Transform(t proj.Transformer) (Geom, error) {
	l2 := make(LineString, len(l))
	var err error
	for i, p := range l {
		p2 := Point{}
		p2.X, p2.Y, err = t(p.X, p.Y)
		if err != nil {
			return nil, err
		}
		l2[i] = p
	}
	return l2, nil
}

// Transform shifts the coordinates of ml according to t.
func (ml MultiLineString) Transform(t proj.Transformer) (Geom, error) {
	ml2 := make(MultiLineString, len(ml))
	for i, l := range ml {
		g, err := l.Transform(t)
		ml2[i] = g.(LineString)
		if err != nil {
			return nil, err
		}
	}
	return ml2, nil
}

// Transform shifts the coordinates of p according to t.
func (p Polygon) Transform(t proj.Transformer) (Geom, error) {
	p2 := make(Polygon, len(p))
	var err error
	for i, r := range p {
		p2[i] = make([]Point, len(r))
		for j, pp := range r {
			pp2 := Point{}
			pp2.X, pp2.Y, err = t(pp.X, pp.Y)
			if err != nil {
				return nil, err
			}
			p2[i][j] = pp2
		}
	}
	return p2, nil
}

// Transform shifts the coordinates of mp according to t.
func (mp MultiPolygon) Transform(t proj.Transformer) (Geom, error) {
	mp2 := make(MultiPolygon, len(mp))
	for i, p := range mp {
		g, err := p.Transform(t)
		mp2[i] = g.(Polygon)
		if err != nil {
			return nil, err
		}
	}
	return mp2, nil
}

// Transform shifts the coordinates of gc according to t.
func (gc GeometryCollection) Transform(t proj.Transformer) (Geom, error) {
	gc2 := make(GeometryCollection, len(gc))
	var err error
	for i, g := range gc {
		gc2[i], err = g.Transform(t)
		if err != nil {
			return nil, err
		}
	}
	return gc2, nil
}

// Transform shifts the coordinates of b according to t.
func (b *Bounds) Transform(t proj.Transformer) (Geom, error) {
	b2 := &Bounds{}
	g, err := b.Max.Transform(t)
	if err != nil {
		return nil, err
	}
	b2.Max = g.(Point)
	g, err = b.Min.Transform(t)
	if err != nil {
		return nil, err
	}
	b2.Min = g.(Point)
	return b2, nil
}
