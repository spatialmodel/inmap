/*
Copyright (C) 2012-2014 Regents of the University of Minnesota.
This file is part of AEP.

AEP is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

AEP is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with AEP.  If not, see <http://www.gnu.org/licenses/>.
*/

package aep

import "time"

type Outputter interface {
	Output(tp *TemporalProcessor, startTime, endTime time.Time, timeStep time.Duration)
	PlumeRise(gridIndex int, r *ParsedRecord) (kPlume int)
	Kemit() int
}

type outputTimer struct {
	startTime, endTime, currentTime time.Time
	timeStep                        time.Duration
}

func newOutputTimer(startTime, endTime time.Time, timeStep time.Duration) *outputTimer {
	return &outputTimer{
		startTime:   startTime,
		currentTime: startTime,
		endTime:     endTime,
		timeStep:    timeStep,
	}
}

func (o *outputTimer) NextTime() (keepGoing bool) {
	o.currentTime = o.currentTime.Add(o.timeStep)
	keepGoing = o.currentTime.Before(o.endTime)
	return
}
