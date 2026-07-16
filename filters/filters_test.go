package filters

import (
	"reflect"
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
