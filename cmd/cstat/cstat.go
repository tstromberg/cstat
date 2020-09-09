// cstat records CPU busy states. Similar to iostat, but with greater precision.
package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/shirou/gopsutil/cpu"
)

var duration = flag.Duration("for", 365*24*time.Hour, "How long to poll until exiting")
var poll = flag.Duration("poll", 1*time.Second, "How often to poll")
var showHeader = flag.Bool("header", true, "show header")
var justBusy = flag.Bool("busy", false, "just show busy score")
var showTotal = flag.Bool("total", true, "show total at end")

func main() {
	flag.Parse()

	header()
	start := time.Now()
	lastSample := start
	sst, err := cpu.Times(false)
	if err != nil {
		panic(err)
	}

	pst := sst
	sigs := make(chan os.Signal, 1)
	done := make(chan bool, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigs
		total(sst, pst, start, lastSample)
		os.Exit(0)
		done <- true
	}()

	for {
		if time.Since(start) > *duration {
			total(sst, pst, start, lastSample)
			os.Exit(0)
		}
		time.Sleep(*poll)

		st, err := cpu.Times(false)
		if err != nil {
			panic(err)
		}
		lastSample = time.Now()
		display(pst, st, start, lastSample)
		pst = st
	}
}

func header() {
	if *showHeader {
		fmt.Printf("elapsed\tbusy%%\tsys%%\tuser%%\tnice%%\tidle%%\n")
	}
}

func display(psta []cpu.TimesStat, sta []cpu.TimesStat, start time.Time, last time.Time) {
	pst := psta[0]
	st := sta[0]
	idle := st.Idle - pst.Idle
	total := (st.User + st.Nice + st.System + st.Idle) - (pst.User + pst.Nice + pst.System + pst.Idle)
	busy := total - idle

	if *justBusy {
		fmt.Printf("%.3f\n", float64(busy)/float64(total)*100)
	} else {
		fmt.Printf("%d\t%.3f\t%.3f\t%.3f\t%.3f\t%.3f\n",
			int64(last.Sub(start).Milliseconds())/1000,
			float64(busy)/float64(total)*100,
			float64(st.System-pst.System)/float64(total)*100,
			float64(st.User-pst.User)/float64(total)*100,
			float64(st.Nice-pst.Nice)/float64(total)*100,
			float64(st.Idle-pst.Idle)/float64(total)*100,
		)
	}
}

func total(pst []cpu.TimesStat, st []cpu.TimesStat, start time.Time, last time.Time) {
	if *showTotal {
		fmt.Printf("\n\nmeasured average over %s\n", last.Sub(start))
		header()
		display(pst, st, start, last)
	}
}
