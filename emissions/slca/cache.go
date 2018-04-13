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

package slca

import (
	"encoding/gob"
	"fmt"

	"bitbucket.org/ctessum/sparse"
)

type requestPayload struct {
	edge       *ResultEdge
	results    *Results
	spatialRef *SpatialRef
	key        string
}

func newRequestPayload(sr *SpatialResults, e *ResultEdge) (*requestPayload, error) {
	var key string
	r := sr.Results
	p := r.GetFromNode(e).Process
	switch p.Type() {
	case Stationary:
		if p.SpatialRef() == nil {
			return nil, fmt.Errorf("stationary process %s (id=%s) has no SpatialRef", p.GetName(), p.GetIDStr())
		}
		key = p.SpatialRef().Key()
	case Transportation, Vehicle: // TODO: vehicle needs its own key.
		key = "transportation"
	case NoSpatial:
		key = "NoSpatial"
	default:
		return nil, fmt.Errorf("in slca.newRequest: can't make key for type %v", p.Type())
	}

	return &requestPayload{
		edge:       e,
		results:    r,
		spatialRef: p.SpatialRef(),
		key:        key,
	}, nil
}

func init() {
	// These are the types that will be stored in the cache.
	gob.Register(sparse.SparseArray{})
	gob.Register(sparse.DenseArray{})
	gob.Register(map[string][]float64{})
	gob.Register(map[string]map[string]*sparse.DenseArray{})
}
