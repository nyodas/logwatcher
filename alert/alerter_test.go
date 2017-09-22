package alert

import (
	"testing"

	"github.com/rcrowley/go-metrics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type MyTestSuite struct {
	suite.Suite
	alerter             Alerter
	testMetricsRegistry metrics.Registry
}

func (s *MyTestSuite) SetupSuite() {
	s.testMetricsRegistry = metrics.NewRegistry()
	metrics.GetOrRegisterMeter("TOTAL", s.testMetricsRegistry).Mark(10)
	s.alerter = NewAlerter(10)
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
	mark := 10
	for i := 1; i <= 3; i++ {
		assert.Equal(s.T(), s.alerter.previousSnapshot["TOTAL"], int64(mark), "Should be the previous count")
		metrics.GetOrRegisterMeter("TOTAL", s.testMetricsRegistry).Mark(int64(i * 10))
		mark += i * 10
		s.alerter.GenAlert(s.testMetricsRegistry)
	}
	assert.Equal(s.T(), s.alerter.alerts[len(s.alerter.alerts)-1].Firing, true, "alert should be firing")
	metrics.GetOrRegisterMeter("TOTAL", s.testMetricsRegistry).Mark(int64(1))
	s.alerter.GenAlert(s.testMetricsRegistry)
	assert.Equal(s.T(), s.alerter.alerts[len(s.alerter.alerts)-1].Firing, false, "alert should be recovered")
}
