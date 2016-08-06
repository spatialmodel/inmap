package proj

type ellipsoidDef struct {
	a, b, rf    float64
	ellipseName string
}

var ellipsoidDefs = map[string]ellipsoidDef{
	"MERIT": ellipsoidDef{
		a:           6378137.0,
		rf:          298.257,
		ellipseName: "MERIT 1983",
	},
	"SGS85": ellipsoidDef{
		a:           6378136.0,
		rf:          298.257,
		ellipseName: "Soviet Geodetic System 85",
	},
	"GRS80": ellipsoidDef{
		a:           6378137.0,
		rf:          298.257222101,
		ellipseName: "GRS 1980(IUGG, 1980)",
	},
	"IAU76": ellipsoidDef{
		a:           6378140.0,
		rf:          298.257,
		ellipseName: "IAU 1976",
	},
	"airy": ellipsoidDef{
		a:           6377563.396,
		b:           6356256.910,
		ellipseName: "Airy 1830",
	},
	"APL4": ellipsoidDef{
		a:           6378137,
		rf:          298.25,
		ellipseName: "Appl. Physics. 1965",
	},
	"NWL9D": ellipsoidDef{
		a:           6378145.0,
		rf:          298.25,
		ellipseName: "Naval Weapons Lab., 1965",
	},
	"mod_airy": ellipsoidDef{
		a:           6377340.189,
		b:           6356034.446,
		ellipseName: "Modified Airy",
	},
	"andrae": ellipsoidDef{
		a:           6377104.43,
		rf:          300.0,
		ellipseName: "Andrae 1876 (Den., Iclnd.)",
	},
	"aust_SA": ellipsoidDef{
		a:           6378160.0,
		rf:          298.25,
		ellipseName: "Australian Natl & S. Amer. 1969",
	},
	"GRS67": ellipsoidDef{
		a:           6378160.0,
		rf:          298.2471674270,
		ellipseName: "GRS 67(IUGG 1967)",
	},
	"bessel": ellipsoidDef{
		a:           6377397.155,
		rf:          299.1528128,
		ellipseName: "Bessel 1841",
	},
	"bess_nam": ellipsoidDef{
		a:           6377483.865,
		rf:          299.1528128,
		ellipseName: "Bessel 1841 (Namibia)",
	},
	"clrk66": ellipsoidDef{
		a:           6378206.4,
		b:           6356583.8,
		ellipseName: "Clarke 1866",
	},
	"clrk80": ellipsoidDef{
		a:           6378249.145,
		rf:          293.4663,
		ellipseName: "Clarke 1880 mod.",
	},
	"clrk58": ellipsoidDef{
		a:           6378293.645208759,
		rf:          294.2606763692654,
		ellipseName: "Clarke 1858",
	},
	"CPM": ellipsoidDef{
		a:           6375738.7,
		rf:          334.29,
		ellipseName: "Comm. des Poids et Mesures 1799",
	},
	"delmbr": ellipsoidDef{
		a:           6376428.0,
		rf:          311.5,
		ellipseName: "Delambre 1810 (Belgium)",
	},
	"engelis": ellipsoidDef{
		a:           6378136.05,
		rf:          298.2566,
		ellipseName: "Engelis 1985",
	},
	"evrst30": ellipsoidDef{
		a:           6377276.345,
		rf:          300.8017,
		ellipseName: "Everest 1830",
	},
	"evrst48": ellipsoidDef{
		a:           6377304.063,
		rf:          300.8017,
		ellipseName: "Everest 1948",
	},
	"evrst56": ellipsoidDef{
		a:           6377301.243,
		rf:          300.8017,
		ellipseName: "Everest 1956",
	},
	"evrst69": ellipsoidDef{
		a:           6377295.664,
		rf:          300.8017,
		ellipseName: "Everest 1969",
	},
	"evrstSS": ellipsoidDef{
		a:           6377298.556,
		rf:          300.8017,
		ellipseName: "Everest (Sabah & Sarawak)",
	},
	"fschr60": ellipsoidDef{
		a:           6378166.0,
		rf:          298.3,
		ellipseName: "Fischer (Mercury Datum) 1960",
	},
	"fschr60m": ellipsoidDef{
		a:           6378155.0,
		rf:          298.3,
		ellipseName: "Fischer 1960",
	},
	"fschr68": ellipsoidDef{
		a:           6378150.0,
		rf:          298.3,
		ellipseName: "Fischer 1968",
	},
	"helmert": ellipsoidDef{
		a:           6378200.0,
		rf:          298.3,
		ellipseName: "Helmert 1906",
	},
	"hough": ellipsoidDef{
		a:           6378270.0,
		rf:          297.0,
		ellipseName: "Hough",
	},
	"intl": ellipsoidDef{
		a:           6378388.0,
		rf:          297.0,
		ellipseName: "International 1909 (Hayford)",
	},
	"kaula": ellipsoidDef{
		a:           6378163.0,
		rf:          298.24,
		ellipseName: "Kaula 1961",
	},
	"lerch": ellipsoidDef{
		a:           6378139.0,
		rf:          298.257,
		ellipseName: "Lerch 1979",
	},
	"mprts": ellipsoidDef{
		a:           6397300.0,
		rf:          191.0,
		ellipseName: "Maupertius 1738",
	},
	"new_intl": ellipsoidDef{
		a:           6378157.5,
		b:           6356772.2,
		ellipseName: "New International 1967",
	},
	"plessis": ellipsoidDef{
		a:           6376523.0,
		rf:          6355863.0,
		ellipseName: "Plessis 1817 (France)",
	},
	"krass": ellipsoidDef{
		a:           6378245.0,
		rf:          298.3,
		ellipseName: "Krassovsky, 1942",
	},
	"SEasia": ellipsoidDef{
		a:           6378155.0,
		b:           6356773.3205,
		ellipseName: "Southeast Asia",
	},
	"walbeck": ellipsoidDef{
		a:           6376896.0,
		b:           6355834.8467,
		ellipseName: "Walbeck",
	},
	"WGS60": ellipsoidDef{
		a:           6378165.0,
		rf:          298.3,
		ellipseName: "WGS 60",
	},
	"WGS66": ellipsoidDef{
		a:           6378145.0,
		rf:          298.25,
		ellipseName: "WGS 66",
	},
	"WGS7": ellipsoidDef{
		a:           6378135.0,
		rf:          298.26,
		ellipseName: "WGS 72",
	},
	"WGS84": ellipsoidDef{
		a:           6378137.0,
		rf:          298.257223563,
		ellipseName: "WGS 84",
	},
	"sphere": ellipsoidDef{
		a:           6370997.0,
		b:           6370997.0,
		ellipseName: "Normal Sphere (r=6370997)",
	},
}
