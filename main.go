package main

import (
	"fmt"
	"io"
	"os"
	"os/signal"

	"github.com/gizak/termui"
	"github.com/hpcloud/tail"
	"github.com/nyodas/logwatcher/alert"
	"github.com/nyodas/logwatcher/logformater"
	"github.com/nyodas/logwatcher/parser"
	"github.com/nyodas/logwatcher/ui"
	"github.com/rcrowley/go-metrics"
	"gopkg.in/alecthomas/kingpin.v2"
	"k8s.io/kubernetes/pkg/util/wait"
)

var (
	statsInterval = kingpin.Flag("interval", "Interval for stats logging in seconds").
			Short('i').
			Default("5s").
			Duration()
	alertCeiling = kingpin.Flag("alert", "Ceiling for the alert.").
			Short('a').
			Default("10").
			Int64()
	logFile = kingpin.Flag("file", "File to watch").
		Required().
		Short('f').
		String()
	uiReady = kingpin.Flag("ui", "File to watch").
		Short('u').
		Bool()
)

func main() {
	kingpin.Version("0.0.1")
	kingpin.Parse()
	tailConfig := tail.Config{
		Follow: true,
		Location: &tail.SeekInfo{
			Offset: 0,
			Whence: io.SeekEnd},
		ReOpen: true,
		Logger: tail.DiscardingLogger,
	}
	metricRegistry := metrics.NewRegistry()
	// Preregister total , for alerting purpose.
	metrics.GetOrRegisterMeter("TOTAL", metricRegistry)

	// Handle Signal to quit properly.
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt)
	go func() {
		for range sigs {
			os.Exit(0)
		}
	}()

	alerter := alert.NewAlerter(*alertCeiling)
	go alerter.Poll(metricRegistry)
	t, err := tail.TailFile(*logFile, tailConfig)
	if err != nil {
		fmt.Println(err)
		return
	}
	go setMetricsFromLines(t.Lines, metricRegistry)

	if *uiReady {
		ui := ui.Init()
		ui.PrepareMetricsUi()
		ui.PrepareAlertsUi()
		ui.PrepareTabs()
		ui.PrepareKeyHandler()
		ui.LaunchMetricsUiTimer(*statsInterval, metricRegistry)
		go ui.AlertsHandler(alerter)
		termui.Loop()
		return
	}

	go alerter.Notify()
	//Infinite Loop every statsInterval and Print logs.
	wait.PollInfinite(*statsInterval, func() (bool, error) {
		rows := logformater.GenLogs(metricRegistry)
		logformater.PrintLogs(rows, os.Stdout)
		return false, nil
	})
}

// Parse each line returned by tail. And add the metrics for every domain encountered
func setMetricsFromLines(t chan *tail.Line, metricRegistry metrics.Registry) {
	for line := range t {
		result_slice := parser.ParseNCSA(line.Text)
		if len(result_slice) < 1 {
			// In case of a bad match continue.
			continue
		}
		// TODO: Do something w/h the timestamp
		// TODO: Add metrics for other things like http code.
		sectionMeter := metrics.GetOrRegisterMeter(result_slice[0][6], metricRegistry)
		totalMeter := metrics.GetOrRegisterMeter("TOTAL", metricRegistry)
		sectionMeter.Mark(1)
		totalMeter.Mark(1)
	}
}
