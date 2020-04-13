package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/lufia/iostat"
)

var duration = flag.Duration("total", 365*24*time.Hour, "How long to wait until exiting")
var wait = flag.Duration("wait", 1*time.Second, "How often to poll")
var header = flag.Bool("header", true, "show header")
var justBusy = flag.Bool("busy", false, "just show busy score")

func main() {
	flag.Parse()

	if *header {
		fmt.Printf("elapsed\tbusy%%\tsys%%\tuser%%\tnice%%\tidle%%\n")
	}
	start := time.Now()
	pst, err := iostat.ReadCPUStats()
	if err != nil {
		panic(err)
	}

	for {
		if time.Since(start) > *duration {
			os.Exit(0)
		}
		time.Sleep(*wait)
		st, err := iostat.ReadCPUStats()
		if err != nil {
			panic(err)
		}
		t := time.Now()
		idle := st.Idle - pst.Idle
		total := (st.User + st.Nice + st.Sys + st.Idle) - (pst.User + pst.Nice + pst.Sys + pst.Idle)
		busy := total - idle

		if *justBusy {
			fmt.Printf("%.3f\n", float64(busy)/float64(total)*100)
		} else {
			fmt.Printf("%d\t%.3f\t%.3f\t%.3f\t%.3f\t%.3f\n",
				int64(t.Sub(start).Milliseconds())/1000,
				float64(busy)/float64(total)*100,
				float64(st.Sys-pst.Sys)/float64(total)*100,
				float64(st.User-pst.User)/float64(total)*100,
				float64(st.Nice-pst.Nice)/float64(total)*100,
				float64(st.Idle-pst.Idle)/float64(total)*100,
			)
		}
		pst = st
	}
}
