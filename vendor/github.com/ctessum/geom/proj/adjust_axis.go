package proj

import "fmt"

func adjust_axis(crs *SR, denorm bool, point []float64) ([]float64, error) {
	var v float64
	var t int
	for i := 0; i < 3; i++ {
		if denorm && i == 2 && len(point) == 2 {
			continue
		}
		if i == 0 {
			v = point[0]
			t = 0
		} else if i == 1 {
			v = point[1]
			t = 1
		} else {
			v = point[2]
			t = 2
		}
		switch crs.Axis[i] {
		case 'e':
			point[t] = v
			break
		case 'w':
			point[t] = -v
			break
		case 'n':
			point[t] = v
			break
		case 's':
			point[t] = -v
			break
		case 'u':
			if len(point) == 3 {
				point[2] = v
			}
			break
		case 'd':
			if len(point) == 3 {
				point[2] = -v
			}
			break
		default:
			err := fmt.Errorf("in plot.adjust_axis: unknown axis (%v). check "+
				"definition of %s", crs.Axis[i], crs.Name)
			return nil, err
		}
	}
	return point, nil
}
