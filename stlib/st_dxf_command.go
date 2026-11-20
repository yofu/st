package st

import (
	"fmt"
	"strconv"
	"strings"
)

const dxfDocumentUsage = ":drawings filename {-plans=1,2} {-grids=X1,Y2} {-sections} {-sect=101,201} {-paper=A3|A4|A3P|A4P} {-scale=auto|100} {-columns=1} {-rows=1} {-sectioncolumns=3} {-sectionrows=2} {-title=text} {-settings=model.drawing.json} {-rotate=0|90|180|270}"

func parseDXFDocumentOptions(args map[string]string, sectionsOnly bool) (DXFDocumentOptions, error) {
	o := DXFDocumentOptions{Title: args["TITLE"], Font: args["FONT"], Paper: strings.ToUpper(args["PAPER"]), Sections: sectionsOnly, SettingsPath: args["SETTINGS"]}
	if _, ok := args["SECTIONS"]; ok {
		o.Sections = true
	}
	for _, option := range []struct {
		key string
		dst *int
	}{{"COLUMNS", &o.Columns}, {"ROWS", &o.Rows}, {"SECTIONCOLUMNS", &o.SectionColumns}, {"SECTIONROWS", &o.SectionRows}} {
		if text, ok := args[option.key]; ok {
			n, err := strconv.Atoi(text)
			if err != nil || n < 1 || n > 4 {
				return o, fmt.Errorf("invalid %s", option.key)
			}
			*option.dst = n
		}
	}
	if text, ok := args["SCALE"]; ok && !strings.EqualFold(text, "auto") {
		n, err := strconv.ParseFloat(text, 64)
		if err != nil || !finiteDXF(n) || n <= 0 {
			return o, fmt.Errorf("invalid SCALE")
		}
		o.Scale = n
	}
	for _, key := range []string{"PLANS", "SECT"} {
		if text, ok := args[key]; ok {
			for _, word := range strings.Split(text, ",") {
				id, err := strconv.Atoi(word)
				if err != nil || id < 1 {
					return o, fmt.Errorf("invalid %s", key)
				}
				if key == "PLANS" {
					o.Views = append(o.Views, DXFViewSpec{Kind: "plan", Floor: id})
				} else {
					o.Sections = true
					o.SectionIDs = append(o.SectionIDs, id)
				}
			}
		}
	}
	if text, ok := args["GRIDS"]; ok {
		for _, name := range strings.Split(text, ",") {
			if name == "" {
				return o, fmt.Errorf("invalid GRIDS")
			}
			o.Views = append(o.Views, DXFViewSpec{Kind: "elevation", Grid: name})
		}
	}
	if text, ok := args["ROTATE"]; ok {
		n, err := strconv.Atoi(text)
		if err != nil || (n != 0 && n != 90 && n != 180 && n != 270) {
			return o, fmt.Errorf("invalid ROTATE")
		}
		for i := range o.Views {
			o.Views[i].Rotation = dxfPtr(n)
		}
	}
	if !o.Sections && len(o.Views) == 0 {
		return o, fmt.Errorf("specify -plans, -grids or -sections")
	}
	return o, nil
}
