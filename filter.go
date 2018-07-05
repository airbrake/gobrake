package gobrake

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
)

func NewBlacklistKeysFilter(keys ...interface{}) func(*Notice) *Notice {
	return func(notice *Notice) *Notice {
		for _, key := range keys {
			notice.Env = filterByKey(notice.Env, key)
			notice.Context = filterByKey(notice.Context, key)
			notice.Session = filterByKey(notice.Session, key)
		}

		return notice
	}
}

func filterByKey(values map[string]interface{}, key interface{}) map[string]interface{} {
	const filtered = "[Filtered]"

	switch key := key.(type) {
	case string:
		for k := range values {
			if k == key {
				values[k] = filtered
			}
		}
	case *regexp.Regexp:
		for k := range values {
			if key.MatchString(k) {
				values[k] = filtered
			}
		}
	default:
		panic(fmt.Errorf("unsupported blacklist key type: %T", key))
	}

	return values
}

func gopathFilter(notice *Notice) *Notice {
	s, ok := notice.Context["gopath"].(string)
	if !ok {
		return notice
	}

	dirs := filepath.SplitList(s)
	for i := range notice.Errors {
		backtrace := notice.Errors[i].Backtrace
		for j := range backtrace {
			frame := &backtrace[j]

			for _, dir := range dirs {
				dir = filepath.Join(dir, "src")
				if strings.HasPrefix(frame.File, dir) {
					frame.File = strings.Replace(frame.File, dir, "/GOPATH", 1)
					break
				}
			}
		}
	}

	return notice
}

func gitRevisionFilter(notice *Notice) *Notice {
	rootDir, _ := notice.Context["rootDirectory"].(string)
	rev, _ := notice.Context["revision"].(string)
	if rootDir == "" || rev != "" {
		return notice
	}

	rev, err := gitRevision(rootDir)
	if err != nil {
		return notice
	}

	notice.Context["revision"] = rev
	return notice
}

var (
	mu        sync.RWMutex
	revisions = make(map[string]interface{})
)

func gitRevision(dir string) (string, error) {
	mu.RLock()
	v := revisions[dir]
	mu.RUnlock()

	switch v := v.(type) {
	case error:
		return "", v
	case string:
		return v, nil
	}

	mu.Lock()
	defer mu.Unlock()

	rev, err := _gitRevision(dir)
	if err != nil {
		logger.Printf("gitRevision dir=%q failed: %s", dir, err)
		revisions[dir] = err
		return "", err
	}

	revisions[dir] = rev
	return rev, nil
}

func _gitRevision(dir string) (string, error) {
	head, err := gitHead(dir)
	if err != nil {
		return "", err
	}

	prefix := []byte("ref: ")
	if !bytes.HasPrefix(head, prefix) {
		return string(head), nil
	}
	head = head[len(prefix):]

	refFile := filepath.Join(dir, ".git", string(head))
	rev, err := ioutil.ReadFile(refFile)
	if err == nil {
		return string(trimnl(rev)), nil
	}

	refsFile := filepath.Join(dir, ".git", "packed-refs")
	fd, err := os.Open(refsFile)
	if err != nil {
		return "", err
	}

	scanner := bufio.NewScanner(fd)
	for scanner.Scan() {
		b := scanner.Bytes()
		if len(b) == 0 || b[0] == '#' || b[0] == '^' {
			continue
		}

		bs := bytes.Split(b, []byte{' '})
		if len(bs) != 2 {
			continue
		}

		if bytes.Equal(bs[1], head) {
			return string(bs[0]), nil
		}
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}

	return "", fmt.Errorf("git revision for ref=%q not found", head)
}

func gitHead(dir string) ([]byte, error) {
	headFile := filepath.Join(dir, ".git", "HEAD")
	b, err := ioutil.ReadFile(headFile)
	if err != nil {
		return nil, err
	}
	return trimnl(b), nil
}

func trimnl(b []byte) []byte {
	for _, c := range []byte{'\n', '\r'} {
		if len(b) > 0 && b[len(b)-1] == c {
			b = b[:len(b)-1]
		} else {
			break
		}
	}
	return b
}
