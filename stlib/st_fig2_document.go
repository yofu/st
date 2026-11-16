package st

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// A fig2 page may contain multiple viewports. Each viewport has its own
// commands, while paper and page-feed directives are kept with the document.
type Fig2Sheet struct {
	Paper         string
	Number        string
	NoNumber      bool
	Rows, Columns int
	Views         []Fig2View
}

type Fig2View struct {
	Kind     string
	Slot     int
	Commands [][]string
	Source   string
}

var fig2Loop = regexp.MustCompile("(?i)^for\\s+(.+?)\\s+in\\s+(.+?)\\s*:?$")
var fig2Variable = regexp.MustCompile("\\$\\{([^}]+)\\}|\\$[[:alnum:]_]+")
var fig2Lists = regexp.MustCompile("\\[([^]()]*)\\]")

func fig2Substitute(line string, vars map[string]string) string {
	return fig2Variable.ReplaceAllStringFunc(line, func(ref string) string {
		name := strings.Trim(ref[1:], "{}")
		for len(name) > 0 {
			if value, ok := vars[name]; ok {
				if ref[1] == '{' {
					return value
				}
				return value + ref[len(name)+1:]
			}
			if ref[1] == '{' {
				break
			}
			name = name[:len(name)-1]
		}
		return ref
	})
}

func fig2LoopColumns(spec string) ([][]string, error) {
	spec = strings.TrimSpace(strings.TrimSuffix(spec, ":"))
	offset := 1
	numbered := strings.HasPrefix(spec, "#")
	if numbered {
		spec = spec[1:]
		if start, end, ok := strings.Cut(spec, "~"); ok {
			if start == "" {
				start = "1"
			}
			a, ea := strconv.Atoi(start)
			b, eb := strconv.Atoi(end)
			if ea != nil || eb != nil || a < -1000000 || b > 1000000 || a > b || b-a > 10000 {
				return nil, fmt.Errorf("invalid Fig2 range %q", spec)
			}
			column := make([]string, 0, b-a+1)
			for i := a; i <= b; i++ {
				column = append(column, strconv.Itoa(i))
			}
			return [][]string{column}, nil
		}
		i := strings.IndexByte(spec, '[')
		if i < 0 {
			return nil, fmt.Errorf("invalid Fig2 enumeration %q", spec)
		}
		if i > 0 {
			var err error
			offset, err = strconv.Atoi(strings.TrimSpace(spec[:i]))
			if err != nil || offset < -1000000 || offset > 1000000 {
				return nil, fmt.Errorf("invalid Fig2 enumeration offset")
			}
		}
		spec = spec[i:]
	}
	groups := fig2Lists.FindAllStringSubmatchIndex(spec, -1)
	var columns [][]string
	last := 0
	for _, group := range groups {
		sep := strings.TrimSpace(spec[last:group[0]])
		if (last == 0 && sep != "") || (last != 0 && sep != "*" && sep != "x") {
			return nil, fmt.Errorf("invalid Fig2 list %q", spec)
		}
		columns = append(columns, strings.Split(spec[group[2]:group[3]], ","))
		last = group[1]
	}
	if last == 0 || strings.TrimSpace(spec[last:]) != "" {
		return nil, fmt.Errorf("invalid Fig2 list %q", spec)
	}
	if numbered {
		numbers := make([]string, len(columns[0]))
		for i := range numbers {
			numbers[i] = strconv.Itoa(offset + i)
		}
		columns = append([][]string{numbers}, columns...)
	}
	return columns, nil
}

func expandFig2(lines []string, vars map[string]string, out *[]string, frame *Frame) error {
	for i := 0; i < len(lines); i++ {
		line := fig2Substitute(lines[i], vars)
		words := strings.Fields(line)
		switch strings.ToUpper(words[0]) {
		case "EVAL":
			if len(words) < 3 {
				return NotEnoughArgs("EVAL")
			}
			expr, err := compileFig2Expr(strings.Join(words[2:], " "))
			if err != nil {
				return fmt.Errorf("EVAL %s: %w", words[1], err)
			}
			env := map[string]any{"self": frame}
			for k, v := range vars {
				if n, err := strconv.ParseFloat(v, 64); err == nil {
					env[k] = n
				} else {
					env[k] = v
				}
			}
			val, err := expr(env)
			if err != nil {
				return fmt.Errorf("EVAL %s: %w", words[1], err)
			}
			vars[strings.TrimPrefix(words[1], "$")] = fmt.Sprint(val)
		case "CONST", "CONST*":
			if len(words) < 3 {
				return fmt.Errorf("Fig2: CONST requires a name and value")
			}
			vars[strings.TrimPrefix(words[1], "$")] = strings.Join(words[2:], " ")
		case "FOR":
			match := fig2Loop.FindStringSubmatch(line)
			if match == nil {
				return fmt.Errorf("invalid Fig2 loop %q", line)
			}
			names := strings.Split(match[1], ",")
			columns, err := fig2LoopColumns(match[2])
			if err != nil {
				return err
			}
			if len(names) != len(columns) {
				return fmt.Errorf("Fig2 loop has %d variables and %d lists", len(names), len(columns))
			}
			end, depth := i+1, 1
			for ; end < len(lines); end++ {
				switch strings.ToUpper(strings.Fields(lines[end])[0]) {
				case "FOR":
					depth++
				case "ENDFOR":
					depth--
				}
				if depth == 0 {
					break
				}
			}
			if depth != 0 {
				return fmt.Errorf("Fig2: missing ENDFOR for %q", line)
			}
			count := len(columns[0])
			for _, column := range columns[1:] {
				count = min(count, len(column))
			}
			for row := 0; row < count; row++ {
				scope := make(map[string]string, len(vars)+len(names))
				for k, v := range vars {
					scope[k] = v
				}
				for j, name := range names {
					scope[strings.TrimSpace(name)] = strings.TrimSpace(columns[j][row])
				}
				if err := expandFig2(lines[i+1:end], scope, out, frame); err != nil {
					return err
				}
			}
			i = end
		case "ENDFOR":
			return fmt.Errorf("Fig2: unexpected ENDFOR")
		default:
			if len(*out) >= 100000 {
				return fmt.Errorf("Fig2: expanded document exceeds 100000 lines")
			}
			*out = append(*out, line)
		}
	}
	return nil
}

