package gobrake

import (
	"fmt"
	"runtime"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

func stackFilter(packageName, funcName string, file string, line int) bool {
	return packageName == "runtime" && funcName == "panic"
}

type StackFrame struct {
	File string `json:"file"`
	Line int    `json:"line"`
	Func string `json:"function"`
}

func stack(depth int) []StackFrame {
	stack := []StackFrame{}
	for i := depth; ; i++ {
		pc, file, line, ok := runtime.Caller(i)
		if !ok {
			break
		}
		packageName, funcName := packageFuncName(pc)
		if stackFilter(packageName, funcName, file, line) {
			stack = stack[:0]
			continue
		}
		stack = append(stack, StackFrame{
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
		return "", ""
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

// stackTraces returns the stackTrace of an error.
// It is part of the errors package public interface.
type stackTracer interface {
	StackTrace() errors.StackTrace
}

// getStack returns the stacktrace associated with e. If e is an
// error from the errors package its stacktrace is extracted, otherwise
// the current stacktrace is collected end returned.
func getStack(e interface{}, depth int) []StackFrame {
	if err, ok := e.(stackTracer); ok {
		return stackFromErrorWithStackTrace(err)
	}

	return stack(depth)
}

// stackFromErrorWithStackTrace extracts the stacktrace from e.
func stackFromErrorWithStackTrace(e stackTracer) []StackFrame {
	stackTrace := e.StackTrace()
	frames := make([]StackFrame, len(stackTrace))
	for i, f := range stackTrace {
		line, _ := strconv.ParseInt(fmt.Sprintf("%d", f), 10, 64)
		funcFile := fmt.Sprintf("%+s", f)
		ind := strings.Index(funcFile, "\n\t")
		frames[i] = StackFrame{
			File: funcFile[ind+2:],
			Line: int(line),
			Func: fmt.Sprintf("%n", f),
		}
	}
	return frames
}

// getTypeName returns the type name of e.
func getTypeName(e interface{}) string {
	if err, ok := e.(error); ok {
		e = errors.Cause(err)
	}
	return fmt.Sprintf("%T", e)
}
