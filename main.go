package main

import (
	"fmt"
	"github.com/gizak/termui"
	"github.com/gizak/termui/extra"
	"github.com/hpcloud/tail"
	"github.com/nyodas/logwatcher/alert"
	"github.com/nyodas/logwatcher/logformater"
	"github.com/nyodas/logwatcher/parser"
	"github.com/rcrowley/go-metrics"
	"gopkg.in/alecthomas/kingpin.v2"
	"io"
	"k8s.io/kubernetes/pkg/util/wait"
	"os"
	"strings"
)

var (
	debug         = kingpin.Flag("debug", "Enable debug mode.").Bool()
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
	}
	metricRegistry := metrics.NewRegistry()
	// Preregister total , for alerting purpose.
	totalMeter := metrics.GetOrRegisterMeter("TOTAL", metricRegistry)

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
		var headersTable = []string{"Section", "Count", "Rate 1m", "Rate 5m", "Rate 15m"}
		err = termui.Init()
		if err != nil {
			panic(err)
		}
		defer termui.Close()
		metricsUitable := termui.NewTable()
		metricsUitable.Rows = [][]string{
			headersTable,
		}
		metricsUitable.BorderLabel = "Metrics"
		metricsUitable.TextAlign = termui.AlignCenter

		alertTextBox := termui.NewPar("")
		alertTextBox.BorderLabel = "Alerts"
		alertTextBox.Align()
		alertTextBox.Height = 8
		alertTextBox.BorderFg = termui.ColorYellow

		alertHistoryTextBox := termui.NewPar("")
		alertHistoryTextBox.BorderLabel = "Alert History"
		alertHistoryTextBox.Align()
		alertHistoryTextBox.Height = termui.TermHeight()
		alertHistoryTextBox.Width = termui.TermWidth()
		alertHistoryTextBox.BorderFg = termui.ColorYellow

		metricsUi := termui.NewGrid()
		metricsUi.Width = termui.TermWidth()
		metricsUi.AddRows(
			termui.NewRow(
				termui.NewCol(12, 0, metricsUitable),
			),
			termui.NewRow(
				termui.NewCol(12, 0, alertTextBox),
			),
		)
		metricsUi.Align()
		// calculate layout
		//termui.Body.Align()

		header := termui.NewPar("Press q to quit, Press j or k to switch tabs")
		header.Height = 1
		header.Width = 50
		header.Border = false
		header.TextBgColor = termui.ColorBlue

		tab1 := extra.NewTab("Metrics")
		tab1.AddBlocks(metricsUi)
		tab2 := extra.NewTab("Alert History")
		tab2.AddBlocks(alertHistoryTextBox)

		tabpane := extra.NewTabpane()
		tabpane.Y = 1
		tabpane.Width = 60
		tabpane.Border = true
		tabpane.SetTabs(*tab1, *tab2)
		termui.Render(header, tabpane)

		// Create termui refresh timer (Stats are created a each refresh)
		termui.Merge("timer", termui.NewTimerCh(*statsInterval))
		termui.Render(termui.Body)
		termui.Handle("/sys/kbd/q", func(termui.Event) {
			termui.StopLoop()
		})
		termui.Handle("/sys/kbd/C-c", func(termui.Event) {
			termui.StopLoop()
		})
		termui.Handle("/sys/kbd/j", func(termui.Event) {
			tabpane.SetActiveLeft()
			termui.Clear()
			termui.Render(header, tabpane)
		})
		termui.Handle("/sys/kbd/k", func(termui.Event) {
			tabpane.SetActiveRight()
			termui.Clear()
			termui.Render(header, tabpane)
		})

		termui.Handle("/timer/"+statsInterval.String(), func(e termui.Event) {
			rows := logformater.GenLogs(metricRegistry)
			//TODO: Move this in a func
			var metricsTimedTable [][]string
			metricsTimedTable = append(metricsTimedTable, headersTable)
			if len(rows) > 0 {
				metricsTimedTable = append(metricsTimedTable, rows...)
			}
			metricsUitable.Rows = metricsTimedTable
			metricsUitable.FgColors = make([]termui.Attribute, len(metricsTimedTable))
			metricsUitable.BgColors = make([]termui.Attribute, len(metricsTimedTable))
			metricsUitable.Align()
			metricsUitable.Analysis()
			metricsUitable.SetSize()
			metricsUitable.Border = true
			metricsUi.Align()
			termui.Clear()
			termui.Render(header, tabpane)
			//termui.Render(termui.Body)
		})
		go func(alerter alert.Alerter) {
			for al := range alerter.AlertBus {
				msgArray := strings.Split(alertTextBox.Text, "\n")
				if len(msgArray) > 3 {
					alertTextBox.Text = strings.Join(msgArray[:3], "\n")
				}
				msgHistoryArray := strings.Split(alertHistoryTextBox.Text, "\n")
				if len(msgHistoryArray) > 50 {
					alertHistoryTextBox.Text = strings.Join(msgHistoryArray[:3], "\n")
				}
				alertTextBox.Text = al.String() + "\n" + alertTextBox.Text
				alertHistoryTextBox.Text = al.String() + "\n" + alertHistoryTextBox.Text
				termui.Clear()
				metricsUi.Align()
				termui.Render(header, tabpane)
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
