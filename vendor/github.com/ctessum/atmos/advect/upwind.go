package advect

// UpwindFlux calculates advective mass transfer across a single grid cell edge
// where umhalf is wind velocity at the negative edge of the cell of interest,
// Cm1 is the concentration in the cell adjecent in
// the negative direction, C is the concentration in the cell of interest,
// Δx is length of the cell of interest along the axis of interest.
// If the two cells are not of the same volume, the resulting flux must be multiplied
// by min((Δym1*Δzm1)/(Δy*Δz),1).
func UpwindFlux(umhalf, Cm1, C, Δx float64) float64 {
	if umhalf > 0 {
		return umhalf * Cm1 / Δx
	}
	return umhalf * C / Δx
}
