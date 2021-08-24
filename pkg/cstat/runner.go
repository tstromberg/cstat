// package cstat records CPU busy states. Similar to iostat, but with greater
// precision.
package cstat

import (
	"context"
	"time"

	"github.com/shirou/gopsutil/cpu"
)

// Result is a structured result from a single poll.
type Result struct {
	Elapsed                               time.Duration
	Busy, System, User, Nice, Idle, Total float64
	LastSample                            time.Time
}

// Runner is responsible for running the statistic polls.
type Runner struct {
	poll time.Duration
	c    chan *Result
}

// NewRunner creates a new Runner that polls for CPU statistics periodically
// for a given duration.
func NewRunner(poll time.Duration) *Runner {
	return &Runner{
		poll: poll,
		c:    make(chan *Result),
	}
}

// C returns the channel that the Runner sends intermediate poll Results to.
func (r *Runner) C() <-chan *Result {
	return r.c
}

// Run starts the poll cycle until Stop is called. Results are send to r.c
// whenever available. The caller is responsible for consuming the Results
//  from r.c once they're available.
//
// It returns the total Result from the very start.
func (r *Runner) Run(ctx context.Context) *Result {
	defer close(r.c)
	start := time.Now()
	lastSample := start
	sst, err := cpu.Times(false)
	if err != nil {
		panic(err)
	}

	pst := sst

	for {
		select {
		case <-ctx.Done():
			return makeResult(sst, pst, start, lastSample)
		case <-time.After(r.poll):
			st, err := cpu.Times(false)
			if err != nil {
				panic(err)
			}
			lastSample = time.Now()
			r.c <- makeResult(pst, st, start, lastSample)
			pst = st
		}
	}
}

func makeResult(psta []cpu.TimesStat, sta []cpu.TimesStat, start time.Time, last time.Time) *Result {
	pst := psta[0]
	st := sta[0]
	idle := st.Idle - pst.Idle
	total := (st.User + st.Nice + st.System + st.Idle) - (pst.User + pst.Nice + pst.System + pst.Idle)
	busy := total - idle

	return &Result{
		Elapsed: last.Sub(start),
		Busy:    busy / total,
		System:  (st.System - pst.System) / total,
		User:    (st.User - pst.User) / total,
		Nice:    (st.Nice - pst.Nice) / total,
		Idle:    (st.Idle - pst.Idle) / total,
		Total:   total,
	}
}
