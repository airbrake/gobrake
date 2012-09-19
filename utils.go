package gobrake

import (
	"runtime"
)

type stackEntry struct {
	File string
	Line int
	Func string
}

func stack(skip int) []stackEntry {
	stack := make([]stackEntry, 0, 10)
	for i := skip; ; i++ {
		pc, file, line, ok := runtime.Caller(i)
		if !ok {
			break
		}
		stack = append(stack, stackEntry{
			File: file,
			Line: line,
			Func: runtime.FuncForPC(pc).Name(),
		})
	}
	return stack
}

func proto(n Notifier) string {
	if n.IsSecure() {
		return "https:"
	}
	return "http:"
}
