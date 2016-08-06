package proj

type datumDef struct {
	towgs84   []float64
	ellipse   string
	datumName string
	nadgrids  []string
}

var datumDefs = map[string]datumDef{
	"wgs84": datumDef{
		towgs84:   []float64{0., 0., 0.},
		ellipse:   "WGS84",
		datumName: "WGS84",
	},
	"ch1903": datumDef{
		towgs84:   []float64{674.374, 15.056, 405.346},
		ellipse:   "bessel",
		datumName: "swiss",
	},
	"ggrs87": datumDef{
		towgs84:   []float64{-199.87, 74.79, 246.62},
		ellipse:   "GRS80",
		datumName: "Greek_Geodetic_Reference_System_1987",
	},
	"nad83": datumDef{
		towgs84:   []float64{0., 0., 0.},
		ellipse:   "GRS80",
		datumName: "North_American_Datum_1983",
	},
	"nad27": datumDef{
		nadgrids:  []string{"@conus", "@alaska", "@ntv2_0.gsb", "@ntv1_can.dat"},
		ellipse:   "clrk66",
		datumName: "North_American_Datum_1927",
	},
	"potsdam": datumDef{
		towgs84:   []float64{606.0, 23.0, 413.0},
		ellipse:   "bessel",
		datumName: "Potsdam Rauenberg 1950 DHDN",
	},
	"carthage": datumDef{
		towgs84:   []float64{-263.0, 6.0, 431.0},
		ellipse:   "clark80",
		datumName: "Carthage 1934 Tunisia",
	},
	"hermannskogel": datumDef{
		towgs84:   []float64{653.0, -212.0, 449.0},
		ellipse:   "bessel",
		datumName: "Hermannskogel",
	},
	"ire65": datumDef{
		towgs84:   []float64{482.530, -130.596, 564.557, -1.042, -0.214, -0.631, 8.15},
		ellipse:   "mod_airy",
		datumName: "Ireland 1965",
	},
	"rassadiran": datumDef{
		towgs84:   []float64{-133.63, -157.5, -158.62},
		ellipse:   "intl",
		datumName: "Rassadiran",
	},
	"nzgd49": datumDef{
		towgs84:   []float64{59.47, -5.04, 187.44, 0.47, -0.1, 1.024, -4.5993},
		ellipse:   "intl",
		datumName: "New Zealand Geodetic Datum 1949",
	},
	"osgb36": datumDef{
		towgs84:   []float64{446.448, -125.157, 542.060, 0.1502, 0.2470, 0.8421, -20.4894},
		ellipse:   "airy",
		datumName: "Airy 1830",
	},
	"s_jtsk": datumDef{
		towgs84:   []float64{589, 76, 480},
		ellipse:   "bessel",
		datumName: "S-JTSK (Ferro)",
	},
	"beduaram": datumDef{
		towgs84:   []float64{-106, -87, 188},
		ellipse:   "clrk80",
		datumName: "Beduaram",
	},
	"gunung_segara": datumDef{
		towgs84:   []float64{-403, 684, 41},
		ellipse:   "bessel",
		datumName: "Gunung Segara Jakarta",
	},
	"rnb72": datumDef{
		towgs84:   []float64{106.869, -52.2978, 103.724, -0.33657, 0.456955, -1.84218, 1},
		ellipse:   "intl",
		datumName: "Reseau National Belge 1972",
	},
}
