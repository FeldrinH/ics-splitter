package handler

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/FeldrinH/ics-splitter/helpers"
)

type config struct {
	CalendarUrl string           `json:"calendar_url"`
	Groups      map[string]group `json:"groups"`
}

type group struct {
	Mode   string   `json:"mode"`
	Values []string `json:"values"`
}

func loadConfig(configUrl string) (config, error) {
	resp, err := http.Get(configUrl)
	if err != nil {
		return config{}, errors.New("HTTP error when loading config from " + configUrl)
	}
	defer resp.Body.Close()

	decodedConfig := config{}
	err = json.NewDecoder(resp.Body).Decode(&decodedConfig)
	if err != nil {
		return config{}, errors.New("Config is not valid: " + err.Error())
	}

	return decodedConfig, nil
}

func constructFilterFunc(config config, groupName string) (func(string) bool, error) {
	group, ok := config.Groups[groupName]
	if !ok {
		return nil, errors.New("Group "+groupName+" not in config")
	}

	switch group.Mode {
	case "include":
		filterMap := createFilterMap(group.Values)
		return func(label string) bool {
			return filterMap[label]
		}, nil
	case "exclude":
		filterMap := createFilterMap(group.Values)
		return func(label string) bool {
			return !filterMap[label]
		}, nil
	case "all":
		return func(label string) bool {
			return true
		}, nil
	case "exclude-group":
		filterMap := make(map[string]bool, len(group.Values))
		for _, excludeGroupName := range group.Values {
			group, ok := config.Groups[excludeGroupName]
			if !ok {
				return nil, errors.New("Excluded group " + excludeGroupName + " not in config")
			}
			addValuesToFilterMap(filterMap, group.Values)
		}

		return func(label string) bool {
			return !filterMap[label]
		}, nil
	default:
		return nil, errors.New("Unknown group mode " + group.Mode)
	}
}

func addValuesToFilterMap(filterMap map[string]bool, values []string) {
	for i := 0; i < len(values); i++ {
		filterMap[values[i]] = true
	}
}

func createFilterMap(values []string) map[string]bool {
	filterMap := make(map[string]bool, len(values))
	addValuesToFilterMap(filterMap, values)
	return filterMap
}

func FilterConfig(w http.ResponseWriter, r *http.Request) {
	params := r.URL.Query()

	configUrl := params.Get("url")
	if len(configUrl) == 0 {
		http.Error(w, "Missing query parameter 'url'", http.StatusBadRequest)
		return
	}
	groupName := params.Get("group")
	if len(groupName) == 0 {
		http.Error(w, "Missing query parameter 'group'", http.StatusBadRequest)
		return
	}

	config, err := loadConfig(configUrl)
	if err != nil {
		http.Error(w, "Failed to load config: "+err.Error(), http.StatusInternalServerError)
		return
	}

	filterFunc, err := constructFilterFunc(config, groupName)
	if err != nil {
		http.Error(w, "Failed to process config for group "+groupName+": "+err.Error(), http.StatusInternalServerError)
		return
	}

	resp, err := http.Get(config.CalendarUrl)
	if err != nil {
		http.Error(w, "HTTP error when loading calendar from "+config.CalendarUrl, http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

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
			// Check if event has ended
			if bytes.HasPrefix(line, helpers.EventEnd) {
				isEvent = false
			}

			// Append line to event buffer
			eventBuffer = append(eventBuffer, line...)

			if isEvent {
				// If event is ongoing, process various prefix based updates
				if bytes.HasPrefix(line, helpers.CategoriesPrefix) {
					eventLabel = string(helpers.StripLineEnding(line[len(helpers.CategoriesPrefix):]))
				}
			} else {
				// If event has ended, commit event to output buffer, if it matches filter
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
	w.Header().Set("Content-Disposition", "inline; filename=\"calendar-filtered-"+groupName+".ics\"")
	w.Write(outBuffer)
}
