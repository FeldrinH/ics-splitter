package handler

import (
	"fmt"
	//"os"
	"bufio"
	"bytes"
	"io"
	"net/http"
	"strings"
)

const continuedLinePrefix = ' '
const labelSeparator = ' '

var summaryPrefix = []byte("SUMMARY:")
var eventBegin = []byte("BEGIN:VEVENT")
var eventEnd = []byte("END:VEVENT")

var classifications = map[string]byte{
	"loeng":          'l',
	"praktikum":      'p',
	"seminar":        's',
	"praktika":       'i',
	"e-õpe":          'e',
	"kontrolltöö":    'k',
	"kollokvium":     'q',
	"eksam/arvestus": 'a',
	"korduseksam":    'a',
}

func getClassification(label string) byte {
	val, ok := classifications[label]
	if ok {
		return val
	} else {
		return 'x'
	}
}

func constructFilterMap(filterStr string) map[byte]bool {
	filterMap := make(map[byte]bool, len(filterStr))
	for i := 0; i < len(filterStr); i++ {
		filterMap[filterStr[i]] = true
	}
	return filterMap
}

func Handler(w http.ResponseWriter, r *http.Request) {
	params := r.URL.Query()
	calendarId := params.Get("id")
	if len(calendarId) == 0 {
		http.Error(w, "Missing query parameter 'id'", http.StatusBadRequest)
		return
	}

	var filterFunc func(string) bool
	if include := params["include"]; len(include) > 0 {
		filterMap := constructFilterMap(include[0])
		filterFunc = func(label string) bool {
			return filterMap[getClassification(label)]
		}
	} else if exclude := params["exclude"]; len(exclude) > 0 {
		filterMap := constructFilterMap(exclude[0])
		filterFunc = func(label string) bool {
			return !filterMap[getClassification(label)]
		}
	} else {
		filterFunc = func(label string) bool {
			return true
		}
	}

	resp, err := http.Get(fmt.Sprintf("https://www.is.ut.ee/pls/ois/ois.kalender?id_kalender=%s", calendarId))
	if err != nil {
		http.Error(w, "Failed to get calendar", http.StatusInternalServerError)
		return
	}

	body := bufio.NewReader(resp.Body)
	outBuffer := make([]byte, 0, 1024)
	eventBuffer := make([]byte, 0, 256)
	isEvent := false
	includeEvent := false
	summaryBuilder := strings.Builder{}
	isSummary := false
	for {
		line, err := body.ReadBytes('\n')
		fmt.Println(line)
		if err != nil && err != io.EOF {
			http.Error(w, "Error recieving calendar", http.StatusInternalServerError)
			return
		}

		if isEvent {
			eventBuffer = append(eventBuffer, line...)

			if bytes.HasPrefix(line, eventEnd) {
				isEvent = false
				if includeEvent {
					outBuffer = append(outBuffer, eventBuffer...)
				}
			} else {
				if isSummary {
					if line[0] == continuedLinePrefix {
						summaryBuilder.Write(line[1 : len(line)-1])
					} else {
						isSummary = false
						summary := summaryBuilder.String()
						label := summary[strings.LastIndexByte(summary, labelSeparator)+1:]
						includeEvent = filterFunc(label)
					}
				} else if bytes.HasPrefix(line, summaryPrefix) {
					isSummary = true
					summaryBuilder.Reset()
					summaryBuilder.Write(line[len(summaryPrefix)+1 : len(line)-1])
				}
			}
		} else if bytes.HasPrefix(line, eventBegin) {
			isEvent = true
			includeEvent = false
			eventBuffer = eventBuffer[:0]
			eventBuffer = append(eventBuffer, line...)
		} else {
			outBuffer = append(outBuffer, line...)
		}

		if err == io.EOF {
			break
		}
	}

	w.Header().Set("Content-Type", "text/calendar; charset=UTF-8")
	w.Header().Set("Content-Disposition", "inline; filename=\"calendar-filtered.ics\"")

	w.Write(outBuffer)
}
