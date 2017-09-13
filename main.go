package main

import (
	"fmt"

	"github.com/gizak/termui"
	"github.com/hpcloud/tail"
	logformater "github.com/nyodas/logwatcher/logformater"
	"github.com/rcrowley/go-metrics"
	"gopkg.in/alecthomas/kingpin.v2"
	"io"
	"regexp"
)

var (
	debug         = kingpin.Flag("debug", "Enable debug mode.").Bool()
	statsInterval = kingpin.Flag("interval", "Interval for stats logging in seconds").
			Short('i').
			Default("5s").
			Duration()
	logFile = kingpin.Flag("file", "File to watch").
		Required().
		Short('f').String()
)

const NCSACommonLogFormat = `(\S+)[[:space:]](\S+)[[:space:]](\S+)[[:space:]]\[(.*?)\][[:space:]]"([A-Z]+?) (/?[0-9A-Za-z_-]+)?(?:/?\S+)? (.*?)"[[:space:]](\d+)[[:space:]](\d+)`

var NCSACommonLogFormatRegexp = regexp.MustCompile(NCSACommonLogFormat)

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
	go func() {
		if t, err := tail.TailFile(*logFile, tailConfig); err != nil {
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
				// TODO: Do something w/h the timestamp
				c := metrics.GetOrRegisterMeter(result_slice[0][6], metricRegistry)
				totalMeter := metrics.GetOrRegisterMeter("total", metricRegistry)
				c.Mark(1)
				totalMeter.Mark(1)
			}
		}
	}()

	//TODO: Move this in a function
	var headersTable = []string{"Section", "Count", "Rate 1m", "Rate 5m", "Rate 15m"}
	err := termui.Init()
	if err != nil {
		panic(err)
	}
	defer termui.Close()
	var metricsTable [][]string
	metricsTable = append(metricsTable, headersTable)
	table1 := termui.NewTable()
	table1.Rows = metricsTable
	table1.FgColor = termui.ColorWhite
	table1.BgColor = termui.ColorDefault
	termui.Body.AddRows(
		termui.NewRow(
			termui.NewCol(12, 0, table1),
		),
	)
	// calculate layout
	termui.Body.Align()
	// Create termui refresh timer (Stats are created a each refresh)
	termui.Merge("timer", termui.NewTimerCh(*statsInterval))
	termui.Render(termui.Body)
	termui.Handle("/sys/kbd/q", func(termui.Event) {
		termui.StopLoop()
	})
	termui.Handle("/sys/kbd/C-c", func(termui.Event) {
		termui.StopLoop()
	})
	termui.Handle("/timer/"+statsInterval.String(), func(e termui.Event) {
		rows := logformater.GenLogs(metricRegistry)
		// TODO: Move this in a func
		var metricsTable [][]string
		metricsTable = append(metricsTable, headersTable)
		if len(rows) > 0 {
			metricsTable = append(metricsTable, rows...)
		}
		table1 := termui.NewTable()
		table1.Rows = metricsTable
		table1.FgColor = termui.ColorWhite
		table1.BgColor = termui.ColorDefault
		table1.TextAlign = termui.AlignCenter
		table1.Analysis()
		table1.SetSize()

		table1.Border = true
		uiRows := termui.NewRow(
			termui.NewCol(12, 0, table1),
		)
		termui.Body.Rows[0] = uiRows
		//fmt.Println(metricsTable)
		termui.Clear()
		termui.Render(termui.Body)
	})
	termui.Loop()

}
