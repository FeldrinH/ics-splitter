package handler

import (
	"fmt"
	//"os"
	"net/http"
)

func Handler(wr http.ResponseWriter, req *http.Request) {
	params := req.URL.Query()
	calendarId := params.Get("id")
	if len(calendarId) == 0 {
		http.Error(wr, "Missing query parameter 'id'", http.StatusBadRequest)
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
		http.Error(wr, "Please query parameter 'id'", http.StatusBadRequest)
		return
	}

	fmt.Fprintf(wr, "%+v", calendarId)
}