func LoadFig2Sheets(filename string) ([]Fig2Sheet, error) {
	return loadFig2Sheets(filename, make(map[string]bool), nil)
}

func loadFig2Sheets(filename string, active map[string]bool, frame *Frame) ([]Fig2Sheet, error) {
	path, err := filepath.Abs(filename)
	if err != nil {
		return nil, err
	}
	path = filepath.Clean(path)
	if active[path] || len(active) >= 32 {
		return nil, fmt.Errorf("recursive Fig2 LOAD: %s", filename)
	}
	active[path] = true
	defer delete(active, path)
	raw, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	var lines []string
	for _, line := range strings.Split(ToUtf8string(string(raw)), "\n") {
		line = strings.TrimSpace(strings.TrimPrefix(line, "\ufeff"))
		if line == "" || line == "{" || line == "}" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "\\") && len(lines) > 0 {
			lines[len(lines)-1] += " " + line[1:]
		} else {
			lines = append(lines, line)
		}
	}
	var expanded []string
	if err := expandFig2(lines, make(map[string]string), &expanded, frame); err != nil {
		return nil, err
	}
	var sheets []Fig2Sheet
	var common [][]string
	var current *Fig2View
	template := false
	counters := make(map[string]int)
	rows, columns, remaining := 1, 1, 0
	overlay := false
	for _, line := range expanded {
		line = fig2Variable.ReplaceAllStringFunc(line, func(ref string) string {
			name := strings.Trim(ref[1:], "{}")
			for len(name) > 0 {
				if value, ok := counters[name]; ok {
					counters[name] = value + 1
					if ref[1] == '{' {
						return strconv.Itoa(value)
					}
					return strconv.Itoa(value) + ref[len(name)+1:]
				}
				if ref[1] == '{' {
					break
				}
				name = name[:len(name)-1]
			}
			return ref
		})
		args := strings.Fields(line)
		keyword := strings.ToUpper(args[0])
		if template {
			switch keyword {
			case "PAGE", "FIGURE", "FIGURE_H", "FIGURE_V", "COMMON", "TEMPLATE":
				template = false
			default:
				continue
			}
		}
		switch keyword {
		case "IOTA":
			if len(args) < 2 {
				return nil, NotEnoughArgs("IOTA")
			}
			start := 1
			if len(args) >= 3 {
				var err error
				start, err = strconv.Atoi(args[2])
				if err != nil {
					return nil, err
				}
			}
			counters[strings.TrimPrefix(args[1], "$")] = start
		case "LOAD":
			if len(args) < 2 {
				return nil, NotEnoughArgs("LOAD")
			}
			name := strings.Join(args[1:], " ")
			if filepath.Ext(name) == "" {
				name += ".fig2"
			}
			if !filepath.IsAbs(name) {
				name = filepath.Join(filepath.Dir(filename), name)
			}
			loaded, err := loadFig2Sheets(name, active, frame)
			if err != nil {
				return nil, err
			}
			sheets = append(sheets, loaded...)
			current = nil
		case "TEMPLATE":
			template = true
		case "PRINT":
			continue
		case "COMMON":
			common, current = nil, nil
		case "BLANK":
			if current != nil {
				current.Kind = "BLANK"
			}
		case "MULTIPLE":
			if len(args) < 3 {
				return nil, NotEnoughArgs("MULTIPLE")
			}
			r, e1 := strconv.Atoi(args[1])
			c, e2 := strconv.Atoi(args[2])
			if e1 != nil || e2 != nil || r < 1 || c < 1 || r > 100 || c > 100 {
				return nil, fmt.Errorf("invalid MULTIPLE dimensions")
			}
			rows, columns, remaining = r, c, 0
		case "NOFEED":
			overlay = true
		case "PAGE", "FIGURE", "FIGURE_H", "FIGURE_V":
			paper := "A4TATE"
			if len(args) >= 2 {
				paper = strings.ToUpper(args[1])
			}
			if len(sheets) == 0 || (remaining == 0 && !overlay) {
				number := ""
				if len(args) >= 3 && !strings.EqualFold(args[2], "escape") {
					number = strings.Trim(args[2], "#")
				}
				sheets = append(sheets, Fig2Sheet{Paper: paper, Number: number, NoNumber: len(args) >= 3 && strings.EqualFold(args[2], "escape"), Rows: rows, Columns: columns})
				remaining = rows * columns
				rows, columns = 1, 1
			}
			if !overlay && remaining > 0 {
				remaining--
			}
			overlay = false
			sheet := &sheets[len(sheets)-1]
			view := Fig2View{Kind: keyword, Slot: max(0, sheet.Rows*sheet.Columns-remaining-1), Source: filename}
			// The first ESCAPE suppresses the printed number. A second one
			// (or an ESCAPE following a number) suppresses COMMON.
			if len(args) < 4 || !strings.EqualFold(args[len(args)-1], "escape") {
				view.Commands = append(view.Commands, common...)
			}
			sheet.Views = append(sheet.Views, view)
			current = &sheet.Views[len(sheet.Views)-1]
		default:
			if current != nil {
				current.Commands = append(current.Commands, args)
			} else {
				common = append(common, args)
			}
		}
	}
	if len(sheets) == 0 && len(common) > 0 {
		sheets = append(sheets, Fig2Sheet{Paper: "A4TATE", Rows: 1, Columns: 1, Views: []Fig2View{{Kind: "PAGE", Commands: common, Source: filename}}})
	}
	return sheets, nil
}

