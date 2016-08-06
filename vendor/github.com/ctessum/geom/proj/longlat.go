package proj

// LongLat is a longitude-latitude (i.e., no projection) projection.
func LongLat(this *SR) (forward, inverse Transformer, err error) {
	identity := func(x, y float64) (float64, float64, error) {
		return x, y, nil
	}
	forward = identity
	inverse = identity
	return
}

func init() {
	// Register this projection with the corresponding names.
	registerTrans(LongLat, "longlat", "identity")
}
