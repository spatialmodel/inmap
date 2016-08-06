package geojson

type Geometry struct {
	Type        string      `json:"type"`
	Coordinates interface{} `json:"coordinates"`
}

type InvalidGeometryError struct{}

func (e InvalidGeometryError) Error() string {
	return "geojson: invalid geometry"
}

type UnsupportedGeometryError struct {
	Type string
}

func (e UnsupportedGeometryError) Error() string {
	return "geojson: unsupported geometry type " + e.Type
}
