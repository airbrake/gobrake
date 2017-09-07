package gobrake

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"

	"github.com/pkg/errors"
)

var defaultContextOnce sync.Once
var defaultContext map[string]interface{}

func getDefaultContext() map[string]interface{} {
	defaultContextOnce.Do(func() {
		defaultContext = map[string]interface{}{
			"notifier": map[string]interface{}{
				"name":    "gobrake",
				"version": "3.1.0",
				"url":     "https://github.com/airbrake/gobrake",
			},

			"language":     runtime.Version(),
			"os":           runtime.GOOS,
			"architecture": runtime.GOARCH,
		}
		if s, err := os.Hostname(); err == nil {
			defaultContext["hostname"] = s
		}
		if s := os.Getenv("GOPATH"); s != "" {
			list := filepath.SplitList(s)
			// TODO: multiple root dirs?
			defaultContext["rootDirectory"] = list[0]
		}
	})
	return defaultContext
}

type Error struct {
	Type      string       `json:"type"`
	Message   string       `json:"message"`
	Backtrace []StackFrame `json:"backtrace"`
}

type Notice struct {
	Id    string // id returned by SendNotice
	Error error  // error returned by SendNotice

	Errors  []Error                `json:"errors"`
	Context map[string]interface{} `json:"context"`
	Env     map[string]interface{} `json:"environment"`
	Session map[string]interface{} `json:"session"`
	Params  map[string]interface{} `json:"params"`
}

func (n *Notice) String() string {
	if len(n.Errors) == 0 {
		return "Notice<no errors>"
	}
	e := n.Errors[0]
	return fmt.Sprintf("Notice<%s: %s>", e.Type, e.Message)
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

// parseFrame parses and errors.Frame and returns a non-nil StackFrame
// if the parsing is successful.
//
// We need to parse a formatted output for the func. name/file name/line no.,
// because Frame does not have public methods to access these values, only
// the output is specified.
func parseFrame(f *errors.Frame) *StackFrame {
	// A stack frame can be formatted several ways:
	//    %+s    func\nfile
	//    %d    source line
	buf := fmt.Sprintf("%+s\n%d", f, f)
	parts := strings.Split(buf, "\n")
	if len(parts) != 3 {
		return nil
	}

	fn := strings.TrimSpace(parts[0])
	file := strings.TrimSpace(parts[1])
	line, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		return nil
	}
	return &StackFrame{file, int(line), fn}
}

// stackFromErrorWithStackTrace extracts the stacktrace from e.
func stackFromErrorWithStackTrace(e stackTracer) []StackFrame {
	var frames []StackFrame
	for _, f := range e.StackTrace() {
		pf := parseFrame(&f)
		if pf != nil {
			frames = append(frames, *pf)
		}
	}
	// Remove the frames of runtime.main and runtime.goexit
	frames = frames[:len(frames)-2]
	return frames
}

// getTypeName returns the type name of e.
func getTypeName(e interface{}) string {
	if err, ok := e.(error); ok {
		e = errors.Cause(err)
	}
	return fmt.Sprintf("%T", e)
}

func NewNotice(e interface{}, req *http.Request, depth int) *Notice {
	notice, ok := e.(*Notice)
	if ok {
		return notice
	}

	backtrace := getStack(e, depth)
	typeName := getTypeName(e)

	notice = &Notice{
		Errors: []Error{{
			Type:      typeName,
			Message:   fmt.Sprint(e),
			Backtrace: backtrace,
		}},
		Context: make(map[string]interface{}),
		Env:     make(map[string]interface{}),
		Session: make(map[string]interface{}),
		Params:  make(map[string]interface{}),
	}

	for k, v := range getDefaultContext() {
		notice.Context[k] = v
	}

	if req == nil {
		return notice
	}

	notice.Context["url"] = req.URL.String()
	notice.Context["httpMethod"] = req.Method
	if ua := req.Header.Get("User-Agent"); ua != "" {
		notice.Context["userAgent"] = ua
	}
	notice.Context["userAddr"] = remoteAddr(req)

	for k, v := range req.Header {
		if len(v) == 1 {
			notice.Env[k] = v[0]
		} else {
			notice.Env[k] = v
		}
	}

	return notice
}

func remoteAddr(req *http.Request) string {
	if s := req.Header.Get("X-Forwarded-For"); s != "" {
		parts := strings.Split(s, ",")
		return parts[0]
	}

	if s := req.Header.Get("X-Real-Ip"); s != "" {
		return s
	}

	parts := strings.Split(req.RemoteAddr, ":")
	return parts[0]
}
