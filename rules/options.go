package rules

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

func parseOptionMetadata(doc *Document) error {
	colors := map[string][]int{}
	themes := []string{}
	mappings := map[string]map[string]string{}
	doc.Colors = map[string]map[string][]int{}
	doc.Labels = map[string]map[string]any{}
	language := ""
	for _, line := range doc.Directives {
		key, value, ok := strings.Cut(line.Text, ":")
		if !ok {
			continue
		}
		value = strings.TrimSpace(value)
		switch {
		case strings.HasPrefix(key, "COLOR/"):
			parts := strings.Split(value, ",")
			if len(parts) != 3 {
				return fmt.Errorf("line %d: invalid HSL color", line.Line)
			}
			numbers := [3]float64{}
			for i, s := range parts {
				n, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
				if err != nil {
					return fmt.Errorf("line %d: %w", line.Line, err)
				}
				numbers[i] = n
			}
			colors[strings.TrimPrefix(key, "COLOR/")] = hslRGB(numbers[0]/360, numbers[1]/100, numbers[2]/100)
		case key == "OPTCOLORTHEME":
			themes = strings.Fields(value)
			for _, theme := range themes {
				mappings[theme] = map[string]string{}
			}
		case strings.HasPrefix(key, "OPTCOLOR/"):
			values := strings.Fields(value)
			if len(values) != len(themes) {
				return fmt.Errorf("line %d: inconsistent color themes", line.Line)
			}
			for i, v := range values {
				mappings[themes[i]][strings.TrimPrefix(key, "OPTCOLOR/")] = v
			}
		case strings.HasPrefix(key, "OPTLANG/"):
			language = strings.TrimPrefix(key, "OPTLANG/")
			if len(language) > 2 {
				language = language[:2]
			}
			doc.Labels[language] = map[string]any{"__optiontitle__": value}
		case strings.HasPrefix(key, "OPTLABEL/"):
			if language == "" {
				return fmt.Errorf("line %d: option label has no language", line.Line)
			}
			parts := strings.Split(value, "|")
			for i := range parts {
				parts[i] = strings.TrimSpace(parts[i])
			}
			if len(parts) == 1 {
				parts = append(parts, "")
			}
			doc.Labels[language][strings.TrimPrefix(key, "OPTLABEL/")] = parts
		}
	}
	for theme, mapping := range mappings {
		doc.Colors[theme] = map[string][]int{}
		for option, name := range mapping {
			rgb, ok := colors[name]
			if !ok {
				return fmt.Errorf("unknown color %s for option %s", name, option)
			}
			doc.Colors[theme][option] = append([]int(nil), rgb...)
		}
	}
	return nil
}
func hslRGB(h, s, l float64) []int {
	if s == 0 {
		v := int(math.RoundToEven(l * 255))
		return []int{v, v, v}
	}
	upper := l * (1 + s)
	if l > 0.5 {
		upper = l + s - l*s
	}
	lower := 2*l - upper
	value := func(hue float64) int {
		hue = math.Mod(hue, 1)
		if hue < 0 {
			hue += 1
		}
		v := lower
		if hue < 1.0/6 {
			v = lower + (upper-lower)*hue*6
		} else if hue < 0.5 {
			v = upper
		} else if hue < 2.0/3 {
			v = lower + (upper-lower)*(2.0/3-hue)*6
		}
		return int(math.RoundToEven(v * 255))
	}
	return []int{value(h + 1.0/3), value(h), value(h - 1.0/3)}
}
