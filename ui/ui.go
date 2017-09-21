package ui

import (
	"github.com/gizak/termui"
	"github.com/gizak/termui/extra"
	"github.com/nyodas/logwatcher/alert"
	"strings"
	"time"
)

// TODO: This is an overall dump of the ui , without error checking and proper use of the function. Need rework
type Ui struct {
	MetricsTableHeaders []string
	AlertsUI            AlertsUi
	MetricsUI           MetricsUi
	Tabpanes            tabpane
}

type AlertsUi struct {
	body                 *termui.Grid
	alertsHistoryTextBox *termui.Par
}
type MetricsUi struct {
	body               *termui.Grid
	metricsTable       *termui.Table
	shortAlertsTextBox *termui.Par
}
type tabpane struct {
	header *termui.Par
	panes  *extra.Tabpane
}

func Init() Ui {
	ui := Ui{
		MetricsTableHeaders: []string{"Section", "Count", "Rate 1m", "Rate 5m", "Rate 15m"},
	}
	err := termui.Init()
	if err != nil {
		panic(err)
	}
	return ui
}

func (ui *Ui) PrepareMetricsUi() *termui.Grid {
	metricsUitable := termui.NewTable()
	metricsUitable.Rows = [][]string{
		ui.MetricsTableHeaders,
	}
	metricsUitable.BorderLabel = "Metrics"
	metricsUitable.TextAlign = termui.AlignCenter
	ui.MetricsUI.metricsTable = metricsUitable
	alertTextBox := termui.NewPar("")
	alertTextBox.BorderLabel = "Alerts"
	alertTextBox.Align()
	alertTextBox.Height = 8
	alertTextBox.BorderFg = termui.ColorYellow
	ui.MetricsUI.shortAlertsTextBox = alertTextBox

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
	ui.MetricsUI = MetricsUi{
		body:               metricsUi,
		metricsTable:       metricsUitable,
		shortAlertsTextBox: alertTextBox,
	}
	return metricsUi
}

func (ui *Ui) PrepareAlertsUi() *termui.Grid {
	alertHistoryTextBox := termui.NewPar("")
	alertHistoryTextBox.BorderLabel = "Alert History"
	alertHistoryTextBox.Align()
	alertHistoryTextBox.Height = termui.TermHeight()
	alertHistoryTextBox.Width = termui.TermWidth()
	alertHistoryTextBox.BorderFg = termui.ColorYellow
	alertsUi := termui.NewGrid()
	alertsUi.Width = termui.TermWidth()
	alertsUi.AddRows(
		termui.NewRow(
			termui.NewCol(12, 0, alertHistoryTextBox),
		),
	)
	alertsUi.Align()
	ui.AlertsUI = AlertsUi{
		body:                 alertsUi,
		alertsHistoryTextBox: alertHistoryTextBox,
	}
	return alertsUi
}

func (ui *Ui) PrepareTabs() *extra.Tabpane {
	header := termui.NewPar("Press q to quit, Press j or k to switch tabs")
	header.Height = 1
	header.Width = 50
	header.Border = false
	header.TextBgColor = termui.ColorBlue

	tab1 := extra.NewTab("Metrics")
	tab1.AddBlocks(ui.MetricsUI.body)
	tab2 := extra.NewTab("Alert History")
	tab2.AddBlocks(ui.AlertsUI.body)

	panes := extra.NewTabpane()
	panes.Y = 1
	panes.Width = 60
	panes.Border = true
	panes.SetTabs(*tab1, *tab2)
	termui.Render(header, panes)
	ui.Tabpanes = tabpane{
		header: header,
		panes:  panes,
	}
	return panes
}

func (ui *Ui) UpdateMetricsUI(metricsTimedTable [][]string) {
	ui.MetricsUI.metricsTable.Rows = metricsTimedTable
	ui.MetricsUI.metricsTable.FgColors = make([]termui.Attribute, len(metricsTimedTable))
	ui.MetricsUI.metricsTable.BgColors = make([]termui.Attribute, len(metricsTimedTable))
	ui.MetricsUI.metricsTable.Align()
	ui.MetricsUI.metricsTable.Analysis()
	ui.MetricsUI.metricsTable.SetSize()
	ui.MetricsUI.metricsTable.Border = true
	ui.MetricsUI.metricsTable.Align()
	ui.MetricsUI.body.Align()
	termui.Clear()
	termui.Render(ui.Tabpanes.header, ui.Tabpanes.panes)
}

func (ui *Ui) UpdateMetricsAlert(al *alert.Alert) {
	msgArray := strings.Split(ui.MetricsUI.shortAlertsTextBox.Text, "\n")
	if len(msgArray) > 3 {
		ui.MetricsUI.shortAlertsTextBox.Text = strings.Join(msgArray[:3], "\n")
	}
	ui.MetricsUI.shortAlertsTextBox.Text = al.String() + "\n" + ui.MetricsUI.shortAlertsTextBox.Text
	termui.Clear()
	ui.MetricsUI.body.Align()
	termui.Render(ui.Tabpanes.header, ui.Tabpanes.panes)
}

func (ui *Ui) UpdateAlertHistory(al *alert.Alert) {
	msgHistoryArray := strings.Split(ui.AlertsUI.alertsHistoryTextBox.Text, "\n")
	if len(msgHistoryArray) > 50 {
		ui.AlertsUI.alertsHistoryTextBox.Text = strings.Join(msgHistoryArray[:3], "\n")
	}
	ui.AlertsUI.alertsHistoryTextBox.Text = al.String() + "\n" + ui.AlertsUI.alertsHistoryTextBox.Text
	termui.Clear()
	termui.Render(ui.Tabpanes.header, ui.Tabpanes.panes)
}

func (ui *Ui) PrepareTimer(statsInterval time.Duration) {
	// Create termui refresh timer (Stats are created a each refresh)
	termui.Merge("timer", termui.NewTimerCh(statsInterval))
}

func (ui *Ui) PrepareKeyHandler() {
	termui.Handle("/sys/kbd/q", func(termui.Event) {
		termui.StopLoop()
		termui.Close()
	})
	termui.Handle("/sys/kbd/C-c", func(termui.Event) {
		termui.StopLoop()
		termui.Close()
	})
	termui.Handle("/sys/kbd/j", func(termui.Event) {
		ui.Tabpanes.panes.SetActiveLeft()
		ui.MetricsUI.body.Align()
		termui.Clear()
		termui.Render(ui.Tabpanes.header, ui.Tabpanes.panes)
	})
	termui.Handle("/sys/kbd/k", func(termui.Event) {
		ui.Tabpanes.panes.SetActiveRight()
		termui.Clear()
		termui.Render(ui.Tabpanes.header, ui.Tabpanes.panes)
	})
}
