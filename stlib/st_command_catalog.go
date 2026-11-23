package st

import (
	_ "embed"
	"strings"
)

//go:generate go run ./internal/commandcatalog
//go:embed native_commands.txt
var nativeCommandIndex string

// NativeCommandSyntax includes exact dispatch names even without a completion template.
func NativeCommandSyntax() map[string]string {
	result := map[string]string{}
	for _, row := range strings.Split(strings.TrimSpace(nativeCommandIndex), "\n") {
		name, usage, _ := strings.Cut(row, "\t")
		if usage == "" {
			usage = name
		}
		result[name] = usage
	}
	for pattern, completion := range ExAbbrev {
		name := ":" + strings.ReplaceAll(pattern, "/", "")
		if result[name] == "" || result[name] == name {
			result[name] = completion.String()
		}
	}
	for pattern, completion := range Fig2Abbrev {
		name := "'" + strings.ReplaceAll(pattern, "/", "")
		if result[name] == "" || result[name] == name {
			result[name] = completion.String()
		}
	}
	return result
}
