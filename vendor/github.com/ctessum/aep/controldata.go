package aep

// ControlData holds information about how emissions from this source can be
// controlled.
type ControlData struct {
	// Maximum Available Control Technology Code
	// (6 characters maximum) (optional)
	MACT string

	// Control efficiency percentage (give value of 0-100) (recommended,
	// if left blank, default is 0).
	CEff float64

	// Rule Effectiveness percentage (give value of 0-100) (recommended,
	// if left blank, default is 100)
	REff float64

	// Rule Penetration percentage (give value of 0-100) (optional;
	// if missing will result in default of 100)
	RPen float64
}

func (r *ControlData) setCEff(s string) error {
	if s == "" {
		r.CEff = 0.
		return nil
	}
	var err error
	r.CEff, err = stringToFloat(s)
	return err
}

func (r *ControlData) setREff(s string) error {
	if s == "" {
		r.REff = 100.
		return nil
	}
	var err error
	r.REff, err = stringToFloat(s)
	return err
}

func (r *ControlData) setRPen(s string) error {
	if s == "" {
		r.RPen = 100.
		return nil
	}
	var err error
	r.RPen, err = stringToFloat(s)
	return err
}
