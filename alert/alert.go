package alert

import (
	"fmt"
	"log"
	"time"

	"github.com/rcrowley/go-metrics"
	"k8s.io/apimachinery/pkg/util/wait"
)

type Alert struct {
	Count   int64
	Section string
	Firing  bool
	Time    time.Time
}

type Alerter struct {
	previousSnapshot map[string]int64
	ceiling          int64
	alerts           []Alert
	AlertBus         chan Alert
}

func (al Alert) String() string {
	if al.Firing {
		return fmt.Sprintf("[[FIRING]](fg-red)  - High traffic generated an alert - hits = %d, triggered at %s", al.Count, al.Time)
	}
	return fmt.Sprintf("[[RECOVER]](fg-green) - High traffic generated an alert - current hits = %d, recovered at %s", al.Count, al.Time)
}

func NewAlerter(ceiling int64) Alerter {
	alerter := Alerter{
		ceiling:          ceiling,
		previousSnapshot: make(map[string]int64),
		AlertBus:         make(chan Alert),
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
	wait.PollInfinite(2*time.Minute, func() (bool, error) {
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
	currentAlert := Alert{
		Count:  count,
		Firing: firing,
		Time:   time.Now(),
	}
	if !(firing && !lastStatus) && !(!firing && lastStatus) {
		return
	}
	a.alerts = append(a.alerts, currentAlert)
	a.AlertBus <- currentAlert
}

func (a *Alerter) Notify() {
	for n := range a.AlertBus {
		if n.Firing {
			a.NotifyFiring(n)
		} else {
			a.NotifyRecover(n)
		}
	}
}

func (a *Alerter) NotifyFiring(currentAlert Alert) {
	log.Printf(currentAlert.String())
}

func (a *Alerter) NotifyRecover(currentAlert Alert) {
	log.Printf(currentAlert.String())
}
