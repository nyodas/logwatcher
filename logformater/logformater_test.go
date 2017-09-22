package logformater

import (
	"bytes"
	"testing"

	"github.com/rcrowley/go-metrics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type MyTestSuite struct {
	suite.Suite
	testRows                 [][]string
	testRowsLarge            [][]string
	testMetricsRegistry      metrics.Registry
	testMetricsRegistryLarge metrics.Registry
}

func (s *MyTestSuite) SetupSuite() {
	s.testRows = [][]string{
		{"/test", "10", "11", "11", "11"},
		{"/test1", "20", "22", "22", "22"},
		{"/test2", "30", "33", "33", "33"},
		{"/test3", "40", "44", "44", "44"},
	}
	s.testRowsLarge = [][]string{
		{"/test", "10", "11", "11", "11"},
		{"/test1", "20", "22", "22", "22"},
		{"/test2", "30", "33", "33", "33"},
		{"/test3", "40", "44", "44", "44"},
		{"/test4", "50", "11", "11", "11"},
		{"/test5", "60", "22", "22", "22"},
		{"/test6", "70", "33", "33", "33"},
		{"/test7", "80", "44", "44", "44"},
		{"/test8", "90", "44", "44", "44"},
		{"/test9", "100", "44", "44", "44"},
		{"/test10", "110", "44", "44", "44"},
		{"/test11", "120", "44", "44", "44"},
	}
	s.testMetricsRegistry = metrics.NewRegistry()
	for idx, tt := range s.testRows {
		metrics.GetOrRegisterMeter(tt[0], s.testMetricsRegistry).Mark(int64(10 * (idx + 1)))
	}
	s.testMetricsRegistryLarge = metrics.NewRegistry()
	for idx, tt := range s.testRowsLarge {
		metrics.GetOrRegisterMeter(tt[0], s.testMetricsRegistryLarge).Mark(int64(10 * (idx + 1)))
	}
}

func TestMySuite(t *testing.T) {
	suite.Run(t, new(MyTestSuite))
}

func (s *MyTestSuite) TestPrintLogs() {
	var fakeStdout bytes.Buffer
	PrintLogs(s.testRows, &fakeStdout)
	assert.NotEmpty(s.T(), fakeStdout.String(), "isEmpty")
	for _, tt := range s.testRows {
		for i := 0; i <= 4; i++ {
			assert.Contains(s.T(), fakeStdout.String(), tt[i], "Check if ouput has corect information")
		}
	}
}

func (s *MyTestSuite) TestPrintLogsEmptyRow() {
	var fakeStdout bytes.Buffer
	PrintLogs([][]string{}, &fakeStdout)
	assert.Empty(s.T(), fakeStdout.String(), "isEmpty")
}

func (s *MyTestSuite) TestGenLogs() {
	rows := GenLogs(s.testMetricsRegistry)
	assert.NotEmpty(s.T(), rows, "isEmpty")
	for idx, tt := range rows {
		assert.Contains(s.T(), tt[0], s.testRows[3-idx][0], "Check if ouput has corect information")
		assert.Contains(s.T(), tt[1], s.testRows[3-idx][1], "Check if ouput has corect information")
	}
}

func (s *MyTestSuite) TestGenLogsMaxRow() {
	rows := GenLogs(s.testMetricsRegistryLarge)
	rowsLen := len(rows)
	assert.NotEmpty(s.T(), rows, "isEmpty")
	assert.Len(s.T(), rowsLen, 10, "Max Len should be 10")
	for idx, tt := range rows {
		rowIdx := rowsLen + 1 // The sort + drop need a +1 shift
		assert.Contains(s.T(), tt[0], s.testRowsLarge[rowIdx-idx][0], "Check if ouput has corect information")
		assert.Contains(s.T(), tt[1], s.testRowsLarge[rowIdx-idx][1], "Check if ouput has corect information")
	}
}

func (s *MyTestSuite) TestGenLogsTotal() {
	metrics.GetOrRegisterMeter("TOTAL", s.testMetricsRegistry).Mark(10)
	rows := GenLogs(s.testMetricsRegistry)
	assert.NotEmpty(s.T(), rows, "isEmpty")
	assert.Contains(s.T(), rows[0], "TOTAL", "Total should be on top")
}
