// Copyright 2018 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package schedule presents configured windows in a schedule ordered by labels.
package schedule

import (
	"strings"
	"time"

	"../auklib"
	"../window"
	"github.com/google/logger"
)

// Schedule calculates schedule per label and returns label whose names match the given string(s).
func Schedule(names ...string) ([]window.Schedule, error) {
	var r window.Reader
	m, err := window.Windows(auklib.ConfDir, r)
	if err != nil {
		return nil, err
	}
	if len(names) == 0 {
		names = m.Keys()
	}
	logger.Infof("Aggregating schedule for label(s): %s", strings.Join(names, ", "))
	var out []window.Schedule
	for i := range names {
		schedules := m.AggregateSchedules(names[i])
		if len(schedules) == 0 {
			logger.Errorf("no schedule found for label %q", names[i])
			continue
		}

		// Reset eval variables
		var (
			sched window.Schedule
			now   = time.Now()
		)
		for _, s := range schedules {
			// prefer an open schedule
			if s.IsOpen() {
				sched = s
				break
			}
			// Evaluate the next, closest closed schedule
			if sched.Opens.IsZero() {
				sched = s
				continue
			}
			untilSched := sched.Opens.Sub(now).Seconds()
			untilS := s.Opens.Sub(now).Seconds()
			// New schedule in future, current in the past
			if untilS > 0 && untilSched < 0 {
				sched = s
			}
			// Both schedules in the future, new schedule closer to now
			if untilS >= 0 && untilSched >= 0 && untilS < untilSched {
				sched = s
			}
			// Both schedules in the past, new schedule closer to now
			if untilS < 0 && untilSched < 0 && untilS > untilSched {
				sched = s
			}
		}
		out = append(out, sched)
	}
	return out, nil
}
