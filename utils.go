package gobrake

import (
	"runtime"
)

type stackEntry struct {
	File string `xml:"file,attr" json:"file"`
	Line int    `xml:"number,attr" json:"line"`
	Func string `xml:"method,attr" json:"function"`
}

func stack(skip int) []*stackEntry {
	stack := make([]*stackEntry, 0, 10)
	for i := skip; ; i++ {
		pc, file, line, ok := runtime.Caller(i)
		if !ok {
			break
		}
		stack = append(stack, &stackEntry{
			File: file,
			Line: line,
			Func: runtime.FuncForPC(pc).Name(),
		})
	}
	return stack
}

func scheme(isSecure bool) string {
	if isSecure {
		return "https:"
	}
	return "http:"
}
