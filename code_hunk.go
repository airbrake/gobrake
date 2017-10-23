package gobrake

import (
	"bufio"
	"os"
	"strconv"

	"github.com/airbrake/gobrake/lrucache"
)

var cache = lrucache.New(1000)

func getCode(file string, line int) (map[int]string, error) {
	cacheKey := file + strconv.Itoa(line)

	lines, ok := cache.Get(cacheKey)
	if ok {
		return lines, nil
	}

	lines, err := _getCode(file, line)
	if err != nil {
		return nil, err
	}

	cache.Set(cacheKey, lines)
	return lines, nil
}

func _getCode(file string, line int) (map[int]string, error) {
	const nlines = 2

	fd, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer fd.Close()

	start := line - nlines
	end := line + nlines
	scanner := bufio.NewScanner(fd)

	var i int
	lines := make(map[int]string, nlines)
	for scanner.Scan() {
		i++
		if i < start {
			continue
		}
		if i > end {
			break
		}
		lines[i] = scanner.Text()
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return lines, nil
}
