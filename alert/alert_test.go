package alert

import (
	"testing"

	"sync"

	"github.com/rcrowley/go-metrics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type MyTestSuite struct {
	suite.Suite
	alerter             Alerter
	testMetricsRegistry metrics.Registry
	alertCases          []alertCase
}

type alertCase struct {
	mark   int64
	firing bool
}

func (s *MyTestSuite) SetupSuite() {
	s.testMetricsRegistry = metrics.NewRegistry()
	metrics.GetOrRegisterMeter("TOTAL", s.testMetricsRegistry).Mark(10)
	s.alerter = NewAlerter(10)
	s.alertCases = []alertCase{
		{mark: 9, firing: false},
		{mark: 15, firing: true},
		{mark: 35, firing: true},
		{mark: 3, firing: false},
	}

}

func TestMySuite(t *testing.T) {
	suite.Run(t, new(MyTestSuite))
}

func (s *MyTestSuite) TestNewAlerter() {
	alert := NewAlerter(10)
	assert.ObjectsAreEqual(alert, s.alerter)
	assert.Equal(s.T(), alert.ceiling, int64(10))
}

func (s *MyTestSuite) TestGenAlert() {
	s.alerter.GenAlert(s.testMetricsRegistry)
	assert.Equal(s.T(), s.alerter.previousSnapshot["TOTAL"], int64(10))
	var mark int64 = 10
	wg := sync.WaitGroup{}
	wg.Add(2)
	go func() {
		for n := range s.alerter.AlertBus {
			wg.Done()
			if n.Count > 10 {
				assert.True(s.T(), n.Firing, "Should be firing when count >= 10")
				continue
			}
			assert.False(s.T(), n.Firing, "Should not be firing when count < 10")
		}
	}()
	for _, n := range s.alertCases {
		metrics.GetOrRegisterMeter("TOTAL", s.testMetricsRegistry).Mark(n.mark)
		assert.Equal(s.T(), s.alerter.previousSnapshot["TOTAL"], mark, "Should be the previous count")
		mark += n.mark
		s.alerter.GenAlert(s.testMetricsRegistry)
		if n.mark > s.alerter.ceiling {
			assert.Equal(s.T(), s.alerter.alerts[len(s.alerter.alerts)-1].Firing, n.firing, "Alert should have the proper state")
		}
	}
	// TODO: Add timeout in case of issue in the wait.
	wg.Wait()
}