// ReadFig2View applies a one-based sheet and viewport to the active model.
func ReadFig2View(stw Window, filename string, page, viewport int) error {
	fw, ok := stw.(Fig2Moder)
	if !ok || stw.Frame() == nil {
		return fmt.Errorf("Fig2 requires an open model and a Fig2 window")
	}
	if filename == "" {
		filename = Ce(stw.Frame().Path, ".fig2")
	}
	sheets, err := loadFig2Sheets(filename, make(map[string]bool), stw.Frame())
	if err != nil {
		return err
	}
	if page < 1 || page > len(sheets) {
		return fmt.Errorf("Fig2 page %d is outside 1–%d", page, len(sheets))
	}
	sheet := sheets[page-1]
	if viewport < 1 || viewport > len(sheet.Views) {
		return fmt.Errorf("Fig2 viewport %d is outside 1–%d", viewport, len(sheet.Views))
	}
	if paper, ok := stw.(interface{ SetPaperSize(uint) }); ok {
		switch sheet.Paper {
		case "A4TATE", "A4":
			paper.SetPaperSize(A4_TATE)
		case "A4YOKO":
			paper.SetPaperSize(A4_YOKO)
		case "A3TATE", "A3":
			paper.SetPaperSize(A3_TATE)
		case "A3YOKO":
			paper.SetPaperSize(A3_YOKO)
		}
	}
	resetFig2View(fw)
	if sheet.Views[viewport-1].Kind == "BLANK" {
		return applyFig2Words(fw, []string{"doc"})
	}
	if err := applyFig2View(fw, sheet.Views[viewport-1].Source, sheet.Views[viewport-1].Commands); err != nil {
		return fmt.Errorf("%s, page %d, view %d: %w", filename, page, viewport, err)
	}
	stw.Redraw()
	return nil
}

func applyFig2Words(stw Fig2Moder, args []string) error {
	if len(args) == 0 {
		return nil
	}
	key, un := strings.CutPrefix(args[0], "!")
	if un {
		args = append([]string{key}, args[1:]...)
	}
	switch strings.ToUpper(key) {
	case "PAGE", "FIGURE", "FIGURE_H", "FIGURE_V", "NOFEED":
		return nil
	}
	return Fig2Keyword(stw, args, un)
}

func applyFig2View(fw Fig2Moder, filename string, commands [][]string) error {

	dir := filepath.Dir(filename)
	var err error
	var deferred, expressions [][]string
	for _, args := range commands {
		switch strings.ToUpper(args[0]) {
		case "INPUTFILE", "OUTPUTFILE", "VIEWPOINT":
			deferred = append(deferred, args)
			continue
		case "NODATA":
			deferred = nil
		case "EV", "NV", "RF", "WELDING", "!EV", "!NV", "!RF", "!WELDING":
			expressions = append(expressions, args)
			continue
		}
		if strings.EqualFold(args[0], "READ") || strings.EqualFold(args[0], "READ!") {
			err = readFig2Data(fw, args, dir)
		} else {
			err = applyFig2Words(fw, args)
		}
		if err != nil {
			return fmt.Errorf("%s: %w", strings.Join(args, " "), err)
		}
	}

	for _, args := range expressions {
		if err := applyFig2Words(fw, args); err != nil {
			return fmt.Errorf("%s: %w", strings.Join(args, " "), err)
		}
	}
	for _, args := range deferred {
		if err := applyFig2Words(fw, args); err != nil {
			return err
		}
	}
	if err := positionFig2Annotations(fw); err != nil {
		return err
	}
	return validateFig2Pressure(fw.Frame())
}
