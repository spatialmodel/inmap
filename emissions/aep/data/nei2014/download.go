//+build ignore

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

package main

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	ftp "github.com/remogatto/ftpget"
)

var dir string

func init() {
	flag.StringVar(&dir, "dir", "", "the directory to download the files to")
}

func main() {
	// This is the list of files to download.
	files := []string{
		"ftp.epa.gov/EmisInventory/2014platform/v1/README_2014v1_nata_package.txt",
		"ftp.epa.gov/EmisInventory/2014platform/v1/ancillary_data/ge_dat_for_2014fa_nata_gridding.zip",
		"ftp.epa.gov/EmisInventory/2014platform/v1/ancillary_data/ge_dat_for_2014fa_nata_other.zip",
		"ftp.epa.gov/EmisInventory/2014platform/v1/ancillary_data/ge_dat_for_2014fa_nata_speciation.zip",
		"ftp.epa.gov/EmisInventory/2014platform/v1/ancillary_data/ge_dat_for_2014fa_nata_temporal.zip",
		"ftp.epa.gov/EmisInventory/2014platform/v1/ancillary_data/ocean_chlorine.zip",
		"ftp.epa.gov/EmisInventory/2014platform/v1/ancillary_data/volcanic_mercury.zip",
		"ftp.epa.gov/EmisInventory/2014platform/v1/spatial_surrogates/shapefiles/2014shapefiles.Census.tar.gz",
		"ftp.epa.gov/EmisInventory/2014platform/v1/spatial_surrogates/shapefiles/2014shapefiles.eia.tar.gz",
		"ftp.epa.gov/EmisInventory/2014platform/v1/spatial_surrogates/shapefiles/2014shapefiles.epa_shipping_ports.tar.gz",
		"ftp.epa.gov/EmisInventory/2014platform/v1/spatial_surrogates/shapefiles/2014shapefiles.extended_idle.tar.gz",
		"ftp.epa.gov/EmisInventory/2014platform/v1/spatial_surrogates/shapefiles/2014shapefiles.golf_courses.tar.gz",
		"ftp.epa.gov/EmisInventory/2014platform/v1/spatial_surrogates/shapefiles/2014shapefiles.hpdi_og.tar.gz",
		"ftp.epa.gov/EmisInventory/2014platform/v1/spatial_surrogates/shapefiles/2014shapefiles.hpms.tar.gz",
		"ftp.epa.gov/EmisInventory/2014platform/v1/spatial_surrogates/shapefiles/2014shapefiles.nlcd.tar.gz",
		"ftp.epa.gov/EmisInventory/2014platform/v1/spatial_surrogates/shapefiles/2014shapefiles.ntad.tar.gz",
		"ftp.epa.gov/EmisInventory/2014platform/v1/spatial_surrogates/shapefiles/2014shapefiles.tiger_rail.tar.gz",
		"ftp.epa.gov/EmisInventory/2014platform/v1/spatial_surrogates/shapefiles/2014shapefiles.usfs_timber.tar.gz",
		"ftp.epa.gov/EmisInventory/2014platform/v1/spatial_surrogates/shapefiles/2014shapefiles.usgs_mines.tar.gz",
		"ftp.epa.gov/EmisInventory/2014platform/v1/spatial_surrogates/shapefiles/cty_pophu2k_revised.zip",
		"ftp.epa.gov/EmisInventory/2014platform/v1/spatial_surrogates/shapefiles/mexico_shapefiles.tar.gz",
		"ftp.epa.gov/EmisInventory/2014platform/v1/spatial_surrogates/shapefiles/pr_shape.tar.gz",
		"ftp.epa.gov/EmisInventory/2014platform/v1/spatial_surrogates/shapefiles/us_tracts_shape.tar.gz",
		"ftp.epa.gov/EmisInventory/2014platform/v1/spatial_surrogates/shapefiles/usvi_shape.tar.gz",
		"ftp.epa.gov/EmisInventory/2014platform/v1/spatial_surrogates/Spatial_Allocator_SrgTools_2014Platform.30Sep2016.tar",
		"ftp.epa.gov/EmisInventory/2014platform/v1/2014emissions/2014fa_nata_cb6cmaq_14j_inputs_biogenics.zip",
		"ftp.epa.gov/EmisInventory/2014platform/v1/2014emissions/2014fa_nata_cb6cmaq_14j_inputs_cem.zip",
		"ftp.epa.gov/EmisInventory/2014platform/v1/2014emissions/2014fa_nata_cb6cmaq_14j_inputs_nonpoint.zip",
		"ftp.epa.gov/EmisInventory/2014platform/v1/2014emissions/2014fa_nata_cb6cmaq_14j_inputs_nonroad_part1.zip",
		"ftp.epa.gov/EmisInventory/2014platform/v1/2014emissions/2014fa_nata_cb6cmaq_14j_inputs_nonroad_part2.zip",
		"ftp.epa.gov/EmisInventory/2014platform/v1/2014emissions/2014fa_nata_cb6cmaq_14j_inputs_nonroad_part3.zip",
		"ftp.epa.gov/EmisInventory/2014platform/v1/2014emissions/2014fa_nata_cb6cmaq_14j_inputs_nonroad_part4.zip",
		"ftp.epa.gov/EmisInventory/2014platform/v1/2014emissions/2014fa_nata_cb6cmaq_14j_inputs_onroad.zip",
		"ftp.epa.gov/EmisInventory/2014platform/v1/2014emissions/2014fa_nata_cb6cmaq_14j_inputs_oth_part1.zip",
		"ftp.epa.gov/EmisInventory/2014platform/v1/2014emissions/2014fa_nata_cb6cmaq_14j_inputs_oth_part2.zip",
		"ftp.epa.gov/EmisInventory/2014platform/v1/2014emissions/2014fa_nata_cb6cmaq_14j_inputs_point.zip",
		"ftp.epa.gov/EmisInventory/2014platform/v1/2014emissions/2014fa_nata_cb6cmaq_14j_inputs_ptfire.zip",
		"ftp.epa.gov/EmisInventory/2014/flat_files/SmokeFlatFile_ONROAD_20160910.csv.zip",
		"ftp.epa.gov/EmisInventory/2011v6/v1platform/spatial_surrogates/shapefiles/2010shapefiles.misc.tar.zip",
		"ftp.epa.gov/EmisInventory/2011v6/v1platform/spatial_surrogates/shapefiles/2010shapefiles.fema.tar.zip",
		"ftp.epa.gov/EmisInventory/2011v6/v1platform/spatial_surrogates/shapefiles/2010shapefiles.offshore.tar.zip",
		"ftp.epa.gov/EmisInventory/emiss_shp2003/us/airport-area.dbf",
		"ftp.epa.gov/EmisInventory/emiss_shp2003/us/airport-area.shp",
		"ftp.epa.gov/EmisInventory/emiss_shp2003/us/airport-area.shx",
		"ftp.epa.gov/EmisInventory/emiss_shp2003/us/airport-area.sbn",
		"ftp.epa.gov/EmisInventory/emiss_shp2003/us/airport-area.sbx",
		"ftp.epa.gov/EmisInventory/emiss_shp2003/us/airport-area.prj",
		"ftp.epa.gov/EmisInventory/2011v6/v2platform/spatial_surrogates/CANADA2010_shapefiles_part1.zip",
		"ftp.epa.gov/EmisInventory/2011v6/v2platform/spatial_surrogates/CANADA2010_shapefiles_part2.zip",
		"ftp.epa.gov/EmisInventory/2011v6/v2platform/spatial_surrogates/CANADA2010_shapefiles_part3.zip",
		"http://www12.statcan.gc.ca/census-recensement/2011/geo/bound-limit/files-fichiers/2016/lcd_000b16a_e.zip", // Canadian census divisions
	}

	// copyFiles specifies files that should be copied to other files.
	// The reason for this is that some shapefiles do not come with .prj files,
	// so we copy matching ones from other shapefiles.
	var copyFiles = map[string][]string{ // Lambert projections
		"Canada_2010_surrogate_v1/NAESI/SHAPEFILE/gisnodat.prj": []string{
			"Canada_2010_surrogate_v1/NAESI/SHAPEFILE/naesi_dat.prj",
			"Canada_2010_surrogate_v1/Non_NAESI/SHAPEFILE/da2006_pop_labour.prj",
			"Canada_2010_surrogate_v1/Non_NAESI/SHAPEFILE/naesi_fert.prj",
			"Canada_2010_surrogate_v1/Non_NAESI/SHAPEFILE/naesi_livestk.prj",
		},
		"Canada_2010_surrogate_v1/Non_NAESI/SHAPEFILE/pr2001ca_regions_simplify.prj": []string{ // lat-lon projections
			"Canada_2010_surrogate_v1/Non_NAESI/SHAPEFILE/lowmedjet_ll.prj",
			"Canada_2010_surrogate_v1/Non_NAESI/SHAPEFILE/CANRAIL.prj",
			"Canada_2010_surrogate_v1/Non_NAESI/SHAPEFILE/chboisv8S0_.prj",
			"Canada_2010_surrogate_v1/Non_NAESI/SHAPEFILE/marine.prj",
			"Canada_2010_surrogate_v1/Non_NAESI/SHAPEFILE/merge123_10km.prj",
			"Canada_2010_surrogate_v1/Non_NAESI/SHAPEFILE/paved4.prj",
			"Canada_2010_surrogate_v1/Non_NAESI/SHAPEFILE/treesa.prj",
			"Canada_2010_surrogate_v1/Non_NAESI/SHAPEFILE/ua2001.prj",
			"Canada_2010_surrogate_v1/Non_NAESI/SHAPEFILE/unpaved4.prj",
			"Canada_2010_surrogate_v1/Non_NAESI/SHAPEFILE/unpaved5.prj",
			"Canada_2010_surrogate_v1/Non_NAESI/SHAPEFILE/pr2001ca_regions_bc_waste.prj",
			"mexico/hwybdrx.prj", // This one doesn't look quite right but I'm not sure what it should be.
		},
	}

	flag.Parse()
	if dir == "" {
		log.Fatal("Please specify the download location as an argument (e.g. --dir=$HOME).")
	}

	for _, file := range files {
		log.Printf("downloading %s", file)
		b := new(bytes.Buffer)
		if strings.Contains(file, "http://") {
			resp, err := http.Get(file)
			if err != nil {
				log.Fatal(err)
			}
			_, err = io.Copy(b, resp.Body)
			if err != nil {
				log.Fatal(err)
			}
			resp.Body.Close()
		} else {
			err := ftp.Get(file, b)
			if err != nil {
				log.Fatal(err)
			}
		}
		switch filepath.Ext(file) {
		case ".zip":
			if filepath.Ext(strings.Trim(file, ".zip")) == ".tar" {
				unTarZip(dir, b)
			} else {
				unZip(dir, b)
			}
		case ".gz":
			unTarGZ(dir, b)
		case ".tar":
			unTar(dir, b)
		default:
			saveFile(dir, file, b)
		}
	}
	for src, dsts := range copyFiles {
		fullSrc := filepath.Join(dir, src)
		for _, dst := range dsts {
			fullDst := filepath.Join(dir, dst)
			if err := copyFile(fullSrc, fullDst); err != nil {
				log.Fatalf("copying file %s to %s: %v", fullSrc, fullDst, err)
			}
		}
	}
}

