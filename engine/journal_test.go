package engine

import (
	"reflect"
	"testing"
)

func TestJournalRefs(t *testing.T) {
	for s, want := range map[string][]string{
		"READ JOURNAL ENTRY 26.":                     {"Journal Entry 26"},
		"SEE ENTRIES 23 AND 14.":                     {"Journal Entry 23", "Journal Entry 14"},
		"YOU OVERHEAR TAVERN TALE 5":                 {"Tavern Tale 5"},
		"IN YOUR JOURNAL YOU NOTE PROCLAMATION LIX.": {"Proclamation LIX"},
		"YOU NOTE PROCLAMATIONCXXVI AND CX.":         {"Proclamation CXXVI", "Proclamation CX"},
		"PROCLAMATIONS LXIV, LXXVIII, CIX, AND LIX.": {"Proclamation LXIV", "Proclamation LXXVIII", "Proclamation CIX", "Proclamation LIX"},
		"PROCLAMATIONS ARE POSTED ON THE WALLS":      nil,
		"THE SENTRY 5 LOOKS":                         nil,
		"UNDER ENTRY ":                               nil,
	} {
		if got := journalRefs(s); !reflect.DeepEqual(got, want) {
			t.Errorf("%q: %q, want %q", s, got, want)
		}
	}
}
