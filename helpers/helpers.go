package helpers

import (
	"net/url"
)

var EventBegin = []byte("BEGIN:VEVENT")
var EventEnd = []byte("END:VEVENT")
var SummaryPrefix = []byte("SUMMARY:")
var CategoryPrefix = []byte("CATEGORIES:")

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
	"konsultatsioon": 'c',
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

func CreateFilterFunc(params url.Values) func(string) bool {
	if include := params["include"]; len(include) > 0 {
		filterMap := constructFilterMap(include[0])
		return func(label string) bool {
			return filterMap[getClassification(label)]
		}
	} else if exclude := params["exclude"]; len(exclude) > 0 {
		filterMap := constructFilterMap(exclude[0])
		return func(label string) bool {
			return !filterMap[getClassification(label)]
		}
	} else {
		return func(label string) bool {
			return true
		}
	}
}

func StripLineEnding(data []byte) []byte {
	if len(data) >= 1 && data[len(data)-1] == '\n' {
		if len(data) >= 2 && data[len(data)-2] == '\r' {
			return data[0 : len(data)-2]
		}
		return data[0 : len(data)-1]
	}
	return data
}
