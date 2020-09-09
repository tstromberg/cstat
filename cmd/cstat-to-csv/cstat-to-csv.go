// cstat-to-csv concatenates multiple cstat results into a single CSV file
package main

import (
	"bufio"
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"
)

var (
	headingFlag  = flag.String("heading", "", "header value to use")
	durationFlag = flag.Duration("duration", 0, "only include results which span this duration")

	floatRe    = regexp.MustCompile(`^\d+\.\d+$`)
	durationRe = regexp.MustCompile(`average over ([\w\.]+)$`)
)

// Result is the parsed content of a result file
type Result struct {
	values   []string
	average  string
	duration time.Duration
	src      string
	mtime    time.Time
}

func main() {
	flag.Parse()

	var results []*Result

	for _, f := range flag.Args() {
		r, err := parseResultFile(f)
		if err != nil {
			panic(fmt.Sprintf("parse: %v", err))
		}

		// Incomplete result
		if r.duration == time.Duration(0) {
			continue
		}

		if *durationFlag != time.Duration(0) {
			// too long
			if r.duration > (*durationFlag + 1*time.Second) {
				continue
			}
			// too short
			if r.duration < (*durationFlag - 1*time.Second) {
				continue
			}
		}

		results = append(results, r)
	}

	if err := renderResults(os.Stdout, results); err != nil {
		panic(fmt.Sprintf("display: %v", err))
	}
}

// parseResultFile parses the output of cstat <single column form>
func parseResultFile(path string) (*Result, error) {
	st, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("stat: %w", err)
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open: %w", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	r := &Result{
		src:   path,
		mtime: st.ModTime(),
	}

	durationFound := false
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if floatRe.MatchString(line) {
			if durationFound {
				r.average = line
			} else {
				r.values = append(r.values, line)
			}
			continue
		}

		m := durationRe.FindStringSubmatch(line)
		if m != nil {
			d, err := time.ParseDuration(m[1])
			if err != nil {
				return nil, fmt.Errorf("parse duration: %w", err)
			}
			r.duration = d
			durationFound = true
		}
	}

	return r, scanner.Err()
}

// displayResults outputs to stdout in CSV form
/* mtime | heading | average | heading | results */
func renderResults(w io.Writer, rs []*Result) error {
	sort.Slice(rs, func(i, j int) bool { return rs[i].mtime.Before(rs[j].mtime) })

	records := [][]string{}
	for _, r := range rs {
		record := []string{r.mtime.String(), *headingFlag, r.average, r.duration.String(), *headingFlag}
		record = append(record, r.values...)
		records = append(records, record)
	}
	c := csv.NewWriter(w)
	c.WriteAll(records)
	return c.Error()
}
