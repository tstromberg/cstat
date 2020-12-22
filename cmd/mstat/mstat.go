// mstat records Memory(+ Swap) usage. Similar to vmstat, but with more information.
package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/shirou/gopsutil/mem"
)

var duration = flag.Duration("for", 365*24*time.Hour, "How long to poll until exiting")
var poll = flag.Duration("poll", 1*time.Second, "How often to poll")
var showHeader = flag.Bool("header", true, "show header")
var justUsed = flag.Bool("used", false, "just show used")
var showSwap = flag.Bool("swap", false, "include swap")

func main() {
	flag.Parse()

	header()
	start := time.Now()
	lastSample := start

	sigs := make(chan os.Signal, 1)
	done := make(chan bool, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigs
		os.Exit(0)
		done <- true
	}()

	for {
		if time.Since(start) > *duration {
			os.Exit(0)
		}
		time.Sleep(*poll)

		st, err := mem.VirtualMemory()
		if err != nil {
			panic(err)
		}
		lastSample = time.Now()
		displayMem(st, start, lastSample)
		if *showSwap {
			sst, err := mem.SwapMemory()
			if err != nil {
				panic(err)
			}
			displaySwap(sst, start, lastSample)
		}
	}
}

func header() {
	if *showHeader {
		fmt.Printf("elapsed\ttotal\tused\tfree\tshared\tbuffers\tcached\tavailable\n")
	}
}

func displayMem(st *mem.VirtualMemoryStat, start time.Time, last time.Time) {
	unit := 1024.0

	if *justUsed {
		fmt.Printf("%.3f\n", st.UsedPercent)
	} else {
		fmt.Printf("%d\t%.0f\t%.0f\t%.0f\t%.0f\t%.0f\t%.0f\t%.0f\n",
			int64(last.Sub(start).Milliseconds())/1000,
			float64(st.Total)/unit,
			float64(st.Used)/unit,
			float64(st.Free)/unit, // Linux specific
			float64(st.Shared)/unit, // Linux specific
			float64(st.Buffers)/unit, // Linux specific
			float64(st.Cached)/unit, // Linux specific
			float64(st.Available)/unit,
		)
	}
}

func displaySwap(st *mem.SwapMemoryStat, start time.Time, last time.Time) {
	unit := 1024.0

	if *justUsed {
		fmt.Printf("%.3f\n", st.UsedPercent)
	} else {
		fmt.Printf("%d\t%.0f\t%.0f\t%.0f\n",
			int64(last.Sub(start).Milliseconds())/1000,
			float64(st.Total)/unit,
			float64(st.Used)/unit,
			float64(st.Free)/unit,
		)
	}
}
