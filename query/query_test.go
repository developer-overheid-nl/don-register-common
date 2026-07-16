package query

import (
	"reflect"
	"testing"

	"github.com/developer-overheid-nl/don-register-common/filters"
)

type testItem struct {
	value  string
	label  string
	values []string
	active bool
}

func TestEscapeSQLLikeEscapesWildcardsAndBackslashes(t *testing.T) {
	got := EscapeSQLLike(`50%\_done`)
	want := `50\%\\\_done`
	if got != want {
		t.Fatalf("EscapeSQLLike() = %q, want %q", got, want)
	}
}

func TestCountByFieldWithLabelFiltersTrimsCountsAndSorts(t *testing.T) {
	items := []testItem{
		{value: " api ", label: "API", active: true},
		{value: "api", label: "Different label", active: true},
		{value: " register ", label: "Register", active: true},
		{value: "ignored", label: "Ignored", active: false},
		{value: " ", label: "Blank", active: true},
	}

	got := CountByFieldWithLabel(items, func(item testItem) bool {
		return item.active
	}, func(item testItem) string {
		return item.value
	}, func(item testItem) string {
		return item.label
	})

	want := []filters.FilterCount{
		{Value: "api", Label: "API", Count: 2},
		{Value: "register", Label: "Register", Count: 1},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("CountByFieldWithLabel() = %#v, want %#v", got, want)
	}
}

func TestCountByFieldUsesValueAsSortKeyWithoutLabels(t *testing.T) {
	items := []testItem{
		{value: "Zulu"},
		{value: "alpha"},
		{value: "Alpha"},
	}

	got := CountByField(items, nil, func(item testItem) string {
		return item.value
	})

	want := []filters.FilterCount{
		{Value: "alpha", Count: 1},
		{Value: "Alpha", Count: 1},
		{Value: "Zulu", Count: 1},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("CountByField() = %#v, want %#v", got, want)
	}
}

func TestCountByArrayFieldCountsTrimmedValues(t *testing.T) {
	items := []testItem{
		{values: []string{"api", " register ", ""}, active: true},
		{values: []string{"api"}, active: true},
		{values: []string{"ignored"}, active: false},
	}

	got := CountByArrayField(items, func(item testItem) bool {
		return item.active
	}, func(item testItem) []string {
		return item.values
	})

	want := []filters.FilterCount{
		{Value: "api", Count: 2},
		{Value: "register", Count: 1},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("CountByArrayField() = %#v, want %#v", got, want)
	}
}

func TestSortFilterCountsUsesLabelFallbackThenValue(t *testing.T) {
	counts := []filters.FilterCount{
		{Value: "b", Label: "Zulu"},
		{Value: "c", Label: "Alpha"},
		{Value: "a", Label: "alpha"},
		{Value: "d"},
	}

	SortFilterCounts(counts)

	got := []string{counts[0].Value, counts[1].Value, counts[2].Value, counts[3].Value}
	want := []string{"a", "c", "d", "b"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("sorted values = %#v, want %#v", got, want)
	}
}
