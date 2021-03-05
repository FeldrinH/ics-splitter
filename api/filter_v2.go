package handler

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"github.com/FeldrinH/ics-splitter/helpers"
)

func FilterV2(w http.ResponseWriter, r *http.Request) {
	params := r.URL.Query()
	calendarId := params.Get("id")
	if len(calendarId) == 0 {
		http.Error(w, "Missing query parameter 'id'", http.StatusBadRequest)
		return
	}

	resp, err := http.Get(fmt.Sprintf("https://ois2.ut.ee/api/timetable/personal/link/%s/et", calendarId))
	if err != nil {
		http.Error(w, "Failed to get calendar", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	filterFunc := helpers.CreateFilterFunc(params)

	body := bufio.NewReader(resp.Body)
	outBuffer := make([]byte, 0, 1024)

	isEvent := false
	eventBuffer := make([]byte, 0, 256)
	eventLabel := ""

	for {
		line, err := body.ReadBytes('\n')
		if err != nil && err != io.EOF {
			http.Error(w, "Error recieving calendar", http.StatusInternalServerError)
			return
		}

		if isEvent {
			eventBuffer = append(eventBuffer, line...)

			if bytes.HasPrefix(line, helpers.SummaryPrefix) {
				eventLabel = string(line[len(helpers.CategoryPrefix):])
			} else if bytes.HasPrefix(line, helpers.EventEnd) {
				isEvent = false
				if filterFunc(eventLabel) {
					outBuffer = append(outBuffer, eventBuffer...)
				}
			}
		} else if bytes.HasPrefix(line, helpers.EventBegin) {
			isEvent = true
			eventLabel = ""
			eventBuffer = append(eventBuffer[:0], line...)
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
