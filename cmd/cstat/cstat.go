// cstat records CPU busy states. Similar to iostat, but with greater precision.
package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/tstromberg/cstat/pkg/cstat"
)

var duration = flag.Duration("for", 365*24*time.Hour, "How long to poll until exiting")
var poll = flag.Duration("poll", 1*time.Second, "How often to poll")
var showHeader = flag.Bool("header", true, "show header")
var justBusy = flag.Bool("busy", false, "just show busy score")
var showTotal = flag.Bool("total", true, "show total at end")

func main() {
	flag.Parse()

	header()

	runner := cstat.NewRunner(*duration, *poll)
	var (
		lastResult *cstat.Result
		wg         sync.WaitGroup
	)
	wg.Add(1)
	go func() {
		defer wg.Done()
		lastResult = runner.Run()
	}()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	var terminated bool
	for !terminated {
		select {
		case <-sigs:
			runner.Stop()
		case result, ok := <-runner.C():
			if !ok {
				terminated = true
				break
			}
			display(result)
		}
	}
	wg.Wait()
	total(lastResult)
}

func header() {
	if *showHeader {
		fmt.Printf("elapsed\tbusy%%\tsys%%\tuser%%\tnice%%\tidle%%\n")
	}
}

func display(result *cstat.Result) {

	if *justBusy {
		fmt.Printf("%.3f\n", result.Busy*100)
	} else {
		fmt.Printf("%d\t%.3f\t%.3f\t%.3f\t%.3f\t%.3f\n",
			int64(result.Elapsed.Milliseconds())/1000,
			result.Busy*100,
			result.System*100,
			result.User*100,
			result.Nice*100,
			result.Idle*100,
		)
	}
}

func total(result *cstat.Result) {
	if *showTotal {
		fmt.Printf("\n\nmeasured average over %s\n", result.Elapsed)
		header()
		display(result)
	}
}
