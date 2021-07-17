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

package inmaputil

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/ctessum/geom"
	"github.com/ctessum/geom/encoding/geojson"
	"github.com/ctessum/geom/proj"
	"github.com/lnashier/viper"
	"github.com/spatialmodel/inmap"
	"github.com/spatialmodel/inmap/cloud"
	"github.com/spatialmodel/inmap/emissions/aep"
	"github.com/spatialmodel/inmap/emissions/aep/aeputil"
	"github.com/spf13/cast"
)

// checkOutputVars removes end lines and expands environment
// variables in the output variables.
func checkOutputVars(vars map[string]string) (map[string]string, error) {
	if len(vars) == 0 {
		return nil, fmt.Errorf("there are no variables specified for output. Please fill in " +
			"the OutputVariables configuration and try again.")
	}
	for k, v := range vars {
		v = strings.Replace(v, "\r\n", " ", -1)
		v = strings.Replace(v, "\n", " ", -1)
		vars[os.ExpandEnv(k)] = os.ExpandEnv(v)
	}
	return vars, nil
}

// expandStringSlice expands the environment variables in a slice of strings.
func expandStringSlice(s []string) []string {
	for i := 0; i < len(s); i++ {
		s[i] = os.ExpandEnv(s[i])
	}
	return s
}

// removeShpSupportFiles deletes from the list of files any that do not
// end in `.shp`.
func removeShpSupportFiles(files []string) []string {
	var o []string
	for _, s := range files {
		if strings.HasSuffix(s, ".shp") {
			o = append(o, s)
		}
	}
	return o
}

// checkOutputFile makes sure that the output file is specified and its
// directory exists, and expand any environment variables.
func checkOutputFile(f string) (string, error) {
	if f == "" {
		return "", fmt.Errorf(`you need to specify an output file configuration variable (for example: OutputFile="output.shp"`)
	}
	f = os.ExpandEnv(f)
	if IsBlob(f) {
		url, err := url.Parse(f)
		if err != nil {
			return f, err
		}
		_, err = cloud.OpenBucket(context.TODO(), url.Scheme+"://"+url.Host)
		if err != nil {
			return f, fmt.Errorf("inmap: error when checking OutputFile location: %v", err)
		}
		return f, nil
	}
	outdir := filepath.Dir(f)
	if _, err := os.Stat(outdir); err != nil {
		return f, fmt.Errorf("inmap: the OutputFile directory doesn't exist: %v", err)
	}
	return f, nil
}

// checkLogFile fills in a default value for the log file path if one isn't
// specified.
func checkLogFile(logFile, outputFile string) string {
	if logFile == "" {
		logFile = strings.TrimSuffix(outputFile, filepath.Ext(outputFile)) + ".log"
	}
	return logFile
}

// checkEmissionUnits expands any environment variables in the emissions
// units and ensures that an acceptable value was specified.
func checkEmissionUnits(u string) (string, error) {
	u = os.ExpandEnv(u)
	if u != "tons/year" && u != "kg/year" && u != "ug/s" && u != "μg/s" {
		return u, fmt.Errorf("the EmissionUnits variable in the configuration file "+
			"needs to be set to either tons/year, kg/year, ug/s, or μg/s, but is currently set to `%s`",
			u)
	}
	return u, nil
}

// spatialRef returns the spatial reference associated with config,
// as defined by the GridProj field.
func spatialRef(config *inmap.VarGridConfig) (*proj.SR, error) {
	if config.GridProj == "" {
		return nil, fmt.Errorf("you need to specify the InMAP grid projection in the " +
			"'GridProj' configuration variable.")
	}
	sr, err := proj.Parse(config.GridProj)
	if err != nil {
		return nil, fmt.Errorf("the following error occured while parsing the InMAP grid"+
			"projection (the GridProj variable): %v", err)
	}
	return sr, nil
}

