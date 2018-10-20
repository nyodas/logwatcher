package main

import (
	"fmt"

	"context"
	"github.com/go-kit/kit/log"
	"github.com/hpcloud/tail"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/prometheus/promql"
	promtsdb "github.com/prometheus/prometheus/storage/tsdb"
	"github.com/prometheus/tsdb"
	"github.com/prometheus/tsdb/labels"
	"gopkg.in/alecthomas/kingpin.v2"
	"io/ioutil"
	"os"
	"os/signal"
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
	value  float64
	ref    *uint64
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

	l := log.NewLogfmtLogger(log.NewSyncWriter(os.Stderr))
	l = log.With(l, "ts", log.DefaultTimestampUTC, "caller", log.DefaultCaller)

	var st *tsdb.DB
	st, err = tsdb.Open(dir, l, nil, &tsdb.Options{
		WALFlushInterval:  200 * time.Millisecond,
		RetentionDuration: 160 * 24 * 60 * 60 * 1000, // 15 days in milliseconds
		BlockRanges:       tsdb.ExponentialBlockRanges(2*60*60*1000, 5, 3),
	})
	if err != nil {
		exitWithError(err)
	}
	// Handle Signal to quit properly.
	sigs := make(chan os.Signal, 1)

	localStorage := &promtsdb.ReadyStorage{}
	localStorage.Set(st)
	queryEngine := promql.NewEngine(localStorage, &promql.EngineOptions{
		MaxConcurrentQueries: 20,
		Timeout:              time.Second * 2,
		Logger:               l,
	})
	ctx := context.Background()
	queryEngine.NewInstantQuery("up", time.Now())
	st.EnableCompactions()
	signal.Notify(sigs, os.Interrupt)
	go func() {
		for range sigs {
			st.Close()
			os.Exit(0)
		}
	}()

	if t, err := tail.TailFile(*logFile, tail.Config{Follow: true}); err != nil {
		fmt.Println(err)
	} else {
		var s sample
		var prevTs int64
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

			metrics.request.With(prometheus.Labels{"method": result_slice[0][5], "path": result_slice[0][6]}).Inc()

			if prevTs != t.Unix() {
				prevTs = t.Unix()
				var out dto.Metric
				metrics.request.With(prometheus.Labels{"method": result_slice[0][5], "path": result_slice[0][6]}).Write(&out)
				counter := out.GetCounter()

				s = sample{
					labels: labels.Labels{
						labels.Label{Name: "__name__", Value: "log_count"},
						labels.Label{Name: "method", Value: result_slice[0][5]},
						labels.Label{Name: "path", Value: result_slice[0][6]},
					},
					value: *counter.Value,
				}
				go func() {
					writeTsdb(st.Appender(), s, t.Unix())
				}()
				q, err := queryEngine.NewRangeQuery("log_count", t.Add(-1000*time.Second), t.Add(10*time.Second), 10*time.Second)
				if err != nil {
					exitWithError(err)
				}
				res := q.Exec(ctx)
				fmt.Println(res)
				// fmt.Println(st.Querier(timestamp.FromTime(t.Add(-1000*time.Second)), timestamp.FromTime(t.Add(1000*time.Second))).LabelValues("path"))

				// fmt.Println(result_slice[0][5], result_slice[0][6], *counter.Value, t, t.Unix())
			}
			//metrics.bodySize
		}
	}
}

func writeTsdb(app tsdb.Appender, s sample, ts int64) (err error) {
	if s.ref == nil {
		ref, err := app.Add(s.labels, ts, s.value)
		if err != nil {
			panic(err)
		}
		s.ref = &ref
		return err
	}
	if err := app.AddFast(*s.ref, ts, s.value); err != nil {
		//
		if errors.Cause(err) != tsdb.ErrNotFound {
			panic(err)
		}

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
