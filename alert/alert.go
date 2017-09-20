package alert

import (
	"fmt"
	"github.com/rcrowley/go-metrics"
	"k8s.io/kubernetes/pkg/util/wait"
	"log"
	"time"
)

type alert struct {
	Count   int64
	Section string
	Firing  bool
	Time    time.Time
}

func (al alert) String() string {
	if al.Firing {
		return fmt.Sprintf("[[FIRING]](fg-red) High traffic generated an alert - hits = %d, triggered at %s", al.Count, al.Time)
	}
	return fmt.Sprintf("[[RECOVER]](fg-green) High traffic generated an alert - current hits = %d, recovered at %s", al.Count, al.Time)
}

type Alerter struct {
	previousSnapshot map[string]int64
	ceiling          int64
	alerts           []alert
	AlertBus         chan *alert
}

func NewAlerter(ceiling int64) Alerter {
	alerter := Alerter{
		ceiling:          ceiling,
		previousSnapshot: make(map[string]int64),
		AlertBus:         make(chan *alert),
	}
	return alerter
}

func (a *Alerter) GenAlert(logMetrics metrics.Registry) {
	var rateDiff int64
	currentCount := logMetrics.Get("TOTAL").(metrics.Meter).Count()
	rateDiff = currentCount - a.previousSnapshot["TOTAL"]
	a.previousSnapshot["TOTAL"] = currentCount
	a.Save(rateDiff)
}

func (a *Alerter) Poll(logMetrics metrics.Registry) {
	log.Printf("%s", logMetrics)
	wait.PollInfinite(3*time.Second, func() (bool, error) {
		a.GenAlert(logMetrics)
		return false, nil
	})
}

func (a *Alerter) Save(count int64) {
	// TODO: Add a database for persistence and history.
	// TODO: Use a chan to Notify the alerts state
	var lastStatus bool
	if len(a.alerts) > 0 {
		lastStatus = a.alerts[len(a.alerts)-1].Firing
	}
	firing := count > a.ceiling
	currentAlert := alert{
		Count:  count,
		Firing: firing,
		Time:   time.Now(),
	}
	if !(firing && !lastStatus) && !(!firing && lastStatus) {
		return
	}
	a.alerts = append(a.alerts, currentAlert)
	a.AlertBus <- &currentAlert
}

func (a *Alerter) Notify() {
	for n := range a.AlertBus {
		if n.Firing {
			a.NotifyFiring(*n)
		} else {
			a.NotifyRecover(*n)
		}
	}
}

func (a *Alerter) NotifyFiring(currentAlert alert) {
	log.Printf(currentAlert.String())
}

func (a *Alerter) NotifyRecover(currentAlert alert) {
	log.Printf(currentAlert.String())
}
