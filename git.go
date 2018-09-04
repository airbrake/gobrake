package gobrake

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	revisionsMu sync.RWMutex
	revisions   = make(map[string]interface{})
)

func gitRevision(dir string) (string, error) {
	revisionsMu.RLock()
	v := revisions[dir]
	revisionsMu.RUnlock()

	switch v := v.(type) {
	case error:
		return "", v
	case string:
		return v, nil
	}

	revisionsMu.Lock()
	defer revisionsMu.Unlock()

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

var (
	gitCheckoutsMu sync.RWMutex
	gitCheckouts   = make(map[string]interface{})
)

func gitLastCheckout(dir string) (*logInfo, error) {
	gitCheckoutsMu.RLock()
	v := gitCheckouts[dir]
	gitCheckoutsMu.RUnlock()

	switch v := v.(type) {
	case error:
		return nil, v
	case *logInfo:
		return v, nil
	}

	gitCheckoutsMu.Lock()
	defer gitCheckoutsMu.Unlock()

	info, err := _gitLastCheckout(dir)
	if err != nil {
		logger.Printf("gitCommit dir=%q failed: %s", dir, err)
		gitCheckouts[dir] = err
		return nil, err
	}

	gitCheckouts[dir] = info
	return info, nil
}

type logInfo struct {
	Username string    `json:"username"`
	Email    string    `json:"email"`
	Revision string    `json:"revision"`
	Time     time.Time `json:"time"`
}

func _gitLastCheckout(dir string) (*logInfo, error) {
	headFile := filepath.Join(dir, ".git", "logs", "HEAD")
	line, err := lastCheckoutLine(headFile)
	if err != nil {
		return nil, err
	}

	ind := strings.IndexByte(line, '\t')
	if ind == -1 {
		return nil, fmt.Errorf("gitLastCheckout: tab not found")
	}
	line = line[:ind]

	parts := strings.Split(line, " ")
	if len(parts) < 5 {
		return nil, fmt.Errorf("gitLastCheckout: can't parse %q", line)
	}
	author := parts[2 : len(parts)-2]

	utime, err := strconv.ParseInt(parts[len(parts)-2], 10, 64)
	if err != nil {
		return nil, err
	}

	info := &logInfo{
		Revision: parts[1],
		Time:     time.Unix(utime, 0),
	}
	if email := cleanEmail(author[len(author)-1]); email != "" {
		info.Email = email
		author = author[:len(author)-1]
	}
	info.Username = strings.Join(author, " ")

	return info, nil
}

func lastCheckoutLine(filename string) (string, error) {
	fd, err := os.Open(filename)
	if err != nil {
		return "", err
	}

	scanner := bufio.NewScanner(fd)
	var lastCheckout string
	for scanner.Scan() {
		s := scanner.Text()
		if strings.Contains(s, "\tclone: ") ||
			strings.Contains(s, "\tpull: ") ||
			strings.Contains(s, "\tcheckout: ") {
			lastCheckout = s
		}
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}

	if lastCheckout == "" {
		return "", fmt.Errorf("gitLastCheckout: no entries")
	}
	return lastCheckout, nil
}

func cleanEmail(s string) string {
	if s == "" {
		return ""
	}
	if s[0] == '<' && s[len(s)-1] == '>' {
		return s[1 : len(s)-1]
	}
	return ""
}
