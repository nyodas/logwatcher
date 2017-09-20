package parser

import (
	"regexp"
)

// TODO: Associate map to a named group
const NCSACommonLogFormat = `(\S+)[[:space:]](\S+)[[:space:]](\S+)[[:space:]]\[(.*?)\][[:space:]]"([A-Z]+?) (/?[0-9A-Za-z_-]+)?(?:/?\S+)? (.*?)"[[:space:]](\d+)[[:space:]](\d+)`

var NCSACommonLogFormatRegexp = regexp.MustCompile(NCSACommonLogFormat)

// Parse the string with the NCSA regexp.
func ParseNCSA(line string) (result_slice [][]string) {
	if len(line) < 1 {
		return
	}
	result_slice = NCSACommonLogFormatRegexp.FindAllStringSubmatch(line, -1)
	return
}
