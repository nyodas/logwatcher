package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type MyTestSuite struct {
	suite.Suite
	testInput  []string
	testOutput [][]string
}

func (s *MyTestSuite) SetupSuite() {
	s.testInput = []string{
		`127.0.0.1 - - [04/Sep/1337:19:55:06 +0000] "POST /test HTTP/1.1" 200 440 "http://test.foobar.pw/" "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/63.0.3203.0 Safari/537.36"`,
		`127.0.0.1 - - [04/Sep/1337:19:55:06 +0000] "GET /test HTTP/1.1" 302 38 "http://test.foobar.pw/" "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/63.0.3203.0 Safari/537.36"`,
		`127.0.0.1 - - [04/Sep/1337:19:55:06 +0000] "HEAD /test HTTP/1.1" 404 38 "http://test.foobar.pw/" "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/63.0.3203.0 Safari/537.36"`,
		`127.0.0.1 - - [04/Sep/1337:19:55:06 +0000] "PUT /test HTTP/1.1" 500 10463 "http://test.foobar.pw/" "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/63.0.3203.0 Safari/537.36"`,
	}
	s.testOutput = [][]string{
		{"127.0.0.1", "-", "-", "04/Sep/1337:19:55:06 +0000", "POST", "/test", "HTTP/1.1", "200", "440"},
		{"127.0.0.1", "-", "-", "04/Sep/1337:19:55:06 +0000", "GET", "/test", "HTTP/1.1", "302", "38"},
		{"127.0.0.1", "-", "-", "04/Sep/1337:19:55:06 +0000", "HEAD", "/test", "HTTP/1.1", "404", "38"},
		{"127.0.0.1", "-", "-", "04/Sep/1337:19:55:06 +0000", "PUT", "/test", "HTTP/1.1", "500", "10463"},
	}
}

func TestMySuite(t *testing.T) {
	suite.Run(t, new(MyTestSuite))
}

func (s *MyTestSuite) TestParserNCSA() {
	parseResult := ParseNCSA("")
	assert.Empty(s.T(), parseResult, "Should be empty went input is an empty string")

	for idx, tt := range s.testInput {
		parseResult = ParseNCSA(tt)
		assert.EqualValues(s.T(), s.testOutput[idx], parseResult[0][1:], "Parsing result should match")
	}
}