// VarGridConfig unmarshals a viper configuration for a variable grid.
func VarGridConfig(cfg *viper.Viper) (*inmap.VarGridConfig, error) {
	xNests, err := toIntSliceE(cfg.Get("VarGrid.Xnests"))
	if err != nil {
		return nil, fmt.Errorf("VarGrid.Xnests: %v", err)
	}
	yNests, err := toIntSliceE(cfg.Get("VarGrid.Ynests"))
	if err != nil {
		return nil, fmt.Errorf("VarGrid.Ynests: %v", err)
	}
	ctx := context.TODO()
	c := inmap.VarGridConfig{
		VariableGridXo:       cfg.GetFloat64("VarGrid.VariableGridXo"),
		VariableGridYo:       cfg.GetFloat64("VarGrid.VariableGridYo"),
		VariableGridDx:       cfg.GetFloat64("VarGrid.VariableGridDx"),
		VariableGridDy:       cfg.GetFloat64("VarGrid.VariableGridDy"),
		Xnests:               xNests,
		Ynests:               yNests,
		HiResLayers:          cfg.GetInt("VarGrid.HiResLayers"),
		PopDensityThreshold:  cfg.GetFloat64("VarGrid.PopDensityThreshold"),
		PopThreshold:         cfg.GetFloat64("VarGrid.PopThreshold"),
		PopConcThreshold:     cfg.GetFloat64("VarGrid.PopConcThreshold"),
		CensusFile:           maybeDownload(ctx, os.ExpandEnv(cfg.GetString("VarGrid.CensusFile")), outChan()),
		CensusPopColumns:     expandStringSlice(cfg.GetStringSlice("VarGrid.CensusPopColumns")),
		PopGridColumn:        os.ExpandEnv(cfg.GetString("VarGrid.PopGridColumn")),
		MortalityRateFile:    maybeDownload(ctx, os.ExpandEnv(cfg.GetString("VarGrid.MortalityRateFile")), outChan()),
		MortalityRateColumns: GetStringMapString("VarGrid.MortalityRateColumns", cfg),
		GridProj:             os.ExpandEnv(cfg.GetString("VarGrid.GridProj")),
	}

	vars := []float64{c.VariableGridDx, c.VariableGridDy}
	varNames := []string{"VarGrid.VariableGridDx", "VarGrid.VariableGridDy"}
	for i, v := range vars {
		if !(v > 0) {
			return nil, fmt.Errorf("parsing grid configuration: %s=%g but should be >0", varNames[i], v)
		}
	}

	vars2 := [][]int{c.Xnests, c.Ynests}
	varNames = []string{"VarGrid.Xnests", "VarGrid.Ynests"}
	for i, v := range vars2 {
		if len(v) == 0 {
			return nil, fmt.Errorf("parsing grid configuration: %s is not specified", varNames[i])
		}
	}
	if len(c.Xnests) != len(c.Ynests) {
		return nil, fmt.Errorf("parsing grid configuration: VarGrid.Xnests and VarGrid.Ynests must be the same length; %d != %d", len(c.Xnests), len(c.Ynests))
	}

	for k, v := range c.MortalityRateColumns {
		c.MortalityRateColumns[os.ExpandEnv(k)] = os.ExpandEnv(v)
	}

	return &c, nil
}

// aeputilConfig unmarshals an aeputil inventory and spatial configuration.
func aeputilConfig(cfg *viper.Viper) (*aeputil.InventoryConfig, *aeputil.SpatialConfig, error) {
	outChan := outChan()

	neiFiles, err := getStringMapStringSlice("aep.InventoryConfig.NEIFiles", cfg)
	if err != nil {
		return nil, nil, fmt.Errorf("inmaputil: parsing config variable aep.InventoryConfig.NEIFiles: %v", err)
	}
	for k, vs := range neiFiles {
		for i, v := range vs {
			neiFiles[k][i] = maybeDownload(context.TODO(), os.ExpandEnv(v), outChan)
		}
	}

	coardsFiles, err := getStringMapStringSlice("aep.InventoryConfig.COARDSFiles", cfg)
	if err != nil {
		return nil, nil, fmt.Errorf("inmaputil: parsing config variable aep.InventoryConfig.COARDSFiles: %v", err)
	}
	for k, vs := range coardsFiles {
		for i, v := range vs {
			coardsFiles[k][i] = maybeDownload(context.TODO(), os.ExpandEnv(v), outChan)
		}
	}

	srgSpecSMOKE := maybeDownload(context.TODO(), os.ExpandEnv(cfg.GetString("aep.SrgSpecSMOKE")), outChan)
	srgSpecOSM := maybeDownload(context.TODO(), os.ExpandEnv(cfg.GetString("aep.SrgSpecOSM")), outChan)
	var gridRef []string
	for _, g := range cfg.GetStringSlice("aep.GridRef") {
		gridRef = append(gridRef, maybeDownload(context.TODO(), g, outChan))
	}

	i := &aeputil.InventoryConfig{
		NEIFiles:              neiFiles,
		COARDSFiles:           coardsFiles,
		COARDSYear:            cfg.GetInt("aep.InventoryConfig.COARDSYear"),
		InputUnits:            cfg.GetString("aep.InventoryConfig.InputUnits"),
		SrgSpecSMOKE:          srgSpecSMOKE,
		SrgSpecOSM:            srgSpecOSM,
		PostGISURL:            os.ExpandEnv(cfg.GetString("aep.PostGISURL")),
		SrgShapefileDirectory: cfg.GetString("aep.SrgShapefileDirectory"),
		GridRef:               gridRef,
		SCCExactMatch:         cfg.GetBool("aep.SCCExactMatch"),
	}
	i.PolsToKeep = aep.Speciation{
		"VOC":   {},
		"NOx":   {},
		"NH3":   {},
		"SOx":   {},
		"PM2_5": {},
	}

	s := &aeputil.SpatialConfig{
		SrgSpecSMOKE:          srgSpecSMOKE,
		SrgSpecOSM:            srgSpecOSM,
		PostGISURL:            os.ExpandEnv(cfg.GetString("aep.PostGISURL")),
		SrgShapefileDirectory: cfg.GetString("aep.SrgShapefileDirectory"),
		SCCExactMatch:         cfg.GetBool("aep.SCCExactMatch"),
		GridRef:               gridRef,
		OutputSR:              os.ExpandEnv(cfg.GetString("VarGrid.GridProj")),
		InputSR:               cfg.GetString("aep.SpatialConfig.InputSR"),
		SpatialCache:          cfg.GetString("aep.SpatialConfig.SpatialCache"),
		SrgDataCache:          cfg.GetString("aep.SpatialConfig.SrgDataCache"),
		MaxCacheEntries:       cfg.GetInt("aep.SpatialConfig.MaxCacheEntries"),
		GridName:              cfg.GetString("aep.SpatialConfig.GridName"),
	}

	return i, s, nil
}

