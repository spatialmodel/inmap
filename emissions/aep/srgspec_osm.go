/*
Copyright (C) 2019 the InMAP authors.
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

package aep

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sync"

	backoff "github.com/cenkalti/backoff/v4"
	"github.com/ctessum/geom"
	"github.com/ctessum/geom/encoding/osm"
	"github.com/ctessum/geom/encoding/wkb"
	"github.com/ctessum/geom/index/rtree"
	"github.com/ctessum/geom/proj"
	"github.com/jackc/pgx/v4/pgxpool"
)

// SrgSpecOSM holds OpenStreetMap spatial surrogate specification information.
type SrgSpecOSM struct {
	Region Country `json:"region"`
	Name   string  `json:"name"`
	Code   string  `json:"code"`

	// The name of the PostGIS table that contains the surrogate data.
	// The default osm2pgsql table names are: planet_osm_line,
	// planet_osm_roads, planet_osm_polygon, and planet_osm_point.
	OSMTable string `json:"osm_table"`

	Tags map[string][]string `json:"tags"`

	// BackupSurrogateNames specifies names of surrogates to use if this
	// one doesn't have data for the desired location.
	BackupSurrogateNames []string `json:"backup_surrogate_names"`

	// MergeNames specify names of other surrogates that should be combined
	// to create this surrogate.
	MergeNames []string `json:"merge_names"`
	// MergeMultipliers specifies multipliers associated with the surrogates
	// in MergeNames.
	MergeMultipliers []float64 `json:"merge_multipliers"`

	connectPostGISOnce sync.Once
	postGISURL string
	conn *pgxpool.Pool
}

// ReadSrgSpec reads a OpenStreetMap surrogate specification formated as a
// JSON array of SrgSpecOSM objects.
// postGISURL specifies the URL to use to connect to a PostGIS database
// with the OpenStreetMap data loaded. The URL should be in the format:
// postgres://username:password@hostname:port/databasename".
//
// The OpenStreetMap data can be loaded into the database using the
// osm2pgsql program, for example with the command:
// osm2pgsql -l --hstore-all --hstore-add-index --database=databasename --host=hostname --port=port --username=username --create planet_latest.osm.pbf
//
// The -l and --hstore-all flags for the osm2pgsql command are both necessary,
// and the PostGIS database should have the "hstore" extension installed before
// loading the data.
func ReadSrgSpecOSM(ctx context.Context, r io.Reader, postGISURL string) (*SrgSpecs, error) {
	// Read the surrogate specification.
	d := json.NewDecoder(r)
	var o []*SrgSpecOSM
	if err := d.Decode(&o); err != nil {
		return nil, err
	}

	// Add the db connection to each surrogate.
	srgs := NewSrgSpecs()
	for _, s := range o {
		s.postGISURL = postGISURL
		srgs.Add(s)
	}
	return srgs, nil
}

func (s *SrgSpecOSM) connectPostGIS() {
	if s.postGISURL == "" {
		panic(fmt.Errorf("PostGIS URL is required"))
	}

	// Connect to database.
	var conn *pgxpool.Pool
	var err error
	err = backoff.Retry(func() error {
		conn, err = pgxpool.Connect(context.Background(), s.postGISURL)
		if err != nil {
			return err
		}
		return nil
	}, backoff.WithMaxRetries(backoff.NewExponentialBackOff(), 10))
	if err != nil {
		panic(fmt.Errorf("unable to connect to PostGIS database %s after 10 tries: %w", s.postGISURL, err))
	}
	s.conn = conn
}

func (srg *SrgSpecOSM) backupSurrogateNames() []string { return srg.BackupSurrogateNames }
func (srg *SrgSpecOSM) region() Country                { return srg.Region }
func (srg *SrgSpecOSM) code() string                   { return srg.Code }
func (srg *SrgSpecOSM) name() string                   { return srg.Name }
func (srg *SrgSpecOSM) mergeNames() []string           { return srg.MergeNames }
func (srg *SrgSpecOSM) mergeMultipliers() []float64    { return srg.MergeMultipliers }

// getSrgData returns the spatial surrogate information for this
// surrogate definition and location, where tol is tolerance for geometry simplification.
func (srg *SrgSpecOSM) getSrgData(gridData *GridDef, inputLoc *Location, tol float64) (SearchIntersecter, error) {
	// Calculate the area of interest for our surrogate data.
	srgSR, err := proj.Parse("+proj=longlat")
	if err != nil {
		panic(err)
	}
	inputShapeT, err := inputLoc.Reproject(srgSR) // Convert input shape to surrogate SR.
	if err != nil {
		return nil, err
	}
	gridTransform, err := gridData.SR.NewTransform(srgSR)
	if err != nil {
		return nil, err
	}
	inputShapeBounds := inputShapeT.Bounds()
	srgBounds := inputShapeBounds.Copy()
	for _, cell := range gridData.Cells {
		cellT, err := cell.Transform(gridTransform) // Convert grid cell to surrogate SR.
		if err != nil {
			panic(err)
		}
		b := cellT.Bounds()
		if b.Overlaps(inputShapeBounds) {
			srgBounds.Extend(b) // Calculate bounds in surrogate SR.
		}
	}
	boundsText := fmt.Sprintf("ST_GeomFromText('Polygon((%g %g, %g %g, %g %g, %g %g, %g %g))', 4326)", // WGS84
		srgBounds.Min.X, srgBounds.Min.Y, srgBounds.Max.X, srgBounds.Min.Y, srgBounds.Max.X, srgBounds.Max.Y,
		srgBounds.Min.X, srgBounds.Max.Y, srgBounds.Min.X, srgBounds.Min.Y)

	srgCT, err := srgSR.NewTransform(gridData.SR)
	if err != nil {
		return nil, err
	}

	srgs := rtree.NewTree(25, 50)

	ctx := context.Background()
	if srg.OSMTable == "" {
		return nil, fmt.Errorf("OSM table name not specified for surrogate %s", srg.Name)
	}

	var tagKeys string
	for k := range srg.Tags {
		tagKeys += "'" + k + "',"
	}
	tagKeys = tagKeys[:len(tagKeys)-1] // Remove trailing comma.

	srg.connectPostGISOnce.Do(srg.connectPostGIS)

	rows, err := srg.conn.Query(ctx, `
	SELECT 
		hstore_to_array(tags) tags, ST_AsBinary(way) 
	FROM 
		`+srg.OSMTable+`
	WHERE
		way && `+boundsText+`
	AND
		tags ?| ARRAY [`+tagKeys+`];`)

	if err != nil {
		return nil, fmt.Errorf("reading surrogate data: %w", err)
	}
	var tags [][]byte
	var wayBytes []byte
	for rows.Next() {
		// Extract tags and geometry.
		err = rows.Scan(&tags, &wayBytes)
		if err != nil {
			return nil, fmt.Errorf("reading surrogate data: %w", err)
		}

		// See if any of the tags match the ones we want.
		keep := false
		for i := 0; i < len(tags)/2; i++ {
			key := string(tags[i*2])
			value := string(tags[i*2+1])

			if wantVals, ok := srg.Tags[key]; ok {
				if len(wantVals) == 0 {
					// If no tags are specified, keep all.
					keep = true
					break
				}
				for _, wantVal := range wantVals {
					// If the tag matches, keep this record.
					if value == wantVal {
						keep = true
						break
					}
				}
			}
		}
		if !keep {
			continue
		}

		// Convert geometry to the correct format & projection.
		g, err := wkb.Decode(wayBytes)
		if err != nil {
			return nil, fmt.Errorf("reading surrogate data: %w", err)
		}
		g, err = g.Transform(srgCT)
		if err != nil {
			return nil, fmt.Errorf("transforming surrogate data: %w", err)
		}
		if tol > 0 {
			switch gs := g.(type) {
			case geom.Simplifier:
				g = gs.Simplify(tol)
			}
		}
		srgData := &srgHolder{
			Geom:   g,
			Weight: 1,
		}
		srgs.Insert(srgData)
	}
	rows.Close()
	return srgs, nil
}

func geomCollectionToMultiPoint(gc geom.GeometryCollection, dominantType osm.GeomType) (geom.MultiPoint, error) {
	o := geom.MultiPoint{}
	for _, f := range gc {
		if gc2, ok := f.(geom.GeometryCollection); ok {
			var err error
			f, err = osmGeometry(gc2, dominantType)
			if err != nil {
				return nil, err
			}
		}
		if p, ok := f.(geom.Point); ok {
			o = append(o, p)
		}
		if p, ok := f.(geom.MultiPoint); ok {
			o = append(o, p...)
		}
	}
	if len(o) > 0 {
		return o, nil
	}
	return nil, nil
}

func geomCollectionToMultiPolygon(gc geom.GeometryCollection, dominantType osm.GeomType) (geom.MultiPolygon, error) {
	o := geom.MultiPolygon{}
	for _, f := range gc {
		if gc2, ok := f.(geom.GeometryCollection); ok {
			var err error
			f, err = osmGeometry(gc2, dominantType)
			if err != nil {
				return nil, err
			}
		}
		if p, ok := f.(geom.Polygon); ok {
			o = append(o, p)
		}
		if p, ok := f.(geom.MultiPolygon); ok {
			o = append(o, p...)
		}
	}
	if len(o) > 0 {
		return o, nil
	}
	return nil, nil
}

func geomCollectionToMultiLineString(gc geom.GeometryCollection, dominantType osm.GeomType) (geom.MultiLineString, error) {
	o := geom.MultiLineString{}
	for _, f := range gc {
		if gc2, ok := f.(geom.GeometryCollection); ok {
			var err error
			f, err = osmGeometry(gc2, dominantType)
			if err != nil {
				return nil, err
			}
		}
		if l, ok := f.(geom.LineString); ok {
			o = append(o, l)
		}
		if l, ok := f.(geom.MultiLineString); ok {
			o = append(o, l...)
		}
	}
	if len(o) > 0 {
		return o, nil
	}
	return nil, nil
}

func osmGeometry(g geom.Geom, dominantType osm.GeomType) (geom.Geom, error) {
	if gc, ok := g.(geom.GeometryCollection); ok {
		switch dominantType { // Drop features that do not match the dominant type.
		case osm.Point:
			return geomCollectionToMultiPoint(gc, dominantType)
		case osm.Poly:
			return geomCollectionToMultiPolygon(gc, dominantType)
		case osm.Line:
			return geomCollectionToMultiLineString(gc, dominantType)
		default:
			return nil, fmt.Errorf("invalid geometry type %v", dominantType)
		}
	}
	switch dominantType { // Drop features that do not match the dominant type.
	case osm.Point:
		if _, ok := g.(geom.Point); ok {
			return g, nil
		}
	case osm.Poly:
		if _, ok := g.(geom.Polygonal); ok {
			return g, nil
		}
	case osm.Line:
		if _, ok := g.(geom.Linear); ok {
			return g, nil
		}
	default:
		return nil, fmt.Errorf("invalid geometry type %v", dominantType)
	}
	return nil, nil
}
