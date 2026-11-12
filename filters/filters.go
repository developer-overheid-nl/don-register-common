package filters

import (
	"fmt"
	"sort"
	"strings"
)

type FilterOption struct {
	Value       string  `json:"value"`
	Label       string  `json:"label"`
	Description *string `json:"description"`
	Count       int     `json:"count"`
	Selected    bool    `json:"selected"`
}

type FilterGroup struct {
	Key         string         `json:"key"`
	Label       string         `json:"label"`
	Description string         `json:"description"`
	Type        string         `json:"type"`
	Value       any            `json:"value,omitempty"`
	Count       *int           `json:"count,omitempty"`
	Options     []FilterOption `json:"options,omitempty"`
	// ShowTop is how many options a search-select filter shows before the
	// user searches or expands the list.
	ShowTop *int `json:"showTop,omitempty"`
}

func (f FilterGroup) Validate() error {
	if f.ShowTop != nil {
		if f.Type != "search-select" {
			return fmt.Errorf("filter %q: showTop only applies to search-select, not %q", f.Key, f.Type)
		}
		if *f.ShowTop < 1 {
			return fmt.Errorf("filter %q: showTop must be at least 1, got %d", f.Key, *f.ShowTop)
		}
	}
	switch f.Type {
	case "toggle":
		if f.Value == nil {
			return nil
		}
		if _, ok := f.Value.(bool); !ok {
			return fmt.Errorf("filter %q: toggle value must be bool, got %T", f.Key, f.Value)
		}
	case "date":
		if f.Value != nil {
			if _, ok := f.Value.(string); !ok {
				return fmt.Errorf("filter %q: date value must be string, got %T", f.Key, f.Value)
			}
		}
	}
	return nil
}

type FilterCount struct {
	Value string
	Label string
	Count int
}

func LabeledOptions(counts []FilterCount, selected map[string]bool, labels map[string][2]string, sortOptions bool) []FilterOption {
	options := make([]FilterOption, 0, len(counts))
	for _, fc := range counts {
		options = append(options, LabeledOption(fc.Value, fc.Count, selected[fc.Value], labels))
	}
	options = AppendMissingSelectedOptions(options, selected, func(value string) FilterOption {
		return LabeledOption(value, 0, true, labels)
	})
	if sortOptions {
		SortOptions(options)
	}
	return options
}

func LabeledOption(value string, count int, selected bool, labels map[string][2]string) FilterOption {
	label := value
	var desc *string
	if meta, ok := labels[value]; ok {
		label = meta[0]
		d := meta[1]
		desc = &d
	}
	return FilterOption{
		Value:       value,
		Label:       label,
		Description: desc,
		Count:       count,
		Selected:    selected,
	}
}

func AppendMissingSelectedOptions(options []FilterOption, selected map[string]bool, build func(string) FilterOption) []FilterOption {
	seen := make(map[string]bool, len(options))
	for _, option := range options {
		seen[option.Value] = true
	}
	for value, isSelected := range selected {
		if value == "" || !isSelected || seen[value] {
			continue
		}
		options = append(options, build(value))
	}
	return options
}

func SelectedSet(groups ...[]string) map[string]bool {
	m := make(map[string]bool)
	for _, values := range groups {
		for _, raw := range values {
			for _, val := range strings.Split(raw, ",") {
				trimmed := strings.TrimSpace(val)
				if trimmed != "" {
					m[trimmed] = true
				}
			}
		}
	}
	return m
}

func SelectedLowerSet(groups ...[]string) map[string]bool {
	values := SelectedSet(groups...)
	lowered := make(map[string]bool, len(values))
	for val := range values {
		lowered[strings.ToLower(val)] = true
	}
	return lowered
}

// SortOptionsByCount puts the options with the most matches first, for
// filters that show only the top options; ties sort as in SortOptions.
func SortOptionsByCount(options []FilterOption) {
	sort.SliceStable(options, func(i, j int) bool {
		if options[i].Count != options[j].Count {
			return options[i].Count > options[j].Count
		}
		return optionLess(options[i], options[j])
	})
}

func SortOptions(options []FilterOption) {
	sort.Slice(options, func(i, j int) bool {
		return optionLess(options[i], options[j])
	})
}

// optionLess orders options by label, case-insensitive, then by value.
func optionLess(left, right FilterOption) bool {
	leftLabel := strings.ToLower(left.Label)
	rightLabel := strings.ToLower(right.Label)
	if leftLabel == rightLabel {
		return left.Value < right.Value
	}
	return leftLabel < rightLabel
}
