package aep

import (
	"fmt"
	"strconv"
	"time"
)

// Period specifies the time period of the emissions data.
type Period int

// The Periods are the months of the year, annual, or Cem which is hourly
// continuous emissions monitoring data.
const (
	Jan Period = iota + 1
	Feb
	Mar
	Apr
	May
	Jun
	Jul
	Aug
	Sep
	Oct
	Nov
	Dec
	Annual
	Cem
)

func (p Period) String() string {
	switch p {
	case Jan:
		return "Jan"
	case Feb:
		return "Feb"
	case Mar:
		return "Mar"
	case Apr:
		return "Apr"
	case May:
		return "May"
	case Jun:
		return "Jun"
	case Jul:
		return "Jul"
	case Aug:
		return "Aug"
	case Sep:
		return "Sep"
	case Oct:
		return "Oct"
	case Nov:
		return "Nov"
	case Dec:
		return "Dec"
	case Annual:
		return "Annual"
	default:
		panic(fmt.Sprintf("unknown period: %d", int(p)))
	}
}

// TimeInterval returns the start and the end of the receiver period
// in the given year.
func (p Period) TimeInterval(year string) (start, end time.Time, err error) {
	switch p {
	case Jan, Feb, Mar, Apr, May, Jun, Jul, Aug, Sep, Oct, Nov, Dec:
		start, err = time.Parse("Jan 2006", fmt.Sprintf("%s %s", p, year))
		if err != nil {
			return
		}
		if p == Dec {
			var intYear int64
			intYear, err = strconv.ParseInt(year, 0, 32)
			if err != nil {
				return
			}
			nextYear := fmt.Sprintf("%04d", intYear+1)
			end, err = time.Parse("Jan 2006", fmt.Sprintf("Jan %s", nextYear))
			if err != nil {
				return
			}
		} else {
			end, err = time.Parse("Jan 2006", fmt.Sprintf("%s %s", p+1, year))
			if err != nil {
				return
			}
		}
	case Annual:
		start, err = time.Parse("Jan 2006",
			fmt.Sprintf("Jan %s", year))
		if err != nil {
			return
		}
		var intYear int64
		intYear, err = strconv.ParseInt(year, 0, 32)
		if err != nil {
			return
		}
		nextYear := fmt.Sprintf("%04d", intYear+1)
		end, err = time.Parse("Jan 2006", fmt.Sprintf("Jan %s", nextYear))
		if err != nil {
			return
		}
	default:
		panic(fmt.Sprintf("unknown period: %d", int(p)))
	}
	return
}

// PeriodFromTimeInterval returns the period associated with the given
// time interval.
func PeriodFromTimeInterval(start, end time.Time) (Period, error) {
	const (
		monthHoursLow, monthHoursHigh = 24.0 * 27, 24.0 * 32
		yearHoursLow, yearHoursHigh   = 8700.0, 8800.0
	)
	duration := end.Sub(start).Hours()
	if yearHoursLow <= duration && duration <= yearHoursHigh {
		return Annual, nil
	}
	if !(monthHoursLow <= duration && duration <= monthHoursHigh) {
		return -1, fmt.Errorf("aep: invalid period time interval %v -- %v", start, end)
	}
	// The period is a month.
	return Period(start.Month()), nil
}
