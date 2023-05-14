package rpc

import (
	"regexp"
	"strings"
)

var reGzip = regexp.MustCompile(`(?:^|,)gzip(?:,|$)`)

func CanGzip(ce string) bool {
	return reGzip.MatchString(ce)
}

func CanGzipFast(ce string) bool {
	pos := strings.Index(ce, "gzip")
	return pos >= 0 &&
		(pos == 0 || ce[pos-1] == ',') &&
		(pos+4 == len(ce) || ce[pos+4] == ',')
}
