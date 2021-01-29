// dstat records device utilization. Similar to iostat, but with greater precision.
package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/shirou/gopsutil/disk"
)

type arrayFlags []string

func (i *arrayFlags) String() string {
    return "my string representation"
}

func (i *arrayFlags) Set(value string) error {
    *i = append(*i, value)
    return nil
}

var devices arrayFlags

var duration = flag.Duration("for", 365*24*time.Hour, "How long to poll until exiting")
var poll = flag.Duration("poll", 1*time.Second, "How often to poll")
var showHeader = flag.Bool("header", true, "show header")
var justUtil = flag.Bool("util", false, "just show utilization")
var showTotal = flag.Bool("total", true, "show total at end")

func main() {
	flag.Var(&devices, "device", "Name of disk")
	flag.Parse()

	if len(devices) == 0 {
		partitions, err := disk.Partitions(false)
		if err != nil {
			panic(err)
		}
		for _, part := range partitions {
			// skip the loop devices
			if part.Fstype == "squashfs" {
				continue
			}
			devices = append(devices, part.Device)
		}
	}

	header()
	start := time.Now()
	lastSample := start
	sst, err := disk.IOCounters(devices...)
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

		st, err := disk.IOCounters(devices...)
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
		fmt.Printf("elapsed\tdevice\tread\twrite\tutil%%\n")
	}
}

func display(psta map[string]disk.IOCountersStat, sta map[string]disk.IOCountersStat, start time.Time, last time.Time) {
	for _, device := range devices {
		displayN(psta, sta, start, last, device)
	}
}

func displayN(psta map[string]disk.IOCountersStat, sta map[string]disk.IOCountersStat, start time.Time, last time.Time, device string) {
	pst := psta[device]
	st := sta[device]

	iotime := st.IoTime-pst.IoTime
	total := last.Sub(start).Milliseconds()

	if *justUtil {
		fmt.Printf("%.3f\n", float64(iotime)/float64(total)*100)
	} else {
		fmt.Printf("%d\t%s\t%.3f\t%.3f\t%.3f\n",
			int64(last.Sub(start).Milliseconds())/1000,
			string(device),
			float64(st.ReadBytes-pst.ReadBytes)/1024,
			float64(st.WriteBytes-pst.WriteBytes)/1024,
			float64(iotime)/float64(total)*100,
		)
	}
}

func total(pst map[string]disk.IOCountersStat, st map[string]disk.IOCountersStat, start time.Time, last time.Time) {
	if *showTotal {
		fmt.Printf("\n\nmeasured average over %s\n", last.Sub(start))
		header()
		display(pst, st, start, last)
	}
}
