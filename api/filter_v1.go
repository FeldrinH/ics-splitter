package handler

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/FeldrinH/ics-splitter/helpers"
)

func FilterV1(w http.ResponseWriter, r *http.Request) {
	params := r.URL.Query()
	calendarId := params.Get("id")
	if len(calendarId) == 0 {
		http.Error(w, "Missing query parameter 'id'", http.StatusBadRequest)
		return
	}

	resp, err := http.Get(fmt.Sprintf("https://www.is.ut.ee/pls/ois/ois.kalender?id_kalender=%s", calendarId))
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

	label := ""
	isSummary := false
	summaryBuilder := strings.Builder{}

	for {
		line, err := body.ReadBytes('\n')
		if err != nil && err != io.EOF {
			http.Error(w, "Error recieving calendar", http.StatusInternalServerError)
			return
		}

		if isEvent {
			eventBuffer = append(eventBuffer, line...)

			if isSummary {
				if line[0] == ' ' {
					summaryBuilder.Write(helpers.StripLineEnding(line[1:]))
				} else {
					isSummary = false
					summary := summaryBuilder.String()
					label = summary[strings.LastIndexByte(summary, ' ')+1:]
				}
			} else if bytes.HasPrefix(line, helpers.SummaryPrefix) {
				isSummary = true
				summaryBuilder.Reset()
				summaryBuilder.Write(helpers.StripLineEnding(line[len(helpers.SummaryPrefix)+1:]))
			}

			if bytes.HasPrefix(line, helpers.EventEnd) {
				isEvent = false
				if filterFunc(label) {
					outBuffer = append(outBuffer, eventBuffer...)
				}
			}
		} else if bytes.HasPrefix(line, helpers.EventBegin) {
			isEvent = true
			label = ""
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
