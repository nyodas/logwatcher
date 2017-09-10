package main

import (
	"fmt"

	"github.com/hpcloud/tail"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/tsdb"
	"github.com/prometheus/tsdb/labels"
	"gopkg.in/alecthomas/kingpin.v2"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"sync"
	"time"
)

var (
	debug   = kingpin.Flag("debug", "Enable debug mode.").Bool()
	logFile = kingpin.Flag("file", "File to watch").
		Required().
		Short('f').String()
)

type logsMetrics struct {
	request  *prometheus.CounterVec
	latency  *prometheus.HistogramVec
	bodySize prometheus.Histogram
}
type sample struct {
	labels labels.Labels
	value  int64
	ref    *string
}

func newLogsMetrics() *logsMetrics {
	m := &logsMetrics{}

	m.request = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "request_count",
		Help: "Number of request",
	}, []string{"path", "method"})
	m.bodySize = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name: "request_bodysize",
		Help: "Size of the body sent",
	})

	prometheus.MustRegister(
		m.request,
		m.bodySize,
	)
	return m
}

const NCSACommonLogFormat = `(\S+)[[:space:]](\S+)[[:space:]](\S+)[[:space:]]\[(.*?)\][[:space:]]"([A-Z]+?) (/?[0-9A-Za-z_-]+)?(?:/?\S+)? (.*?)"[[:space:]](\d+)[[:space:]](\d+)`

var NCSACommonLogFormatRegexp = regexp.MustCompile(NCSACommonLogFormat)
var bufPool sync.Pool

func main() {
	kingpin.Version("0.0.1")
	kingpin.Parse()
	//http.Handle("/metrics", promhttp.Handler())
	//go http.ListenAndServe(":1999", nil)
	metrics := newLogsMetrics()
	dir, err := ioutil.TempDir("", "tsdb_bench")
	if err != nil {
		exitWithError(err)
	}

	if err := os.RemoveAll(dir); err != nil {
		exitWithError(err)
	}
	if err := os.MkdirAll(dir, 0777); err != nil {
		exitWithError(err)
	}
	dir = filepath.Join(dir, "storage")
	st, err := tsdb.Open(dir, nil, nil, &tsdb.Options{
		WALFlushInterval:  200 * time.Millisecond,
		RetentionDuration: 15 * 24 * 60 * 60 * 1000, // 15 days in milliseconds
		BlockRanges:       tsdb.ExponentialBlockRanges(2*60*60*1000, 5, 3),
	})
	if err != nil {
		exitWithError(err)
	}
	st.EnableCompactions()
	if t, err := tail.TailFile(*logFile, tail.Config{Follow: true}); err != nil {
		fmt.Println(err)
	} else {
		for line := range t.Lines {
			if len(line.Text) < 1 {
				continue
			}
			result_slice := NCSACommonLogFormatRegexp.FindAllStringSubmatch(line.Text, -1)
			if len(result_slice) < 1 {
				// In case of a bad match continue.
				continue
			}
			//fmt.Println(result_slice[0])
			t, err := time.Parse("02/Jan/2006:15:04:05 -0700", result_slice[0][4])
			if err != nil {
				fmt.Println(err)
			}
			s := sample{
				labels: labels.Labels{
					labels.Label{"name", "log_count"},
					labels.Label{"method", result_slice[0][5]},
					labels.Label{"path", result_slice[0][6]},
				},
			}
			writeTsdb(st.Appender(), s, t.Unix())
			metrics.request.With(prometheus.Labels{"method": result_slice[0][5], "path": result_slice[0][6]}).Inc()
			//metrics.bodySize
			fmt.Println(result_slice[0][5], result_slice[0][6], t.Unix())
		}
	}
}

func writeTsdb(app tsdb.Appender, s sample, ts int64) (err error) {
	if s.ref == nil {
		ref, err := app.Add(s.labels, ts, float64(s.value))
		if err != nil {
			panic(err)
		}
		s.ref = &ref
	} else if err := app.AddFast(*s.ref, ts, float64(s.value)); err != nil {

		//if errors.Cause(err) != tsdb.ErrNotFound {
		//	panic(err)
		//}

		ref, err := app.Add(s.labels, ts, float64(s.value))
		if err != nil {
			panic(err)
		}
		s.ref = &ref
	}
	if err := app.Commit(); err != nil {
		return err
	}
	return
}

func exitWithError(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
