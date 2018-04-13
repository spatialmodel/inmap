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
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/ctessum/geom"
	"github.com/ctessum/geom/carto"
	"github.com/ctessum/geom/proj"
	"github.com/golang/groupcache/lru"
	"github.com/spatialmodel/epi"
	"gonum.org/v1/plot/vg"
	"gonum.org/v1/plot/vg/draw"
	"gonum.org/v1/plot/vg/vgimg"
)

func (db *DB) mapDataServer(resultRequestChan chan *resultRequest,
	mapDataRequestChan chan *mapDataRequest) {

	const (
		GridProj   = "+proj=lcc +lat_1=33.000000 +lat_2=45.000000 +lat_0=40.000000 +lon_0=-97.000000 +x_0=0 +y_0=0 +a=6370997.000000 +b=6370997.000000 +to_meter=1"
		webMapProj = "+proj=merc +a=6378137 +b=6378137 +lat_ts=0.0 +lon_0=0.0 +x_0=0.0 +y_0=0 +k=1.0 +units=m +nadgrids=@null +no_defs"
	)

	// webMapSR is the spatial reference for web mapping.
	webMapSR, err := proj.Parse(webMapProj)
	if err != nil {
		panic(fmt.Errorf("slca: while parsing webMapProj: %v", err))
	}

	gridSR, err := proj.Parse(GridProj)
	if err != nil {
		panic(fmt.Errorf("slca: while parsing GridProj: %v", err))
	}
	webMapTrans, err := gridSR.NewTransform(webMapSR)
	if err != nil {
		panic(fmt.Errorf("slca: while creating transform: %v", err))
	}

	srCells := db.CSTConfig.sr.Geometry()
	cells := make([]geom.Geom, len(srCells))
	for i, c := range srCells {
		cells[i], err = c.Transform(webMapTrans)
		if err != nil {
			panic(err)
		}
	}

	for request := range mapDataRequestChan {
		request.Lock()
		log.Println("mapDataServer got request")
		result := getResults(resultRequestChan, request.r)
		if request.handleErr(result.Err) {
			continue
		}
		request.mapDatas = make(map[string]*carto.MapData)

		spatialResults := NewSpatialResults(result.Results, db)

		// Add the emissions
		emisGridData, err := spatialResults.Emissions()
		if request.handleErr(err) {
			continue
		}

		for varname, data := range emisGridData {
			request.mapDatas[varname.GetName()] = carto.NewMapData(len(cells), carto.LinCutoff)
			m := request.mapDatas[varname.GetName()]
			m.Shapes = cells
			for i, v := range data.Elements {
				m.Data[i] = v
			}
			m.Cmap.AddArray(m.Data)
			m.Cmap.Set()
		}

		// Add the air quality results
		aqData, err := spatialResults.Concentrations()
		if request.handleErr(err) {
			continue
		}
		for varname, data := range aqData {
			request.mapDatas[varname] =
				carto.NewMapData(len(cells), carto.LinCutoff)
			m := request.mapDatas[varname]
			m.Shapes = cells
			for i, v := range data.Elements {
				m.Data[i] = v
			}
			m.Cmap.AddArray(m.Data)
			m.Cmap.Set()
		}

		// Add the health results
		healthData, err := spatialResults.Health(epi.NasariACS)
		if request.handleErr(err) {
			continue
		}
		for varname, data := range healthData {
			request.mapDatas[varname] =
				carto.NewMapData(len(cells), carto.LinCutoff)
			m := request.mapDatas[varname]
			m.Shapes = cells
			for i, v := range data[totalPM25].Elements {
				m.Data[i] = v
			}
			m.Cmap.AddArray(m.Data)
			m.Cmap.Set()
		}
		log.Println("mapDataServer is about to send result")
		request.Unlock()
		fmt.Println("xxxxxxxxxxxxxxx")
		for name := range request.mapDatas {
			fmt.Println(name)
		}
		request.returnChan <- request
	}
}

