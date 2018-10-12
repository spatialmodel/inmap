package greet

// Unit is a holder for a value with units.
type Unit struct {
	Unit    string  `xml:"unit,attr"`
	Amount  float64 `xml:"amount,attr"`
	Enabled bool    `xml:"enabled,attr"`
}