func toIntSliceE(s interface{}) ([]int, error) {
	if v, ok := s.([]interface{}); ok {
		o := make([]int, len(v))
		for i, val := range v {
			o[i] = int(val.(int64))
		}
		return o, nil
	}
	var o []int
	if err := json.Unmarshal([]byte(s.(string)), &o); err != nil {
		return nil, err
	}
	return o, nil
}

// GetStringMapString returns a map[string]string from a viper configuration,
// accounting for the fact that it might be a json object if it was set
// from a command line argument.
func GetStringMapString(varName string, cfg *viper.Viper) map[string]string {
	i := cfg.Get(varName)
	switch i.(type) {
	case map[string]string:
		return i.(map[string]string)
	case map[string]interface{}:
		return cast.ToStringMapString(i)
	case string:
		b := bytes.NewBuffer(([]byte)(i.(string)))
		d := json.NewDecoder(b)
		o := make(map[string]string)
		if err := d.Decode(&o); err != nil {
			panic(err)
		}
		return o
	default:
		panic(fmt.Errorf("invalid type for getStringMapString variable %s: %#v", varName, i))
	}
}

// getStringMapStringSlice returns a map[string][]string from a viper configuration,
// accounting for the fact that it might be a json object if it was set
// from a command line argument.
func getStringMapStringSlice(varName string, cfg *viper.Viper) (map[string][]string, error) {
	i := cfg.Get(varName)
	switch i.(type) {
	case map[string][]string:
		return i.(map[string][]string), nil
	case map[string]interface{}:
		return cast.ToStringMapStringSliceE(i)
	case string:
		if i == "" {
			return make(map[string][]string), nil
		}
		b := bytes.NewBuffer(([]byte)(i.(string)))
		d := json.NewDecoder(b)
		o := make(map[string][]string)
		if err := d.Decode(&o); err != nil {
			return nil, err
		}
		return o, nil
	default:
		panic(fmt.Errorf("invalid type for getStringMapString variable %s: %#v", varName, i))
	}
}

// parseMask returns a mask polygon represented by the
// given GeoJSON file.
func parseMask(maskGeoJSONFile string) (geom.Polygon, error) {
	var mask geom.Polygon
	if m := maskGeoJSONFile; m != "" {
		f, err := os.Open(os.ExpandEnv(m))
		if err != nil {
			return nil, fmt.Errorf("opening emissions mask file: %w", err)
		}
		b, err := ioutil.ReadAll(f)
		if err != nil {
			return nil, fmt.Errorf("reading emissions mask file: %w", err)
		}
		j, err := geojson.Decode(b)
		if err != nil {
			return nil, fmt.Errorf("decoding EmissionMaskGEOJSON: %w", err)
		}
		switch msk := j.(type) {
		case geom.Polygon:
			mask = msk
		case geom.MultiPolygon:
			for _, p := range msk {
				mask = append(mask, p...)
			}
		default:
			return nil, fmt.Errorf("invalid emission mask geometry type %T", j)
		}
	}
	return mask, nil
}
