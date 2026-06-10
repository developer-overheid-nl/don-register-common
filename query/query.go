package query

import (
	"sort"
	"strings"

	"github.com/developer-overheid-nl/don-register-common/filters"
)

func EscapeSQLLike(value string) string {
	var builder strings.Builder
	builder.Grow(len(value))
	for _, char := range value {
		switch char {
		case '\\', '%', '_':
			builder.WriteByte('\\')
		}
		builder.WriteRune(char)
	}
	return builder.String()
}

func CountByField[T any](items []T, include func(T) bool, getValue func(T) string) []filters.FilterCount {
	return CountByFieldWithLabel(items, include, getValue, nil)
}

func CountByFieldWithLabel[T any](items []T, include func(T) bool, getValue func(T) string, getLabel func(T) string) []filters.FilterCount {
	counts := make(map[string]int)
	labels := make(map[string]string)
	for _, item := range items {
		if include != nil && !include(item) {
			continue
		}
		val := strings.TrimSpace(getValue(item))
		if val == "" {
			continue
		}
		counts[val]++
		if getLabel == nil {
			continue
		}
		label := strings.TrimSpace(getLabel(item))
		if label == "" {
			label = val
		}
		if labels[val] == "" {
			labels[val] = label
		}
	}
	return sortedCounts(counts, labels)
}

func CountByArrayField[T any](items []T, include func(T) bool, getValues func(T) []string) []filters.FilterCount {
	counts := make(map[string]int)
	for _, item := range items {
		if include != nil && !include(item) {
			continue
		}
		for _, raw := range getValues(item) {
			if val := strings.TrimSpace(raw); val != "" {
				counts[val]++
			}
		}
	}
	return sortedCounts(counts, nil)
}

func sortedCounts(counts map[string]int, labels map[string]string) []filters.FilterCount {
	result := make([]filters.FilterCount, 0, len(counts))
	for value, count := range counts {
		result = append(result, filters.FilterCount{
			Value: value,
			Label: labels[value],
			Count: count,
		})
	}
	SortFilterCounts(result)
	return result
}

func SortFilterCounts(counts []filters.FilterCount) {
	sort.Slice(counts, func(i, j int) bool {
		iKey := filterCountSortKey(counts[i])
		jKey := filterCountSortKey(counts[j])
		if iKey != jKey {
			return iKey < jKey
		}
		return strings.ToLower(counts[i].Value) < strings.ToLower(counts[j].Value)
	})
}

func filterCountSortKey(count filters.FilterCount) string {
	label := strings.TrimSpace(count.Label)
	if label == "" {
		label = count.Value
	}
	return strings.ToLower(label)
}
