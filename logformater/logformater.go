package logformater

import (
	"fmt"
	"github.com/olekukonko/tablewriter"
	"github.com/rcrowley/go-metrics"
	"io"
	"sort"
	"strconv"
)

var headersTable = []string{"Section", "Count", "Rate 1m", "Rate 5m", "Rate 15m"}

// Formating metrics to get more information to ouput logs
func GenLogs(logMetrics metrics.Registry) (metricsRows [][]string) {
	type kv struct {
		Key   string
		Value interface{}
	}
	var ss []kv
	var totalMeter kv
	logMetrics.Each(func(k string, v interface{}) {
		if k == "TOTAL" {
			return
		}
		ss = append(ss, kv{k, v})
	})

	// Carefull w/h this it could lead to issue if metrics are not Meter. (Reflect might be a solution.)
	sort.Slice(ss, func(i, j int) bool {
		return ss[i].Value.(metrics.Meter).Count() > ss[j].Value.(metrics.Meter).Count()
	})

	if logMetrics.Get("TOTAL") != nil {
		totalMeter = kv{
			Key:   "TOTAL",
			Value: logMetrics.Get("TOTAL"),
		}
		ss = append([]kv{totalMeter}, ss...)
	}
	for idx, kv := range ss {
		if idx >= 10 {
			break
		}
		var metricsRow = []string{
			kv.Key,
			strconv.Itoa(int(kv.Value.(metrics.Meter).Count())),
			fmt.Sprintf("%.2f", kv.Value.(metrics.Meter).Rate1()),
			fmt.Sprintf("%.2f", kv.Value.(metrics.Meter).Rate5()),
			fmt.Sprintf("%.2f", kv.Value.(metrics.Meter).Rate15()),
		}
		metricsRows = append(metricsRows, metricsRow)
	}
	return metricsRows
}

func PrintLogs(rows [][]string, output io.Writer) {
	if len(rows) < 1 {
		return
	}
	table := tablewriter.NewWriter(output)
	table.SetHeader(headersTable)
	table.SetBorder(false)
	table.AppendBulk(rows)
	table.Render() // Send output
}