func (db *DB) resultsMapTileHandler(
	mapDataRequestChan chan *mapDataRequest) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")

		// parse request data
		vars := make([]int64, 4)
		for i, v := range []string{"zoom", "x", "y"} {
			str := r.FormValue(v)
			if str == "" {
				handleErrHTTP(fmt.Errorf("%s not specified", v), w)
				return
			}
			var err error
			vars[i], err = strconv.ParseInt(str, 10, 64)
			handleErrHTTP(err, w)
		}
		varname := r.FormValue("varname")
		if varname == "" {
			handleErrHTTP(fmt.Errorf("varname not specified"), w)
			return
		}

		// get result
		result := getMapData(mapDataRequestChan, r)
		handleErrHTTP(result.err, w)

		// create map tile
		if md, ok := result.mapDatas[varname]; ok {
			err := md.WriteGoogleMapTile(w,
				int(vars[0]), int(vars[1]), int(vars[2]))
			handleErrHTTP(err, w)
		} else {
			log.Printf("No map data for %s", varname)
		}
	}
}

func (db *DB) resultsMapLegendHandler(
	mapDataRequestChan chan *mapDataRequest) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")

		// parse request data
		varname := r.FormValue("varname")
		if varname == "" {
			handleErrHTTP(fmt.Errorf("varname not specified"), w)
			return
		}

		// get result
		result := getMapData(mapDataRequestChan, r)
		handleErrHTTP(result.err, w)

		// create legend
		const (
			LegendWidth  = 3.70 * vg.Inch
			LegendHeight = LegendWidth * 0.1067
		)
		img := vgimg.PngCanvas{Canvas: vgimg.New(LegendWidth, LegendHeight)}
		canvas := draw.New(img)
		if md, ok := result.mapDatas[varname]; ok {
			handleErrHTTP(md.Cmap.Legend(&canvas, "Units!"), w)
			_, err := img.WriteTo(w)
			handleErrHTTP(err, w)
		} else {
			log.Printf("No map data for %s", varname)
		}
	}
}

func getMapData(mapDataRequestChan chan *mapDataRequest,
	r *http.Request) *mapDataRequest {
	request := &mapDataRequest{
		r:          r,
		returnChan: make(chan *mapDataRequest),
	}
	mapDataRequestChan <- request
	result := <-request.returnChan
	return result
}

type mapDataRequest struct {
	sync.Mutex
	r          *http.Request
	returnChan chan *mapDataRequest
	mapDatas   map[string]*carto.MapData
	err        error
}

func mapDataCache(inChan chan *mapDataRequest) (outChan chan *mapDataRequest) {
	outChan = make(chan *mapDataRequest)
	const maxEntries = 10 // max number of items in the cache
	cache := lru.New(maxEntries)
	go func() {
		for request := range inChan {
			pathwayName := request.r.FormValue("pathselect")
			unitsStr := request.r.FormValue("units")
			amtStr := request.r.FormValue("amt")
			varname := request.r.FormValue("varname")
			key := fmt.Sprintf("%s_%s_%s_%s", pathwayName,
				unitsStr, amtStr, varname)
			if d, ok := cache.Get(key); ok {
				data := d.(*mapDataRequest)
				data.Lock()
				// Make sure request has finished processing
				if data.mapDatas != nil {
					data.Unlock()
					request.returnChan <- data
				} else {
					// if the result is already processing, wait then check the cache
					// again.
					data.Unlock()
					time.Sleep(time.Second)
					inChan <- data
				}
				continue
			}
			cache.Add(key, request)
			outChan <- request
		}
	}()
	return
}

// handleErr deals with an error and returns true if an error was encountered
func (mr *mapDataRequest) handleErr(err error) bool {
	if err != nil {
		log.Println(err)
		mr.err = err
		mr.returnChan <- mr
		mr.Unlock()
		return true
	}
	return false
}
