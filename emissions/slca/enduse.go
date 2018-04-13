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

// endUse is a dummy process type used a place holder
// for the end use of a life cycle.
type endUse int

func (e endUse) GetName() string { return "End use" }

func (e endUse) GetIDStr() string { return "End use" }

func (e endUse) Type() ProcessType { return Stationary }

func (e endUse) OnsiteResults(Pathway, Output, LCADB) *OnsiteResults { return nil }

func (e endUse) SpatialRef() *SpatialRef { return &SpatialRef{NoSpatial: true} }

func (e endUse) GetOutput(Resource, LCADB) Output { return nil }
func (e endUse) GetMainOutput(LCADB) Output       { return nil }
