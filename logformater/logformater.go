package logformater

import (
	"fmt"
	"github.com/rcrowley/go-metrics"
	"sort"
	"strconv"
)

// Formating metrics to get more information to ouput logs
func GenLogs(logMetrics metrics.Registry) (metricsRows [][]string) {
	type kv struct {
		Key   string
		Value interface{}
	}
	var ss []kv
	logMetrics.Each(func(k string, v interface{}) {
		ss = append(ss, kv{k, v})
	})

	// Carefull w/h this it could lead to issue if metrics are not Meter. (Reflect might be a solution.)
	sort.Slice(ss, func(i, j int) bool {
		return ss[i].Value.(metrics.Meter).Count() > ss[j].Value.(metrics.Meter).Count()
	})

	for idx, kv := range ss {
		if idx > 10 {
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