// copyFile copies the file src to file
// dst. The file will be created if it does not already exist. If the
// destination file exists, all it's contents will be replaced by the contents
// of the source file.
func copyFile(src, dst string) (err error) {
	in, err := os.Open(src)
	if err != nil {
		return
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return
	}
	if _, err = io.Copy(out, in); err != nil {
		return
	}
	err = out.Sync()
	out.Close()
	return
}

func saveFile(dir, filename string, b *bytes.Buffer) {
	saveLoc := filepath.Join(dir, filepath.Base(filename))
	dst, err := os.Create(saveLoc)
	if err != nil {
		log.Fatal(err)
	}
	_, err = io.Copy(dst, b)
	if err != nil {
		log.Fatal(err)
	}
	dst.Close()
}

func unTarGZ(dir string, file *bytes.Buffer) {
	rGZ, err := gzip.NewReader(file)
	if err != nil {
		log.Fatal(err)
	}
	unTar(dir, rGZ)
}

func unTar(dir string, file io.Reader) {
	r := tar.NewReader(file)

	for {
		header, err := r.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			log.Fatal(err)
		}
		saveLoc := filepath.Join(dir, header.Name)

		err = os.MkdirAll(filepath.Dir(saveLoc), os.ModePerm)
		if err != nil {
			log.Fatal(err)
		}

		switch header.Typeflag {
		case tar.TypeDir:
		case tar.TypeReg, tar.TypeRegA:
			fmt.Println("saving ", saveLoc)
			dst, err := os.Create(saveLoc)
			if err != nil {
				log.Fatal(err)
			}
			_, err = io.Copy(dst, r)
			if err != nil {
				log.Fatal(err)
			}
			dst.Close()
		default:
			log.Fatalf("unsupported type %c in file %s", header.Typeflag, header.Name)
		}
	}
}

func unZip(dir string, file *bytes.Buffer) {
	r, err := zip.NewReader(bytes.NewReader(file.Bytes()), int64(file.Len()))
	if err != nil {
		log.Fatal(err)
	}
	for _, zf := range r.File {
		saveLoc := filepath.Join(dir, zf.Name)

		err = os.MkdirAll(filepath.Dir(saveLoc), os.ModePerm)
		if err != nil {
			log.Fatal(err)
		}

		if filepath.Ext(saveLoc) == "" {
			continue
		}
		fmt.Println("saving ", saveLoc)
		dst, err := os.Create(saveLoc)
		if err != nil {
			log.Fatal(err)
		}
		src, err := zf.Open()
		if err != nil {
			log.Fatal(err)
		}

		_, err = io.Copy(dst, src)
		if err != nil {
			log.Fatal(err)
		}
		dst.Close()
		src.Close()
	}
}

func unTarZip(dir string, file *bytes.Buffer) {
	r, err := zip.NewReader(bytes.NewReader(file.Bytes()), int64(file.Len()))
	if err != nil {
		log.Fatal(err)
	}
	for _, zf := range r.File {
		r, err := zf.Open()
		if err != nil {
			log.Fatal(err)
		}
		unTar(dir, r)
	}
}
