package gobrake

import (
	"runtime"
	"strings"
)

func stackFilter(file string, line int, packageName, funcName string) bool {
	return packageName == "runtime" && funcName == "panic"
}

type stackEntry struct {
	File string `xml:"file,attr" json:"file"`
	Line int    `xml:"number,attr" json:"line"`
	Func string `xml:"method,attr" json:"function"`
}

func stack(skip int, filter func(string, int, string, string) bool) []*stackEntry {
	stack := make([]*stackEntry, 0, 10)
	for i := skip; ; i++ {
		pc, file, line, ok := runtime.Caller(i)
		if !ok {
			break
		}
		packageName, funcName := packageFuncName(pc)
		if filter(file, line, packageName, funcName) {
			stack = stack[:0]
			continue
		}
		stack = append(stack, &stackEntry{
			File: file,
			Line: line,
			Func: funcName,
		})
	}

	return stack
}

func packageFuncName(pc uintptr) (string, string) {
	f := runtime.FuncForPC(pc)
	if f == nil {
		return "???", "???"
	}

	packageName := ""
	funcName := f.Name()

	if ind := strings.LastIndex(funcName, "/"); ind > 0 {
		packageName += funcName[:ind+1]
		funcName = funcName[ind+1:]
	}
	if ind := strings.Index(funcName, "."); ind > 0 {
		packageName += funcName[:ind]
		funcName = funcName[ind+1:]
	}

	return packageName, funcName
}

func scheme(isSecure bool) string {
	if isSecure {
		return "https:"
	}
	return "http:"
}
