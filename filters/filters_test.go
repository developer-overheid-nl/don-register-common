package filters

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestFilterGroupValidate(t *testing.T) {
	tests := []struct {
		name    string
		group   FilterGroup
		wantErr bool
	}{
		{
			name:  "toggle accepts nil",
			group: FilterGroup{Key: "active", Type: "toggle"},
		},
		{
			name:  "toggle accepts bool",
			group: FilterGroup{Key: "active", Type: "toggle", Value: true},
		},
		{
			name:    "toggle rejects non bool",
			group:   FilterGroup{Key: "active", Type: "toggle", Value: "true"},
			wantErr: true,
		},
		{
			name:  "date accepts string",
			group: FilterGroup{Key: "created", Type: "date", Value: "2026-07-08"},
		},
		{
			name:    "date rejects non string",
			group:   FilterGroup{Key: "created", Type: "date", Value: 20260708},
			wantErr: true,
		},
		{
			name:  "search-select accepts no showTop",
			group: FilterGroup{Key: "organisation", Type: "search-select"},
		},
		{
			name:  "search-select accepts positive showTop",
			group: FilterGroup{Key: "organisation", Type: "search-select", ShowTop: intPtr(10)},
		},
		{
			name:    "search-select rejects zero showTop",
			group:   FilterGroup{Key: "organisation", Type: "search-select", ShowTop: intPtr(0)},
			wantErr: true,
		},
		{
			name:    "showTop only applies to search-select",
			group:   FilterGroup{Key: "license", Type: "multi-select", ShowTop: intPtr(5)},
			wantErr: true,
		},
		{
			name:  "unknown type is ignored",
			group: FilterGroup{Key: "owner", Type: "select", Value: 12},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.group.Validate()
			if (err != nil) != tt.wantErr {
				t.Fatalf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLabeledOptionsAddsMetadataMissingSelectedAndSorts(t *testing.T) {
	selected := map[string]bool{
		"api":      true,
		"missing":  true,
		"ignored":  false,
		"":         true,
		"register": false,
	}
	labels := map[string][2]string{
		"api":     {"API", "Application programming interface"},
		"missing": {"Missing label", "Missing description"},
	}
	counts := []FilterCount{
		{Value: "register", Count: 2},
		{Value: "api", Count: 4},
	}

	got := LabeledOptions(counts, selected, labels, true)

	if len(got) != 3 {
		t.Fatalf("LabeledOptions() returned %d options, want 3: %#v", len(got), got)
	}
	if got[0].Value != "api" || got[0].Label != "API" || !got[0].Selected || got[0].Count != 4 {
		t.Fatalf("first option = %#v, want labeled selected api", got[0])
	}
	if got[0].Description == nil || *got[0].Description != "Application programming interface" {
		t.Fatalf("api description = %v, want metadata description", got[0].Description)
	}
	if got[1].Value != "missing" || got[1].Count != 0 || !got[1].Selected {
		t.Fatalf("second option = %#v, want missing selected option", got[1])
	}
	if got[2].Value != "register" || got[2].Label != "register" || got[2].Selected {
		t.Fatalf("third option = %#v, want unselected register option", got[2])
	}
}

func TestSelectedSetsTrimSplitAndLowercaseValues(t *testing.T) {
	got := SelectedSet([]string{"api, register", "  "}, []string{"docs", "api"})
	want := map[string]bool{"api": true, "register": true, "docs": true}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("SelectedSet() = %#v, want %#v", got, want)
	}

	lowered := SelectedLowerSet([]string{"API, Register"})
	loweredWant := map[string]bool{"api": true, "register": true}
	if !reflect.DeepEqual(lowered, loweredWant) {
		t.Fatalf("SelectedLowerSet() = %#v, want %#v", lowered, loweredWant)
	}
}

func TestSortOptionsUsesLabelCaseInsensitiveThenValue(t *testing.T) {
	options := []FilterOption{
		{Value: "b", Label: "Zulu"},
		{Value: "c", Label: "alpha"},
		{Value: "a", Label: "Alpha"},
	}

	SortOptions(options)

	got := []string{options[0].Value, options[1].Value, options[2].Value}
	want := []string{"a", "c", "b"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("sorted values = %#v, want %#v", got, want)
	}
}

func TestSortOptionsByCountPutsLargestFirstThenLabel(t *testing.T) {
	options := []FilterOption{
		{Value: "a", Label: "Zulu", Count: 3},
		{Value: "b", Label: "beta", Count: 10},
		{Value: "c", Label: "Alpha", Count: 3},
		{Value: "d", Label: "alpha", Count: 3},
	}

	SortOptionsByCount(options)

	got := []string{options[0].Value, options[1].Value, options[2].Value, options[3].Value}
	want := []string{"b", "c", "d", "a"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("sorted values = %#v, want %#v", got, want)
	}
}

func TestFilterGroupMarshalsShowTopOnlyWhenSet(t *testing.T) {
	without, err := json.Marshal(FilterGroup{Key: "license", Type: "multi-select"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(without), "showTop") {
		t.Fatalf("showTop present without value: %s", without)
	}
	with, err := json.Marshal(FilterGroup{Key: "organisation", Type: "search-select", ShowTop: intPtr(10)})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(with), `"showTop":10`) {
		t.Fatalf("showTop missing: %s", with)
	}
}

func intPtr(value int) *int { return &value }
