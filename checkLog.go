package main

import (
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/infrasonar/go-libagent"
)

type LogHelper struct {
	layout string
	lsz    int
	fn     string
	sz     int
}

var lh *LogHelper

func InitLogHelper() {
	layout := os.Getenv("LOG_DATE_FMT")
	if layout == "" {
		layout = "01/02/2006 15:04:05.999999"
	}
	lsz := len(layout)

	_, err := time.Parse(layout, layout)
	if err != nil {
		log.Fatal(err)
	}

	fn := os.Getenv("LOG_FILE_PATH")
	if fn == "" {
		fn = "/opt/QStar/log/syslog"
	}

	szs := os.Getenv("LOG_BUF_SIZE")
	if szs == "" {
		szs = "8192"
	}

	sz, err := strconv.Atoi(szs)
	if err != nil {
		log.Fatal(err)
	} else if sz <= 0 {
		log.Fatal("Invalid LOG_BUF_SIZE")
	}

	lh = &LogHelper{
		layout: layout,
		lsz:    lsz,
		fn:     fn,
		sz:     sz,
	}
}

func CheckLog(_ *libagent.Check) (map[string][]map[string]any, error) {
	state := map[string][]map[string]any{}

	file, err := os.Open(lh.fn)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	buf := make([]byte, lh.sz)
	stat, statErr := file.Stat()
	if statErr != nil {
		panic(statErr)
	}
	start := stat.Size() - int64(lh.sz)
	if start < 0 {
		start = 0
	}

	_, err = file.ReadAt(buf, start)
	if err != nil {
		return nil, err
	}

	var parseError error
	var item map[string]any = nil
	prevName := ""
	items := []map[string]any{}

	logStr := string(buf[:])
	lines := strings.Split(strings.ReplaceAll(logStr, "\r\n", "\n"), "\n")

	for i, line := range lines {
		if (start != 0 && i == 0) || len(line) <= lh.lsz {
			continue // Ignore fist line if start is not the begin
		}

		dtstr := line[:lh.lsz]

		dt, err := time.Parse(lh.layout, dtstr)
		if err != nil {
			if item == nil {
				if parseError == nil {
					log.Printf("Failed to read date from line %v (line: %v, layout: %v)\n", i, dtstr, lh.layout)
				}
				parseError = err
			} else {
				item["message"] = item["message"].(string) + "\n" + strings.TrimSpace(line)
			}
			continue // Ignore lines with errors
		}

		message := strings.TrimSpace(line[lh.lsz:])

		name := strconv.FormatInt(dt.UnixNano(), 10)
		timestamp := float64(dt.UnixMilli()) / 1000.0

		if name == prevName {
			continue // Duplicate name
		}
		prevName = name
		item = map[string]any{
			"name":      name,
			"timestamp": libagent.IFloat64(timestamp),
			"datestr":   dtstr,
			"message":   message,
		}

		items = append(items, item)
	}

	if len(items) > 0 {
		parseError = nil // if we have at least one line parsed, we have no error
	}

	state["log"] = items

	// Print debug dump
	// b, _ := json.MarshalIndent(state, "", "    ")
	// log.Fatal(string(b))

	return state, parseError
}
