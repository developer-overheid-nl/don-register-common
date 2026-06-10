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
}

func (f FilterGroup) Validate() error {
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

func SortOptions(options []FilterOption) {
	sort.Slice(options, func(i, j int) bool {
		left := strings.ToLower(options[i].Label)
		right := strings.ToLower(options[j].Label)
		if left == right {
			return options[i].Value < options[j].Value
		}
		return left < right
	})
}
