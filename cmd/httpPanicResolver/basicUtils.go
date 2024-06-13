package main

import (
	"net/url"
	"strings"
)

// returns a string by parsing the string to a link
func makeLinks(trace string) string {
	list := strings.Split(trace, "\n")
	for li, line := range list {
		if len(line) == 0 || line[0] != '\t' {
			continue
		}
		file := ""
		for i, ch := range line {
			if ch == ':' {
				file = line[1:i]
				break
			}
		}
		var lineNumber strings.Builder
		for i := len(file) + 2; i < len(line); i++ {
			if line[i] < '0' || line[i] > '9' {
				break
			}
			lineNumber.WriteByte(line[i])
		}
		v := url.Values{}
		v.Set("source", file)
		v.Set("line", lineNumber.String())
		list[li] = "\t <a href=\"/debug/?" + v.Encode() + "\">" + file + ":" + lineNumber.String() + "</a>" + line[len(file)+2+len(lineNumber.String()):]
	}
	return strings.Join(list, "\n")
}
