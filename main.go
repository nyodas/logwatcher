package main

import (
	"fmt"
	"github.com/gizak/termui"
	"github.com/hpcloud/tail"
	"github.com/nyodas/logwatcher/alert"
	"github.com/nyodas/logwatcher/logformater"
	"github.com/nyodas/logwatcher/parser"
	"github.com/nyodas/logwatcher/ui"
	"github.com/rcrowley/go-metrics"
	"gopkg.in/alecthomas/kingpin.v2"
	"io"
	"k8s.io/kubernetes/pkg/util/wait"
	"os"
	"os/signal"
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
	totalMeter := metrics.GetOrRegisterMeter("TOTAL", metricRegistry)

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt)
	go func() {
		for _ = range sigs {
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
	go func(t chan *tail.Line) {
		for line := range t {
			result_slice := parser.ParseNCSA(line.Text)
			if len(result_slice) < 1 {
				// In case of a bad match continue.
				continue
			}
			// TODO: Do something w/h the timestamp
			sectionMeter := metrics.GetOrRegisterMeter(result_slice[0][6], metricRegistry)
			totalMeter = metrics.GetOrRegisterMeter("TOTAL", metricRegistry)
			sectionMeter.Mark(1)
			totalMeter.Mark(1)
		}
	}(t.Lines)

	if *uiReady {
		ui := ui.Init()
		ui.PrepareMetricsUi()
		ui.PrepareAlertsUi()
		ui.PrepareTabs()
		ui.PrepareTimer(*statsInterval)
		ui.PrepareKeyHandler()

		termui.Handle("/timer/"+statsInterval.String(), func(e termui.Event) {
			rows := logformater.GenLogs(metricRegistry)
			//TODO: Move this in a func
			var metricsTimedTable [][]string
			metricsTimedTable = append(metricsTimedTable, ui.MetricsTableHeaders)
			if len(rows) > 0 {
				metricsTimedTable = append(metricsTimedTable, rows...)
			}
			ui.UpdateMetricsUI(metricsTimedTable)
		})
		go func(alerter alert.Alerter) {
			for al := range alerter.AlertBus {
				ui.UpdateMetricsAlert(al)
				ui.UpdateAlertHistory(al)
			}
		}(alerter)
		termui.Loop()
	} else {
		go alerter.Notify()
		//Loop and Print logs.
		wait.PollInfinite(*statsInterval, func() (bool, error) {
			rows := logformater.GenLogs(metricRegistry)
			logformater.PrintLogs(rows, os.Stdout)
			return false, nil
		})
	}

}
