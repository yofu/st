package main

import (
	"fmt"
	"strconv"
	"strings"

	st "github.com/yofu/st/stlib"
)

func editSectionDialog(frame *st.Frame, name string, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("specify a section number")
	}
	id, err := strconv.Atoi(args[0])
	if err != nil {
		return err
	}
	sec := frame.Sects[id]
	if sec == nil {
		return fmt.Errorf("unknown section %d", id)
	}
	switch name {
	case "DELETE_SECTION":
		if len(args) != 1 {
			return fmt.Errorf("use DELETE_SECTION section")
		}
		for _, el := range frame.Elems {
			if el.Sect.Num == id {
				return fmt.Errorf("section %d is used by element %d", id, el.Num)
			}
		}
		frame.DeleteSect(id)
	case "SECTION_COLOR":
		if len(args) != 4 {
			return fmt.Errorf("use SECTION_COLOR section R G B")
		}
		color := 0
		for _, v := range args[1:] {
			n, err := strconv.ParseUint(v, 10, 8)
			if err != nil {
				return fmt.Errorf("RGB components must be integers from 0 to 255")
			}
			color = color*256 + int(n)
		}
		sec.Color = color
	case "ADD_SECTION_FIG":
		if len(args) > 2 {
			return fmt.Errorf("use ADD_SECTION_FIG section [property]")
		}
		f := st.NewFig()
		f.Num = len(sec.Figs) + 1
		if len(args) == 2 {
			num, err := strconv.Atoi(args[1])
			if err != nil {
				return err
			}
			f.Prop = frame.Props[num]
			if f.Prop == nil {
				return fmt.Errorf("unknown property %d", num)
			}
		} else {
			f.Prop = frame.DefaultProp()
		}
		sec.Figs = append(sec.Figs, f)
	case "EDIT_SECTION_FIG":
		if len(args) < 3 {
			return fmt.Errorf("use EDIT_SECTION_FIG section fig KEY=value [...]")
		}
		num, err := strconv.Atoi(args[1])
		if err != nil {
			return err
		}
		var fig *st.Fig
		for _, f := range sec.Figs {
			if f.Num == num {
				fig = f
				break
			}
		}
		if fig == nil {
			return fmt.Errorf("section %d has no figure %d", id, num)
		}
		for _, arg := range args[2:] {
			key, val, ok := strings.Cut(arg, "=")
			if !ok {
				return fmt.Errorf("expected KEY=value, got %q", arg)
			}
			key = strings.ToUpper(key)
			if key == "PROP" {
				n, err := strconv.Atoi(val)
				if err != nil {
					return err
				}
				if frame.Props[n] == nil {
					return fmt.Errorf("unknown property %d", n)
				}
				fig.Prop = frame.Props[n]
				continue
			}
			f, err := finiteArg(val)
			if err != nil {
				return err
			}
			switch key {
			case "AREA", "IXX", "IYY", "VEN", "THICK":
				fig.Value[key] = f
			case "LLOAD0", "LLOAD1", "LLOAD2":
				sec.Lload[key[len(key)-1]-'0'] = f
			default:
				return fmt.Errorf("unknown section figure field %s", key)
			}
		}
	}
	return nil
}
