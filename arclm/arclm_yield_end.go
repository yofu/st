package arclm

import "fmt"

// ParseYieldEnds reads YIELDEND flags in Enod order: 1 enables material
// yielding, 0 keeps that end elastic. The returned flags disable yielding,
// so an omitted option (the zero value) retains the legacy behavior.
func ParseYieldEnds(words []string) ([2]bool, error) {
	var elastic [2]bool
	if len(words) != 2 {
		return elastic, fmt.Errorf("YIELDEND requires two values (0 or 1)")
	}
	for end, word := range words {
		switch word {
		case "0":
			elastic[end] = true
		case "1":
		default:
			return [2]bool{}, fmt.Errorf("YIELDEND value %d must be 0 or 1, got %q", end+1, word)
		}
	}
	return elastic, nil
}

// FormatYieldEnds omits the legacy default to keep existing exports unchanged.
func FormatYieldEnds(elastic [2]bool) string {
	if elastic == [2]bool{} {
		return ""
	}
	flags := [2]int{1, 1}
	for end, disabled := range elastic {
		if disabled {
			flags[end] = 0
		}
	}
	return fmt.Sprintf("YIELDEND %d %d", flags[0], flags[1])
}
