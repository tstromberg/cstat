package cstat

import (
	"time"

	"github.com/shirou/gopsutil/cpu"
)

type Result struct {
	Elapsed                               time.Duration
	Busy, System, User, Nice, Idle, Total float64
	LastSample                            time.Time
}

type Runner struct {
	duration, poll time.Duration
	c              chan *Result
	stop           chan struct{}
}

func NewRunner(duration, poll time.Duration) *Runner {
	return &Runner{
		duration: duration,
		poll:     poll,
		c:        make(chan *Result),
		stop:     make(chan struct{}),
	}
}

func (r *Runner) C() <-chan *Result {
	return r.c
}

func (r *Runner) Stop() {
	close(r.stop)
}

func (r *Runner) Run() *Result {
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
		case <-r.stop:
			return makeResult(sst, pst, start, lastSample)
		case <-time.After(r.poll):
			if time.Since(start) > r.duration {
				return makeResult(sst, pst, start, lastSample)
			}

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
