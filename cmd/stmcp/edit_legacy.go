package main

import (
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strings"

	st "github.com/yofu/st/stlib"
)

var restraintPresets = map[string][6]bool{
	"CONFFIX":      {true, true, true, true, true, true},
	"CONFPIN":      {true, true, true},
	"CONFXYROLLER": {false, false, true},
	"CONFFREE":     {},
}

func (s *modelSession) runLegacyEdit(w *editWindow, args []string, inputs []editInput) error {
	if len(args) == 0 {
		return fmt.Errorf("enter a model-editing command")
	}
	name := strings.ToUpper(args[0])
	if len(inputs) != 0 && name != "EDITLINEELEM" {
		return fmt.Errorf("this operation takes selections and inline arguments")
	}
	if slices.Contains(projectEditors, name) {
		return w.runProjectEdit(name, args[1:])
	}
	if preset, ok := restraintPresets[name]; ok {
		for _, n := range w.SelectedNodes() {
			if !n.Lock {
				copy(n.Conf, preset[:])
				if name == "CONFFREE" {
					n.Pile = nil
				}
			}
		}
		return nil
	}
	frame, nodes, elems := w.frame, w.SelectedNodes(), w.SelectedElems()
	switch name {
	case "DELETE_SELECTED":
		st.DeleteSelected(w)
	case "DELETE_SECTION", "SECTION_COLOR", "ADD_SECTION_FIG", "EDIT_SECTION_FIG":
		return editSectionDialog(frame, name, args[1:])
	case "LOCK_NODES", "UNLOCK_NODES":
		for _, n := range nodes {
			n.Lock = name == "LOCK_NODES"
		}
	case "EDITLINEELEM":
		return w.interactive(st.EditPlateElem, inputs)
	case "BONDPIN", "BONDRIGID":
		for _, el := range elems {
			if el.Lock || !el.IsLineElem() {
				continue
			}
			if name == "BONDRIGID" {
				clear(el.Bonds)
			} else {
				for _, i := range []int{4, 5, 10, 11} {
					el.Bonds[i] = st.Pin
				}
			}
		}
	case "ADDPLATEALL":
		count := 4
		if len(args) == 2 {
			if args[1] != "3" && args[1] != "4" {
				return fmt.Errorf("use ADDPLATEALL [3|4]")
			}
			count = int(args[1][0] - '0')
		}
		w.SelectElem(addClosedPlates(frame, elems, count))
	case "EDITWRECT":
		if len(args) != 3 {
			return fmt.Errorf("use EDITWRECT width height with selected plates")
		}
		width, err := finiteArg(args[1])
		if err != nil {
			return err
		}
		height, err := finiteArg(args[2])
		if err != nil {
			return err
		}
		for _, el := range elems {
			if !el.IsLineElem() {
				el.Wrect = []float64{width, height}
			}
		}
	case "DUPLICATEELEM":
		added := make([]*st.Elem, 0, len(elems))
		for _, el := range elems {
			if !el.IsNotEditable(frame.Show) {
				copy := el.Snapshot(frame)
				copy.Chain = nil
				frame.AddElem(-1, copy)
				added = append(added, copy)
			}
		}
		w.SelectElem(added)
	case "PINCHNODE":
		if len(args) != 2 {
			return fmt.Errorf("use PINCHNODE z_increment with selected nodes")
		}
		dz, err := finiteArg(args[1])
		if err != nil {
			return err
		}
		for _, n := range nodes {
			n.Coord[2] += dz
		}
	case "ELEMSAMENODE":
		bad := frame.ElemSameNode()
		w.SelectElem(bad)
		if w.confirm {
			for _, el := range bad {
				frame.DeleteElem(el.Num)
			}
		}
	case "PRUNEENOD":
		for _, el := range elems {
			el.PruneEnod()
		}
	case "CATBYNODE":
		for _, n := range nodes {
			if err := frame.CatByNode(n, true); err != nil {
				return err
			}
		}
	case "CATINTERMEDIATENODE":
		nodes = nil
		for _, n := range frame.Nodes {
			var lines []*st.Elem
			for _, el := range frame.SearchElem(n) {
				if el.IsLineElem() {
					lines = append(lines, el)
				}
			}
			if len(lines) >= 2 && !slices.ContainsFunc(lines[1:], func(el *st.Elem) bool {
				return el.Sect != lines[0].Sect || !el.IsParallel(lines[0].Direction(false), w.EPS())
			}) {
				nodes = append(nodes, n)
			}
		}
		slices.SortFunc(nodes, func(a, b *st.Node) int { return a.Num - b.Num })
		w.SelectNode(nodes)
		if w.confirm {
			for _, n := range nodes {
				_ = frame.CatByNode(n, true)
			}
		}
	case "JOINPLATEELEM":
		if len(elems) != 2 || elems[0].IsLineElem() || elems[1].IsLineElem() {
			return fmt.Errorf("select two plate elements")
		}
		return frame.JoinPlateElem(elems[0], elems[1])
	case "MERGENODE":
		if len(nodes) < 2 {
			return fmt.Errorf("select at least two nodes")
		}
		frame.MergeNode(nodes)
	case "CONNECTED":
		var result []*st.Node
		for _, n := range nodes {
			for _, next := range frame.Connected(n) {
				if !slices.Contains(result, next) {
					result = append(result, next)
				}
			}
		}
		w.SelectNode(result)
	case "ONNODE":
		var result []*st.Node
		for _, el := range elems {
			for _, n := range el.OnNode(0, w.EPS()) {
				if !slices.Contains(result, n) {
					result = append(result, n)
				}
			}
		}
		w.SelectNode(result)
	case "SELECTNODE", "SELECTELEM", "SELECTSECT":
		if len(args) != 2 {
			return fmt.Errorf("specify one ID or an ID range")
		}
		command := map[string]string{"SELECTNODE": ":node ", "SELECTELEM": ":elem ", "SELECTSECT": ":elem sect "}[name]
		return s.runModelEx(w, command+args[1])
	case "SELECTCONFED":
		st.SelectConfed(w)
	case "SELECTCOLUMNBASE":
		w.Deselect()
		low, high := 0.0, 0.0
		if limits := frame.Ai.Boundary; len(limits) >= 2 {
			low, high = limits[0], limits[1]
		}
		var result []*st.Node
		for _, el := range frame.Elems {
			if el.Etype != st.COLUMN || el.IsHidden(frame.Show) {
				continue
			}
			for _, n := range el.Enod {
				if n.Coord[2] >= low && n.Coord[2] <= high && !slices.Contains(result, n) {
					result = append(result, n)
				}
			}
		}
		slices.SortFunc(result, func(a, b *st.Node) int { return a.Num - b.Num })
		w.SelectNode(result)
	case "SELECTCHILDREN":
		var children []*st.Elem
		for _, el := range elems {
			for _, child := range el.Children {
				if child != nil && !slices.Contains(children, child) {
					children = append(children, child)
				}
			}
		}
		w.SelectElem(children)
	case "NODENOREFERENCE", "NODEDUPLICATION":
		if !w.confirm {
			result := frame.NodeNoReference()
			if name == "NODEDUPLICATION" {
				result = nil
				for n := range frame.NodeDuplication(w.EPS()) {
					result = append(result, n)
				}
			}
			w.SelectNode(result)
			return nil
		}
		return s.runModelEx(w, ":"+strings.ToLower(name))
	case "ELEMDUPLICATION":
		if err := s.runModelEx(w, ":elemduplication "+strings.Join(args[1:], " ")); err != nil {
			return err
		}
		if w.confirm {
			for _, el := range w.SelectedElems() {
				frame.DeleteElem(el.Num)
			}
		}
	case "NODESORT":
		return s.runModelEx(w, ":nodesort "+strings.Join(args[1:], " "))
	case "DIVIDEATONS", "DIVIDEATMID", "DIVIDEINN", "DIVIDEATELEM":
		method := map[string]string{"DIVIDEATONS": "ons", "DIVIDEATMID": "mid", "DIVIDEINN": "n", "DIVIDEATELEM": "elem"}[name]
		if name == "DIVIDEINN" && len(args) == 2 {
			if _, err := positiveCount(args[1], 10000); err != nil {
				return err
			}
		}
		return s.runModelEx(w, ":divide "+method+" "+strings.Join(args[1:], " "))
	case "INSERT", "INTERSECTALL", "WEIGHTCOPY":
		return s.runFileOrIntersection(w, strings.ToLower(name), args[1:])
	default:
		return fmt.Errorf("unknown model editor %q; see list_model_editors", name)
	}
	return nil
}

func addClosedPlates(frame *st.Frame, elems []*st.Elem, maxSize int) []*st.Elem {
	type neighbours struct {
		order []*st.Node
		has   map[*st.Node]bool
	}
	adjacent := make(map[*st.Node]*neighbours)
	links := func(n *st.Node) *neighbours {
		if adjacent[n] == nil {
			adjacent[n] = &neighbours{has: make(map[*st.Node]bool)}
		}
		return adjacent[n]
	}
	link := func(a, b *st.Node) {
		n := links(a)
		if !n.has[b] {
			n.has[b] = true
			n.order = append(n.order, b)
		}
	}
	lines := make([]*st.Elem, 0, len(elems))
	for _, el := range elems {
		if el.IsLineElem() && el.Enod[0] != el.Enod[1] {
			lines = append(lines, el)
		}
	}
	slices.SortFunc(lines, func(a, b *st.Elem) int { return a.Num - b.Num })
	for _, el := range lines {
		a, b := el.Enod[0], el.Enod[1]
		link(a, b)
		link(b, a)
	}
	var added []*st.Elem
	add := func(nodes ...*st.Node) {
		if len(frame.SearchElem(nodes...)) == 0 {
			added = append(added, frame.AddPlateElem(-1, nodes, frame.DefaultSect(), st.NULL))
		}
	}
	for _, el := range lines {
		a, b := el.Enod[0], el.Enod[1]
		for _, c := range links(a).order {
			if c == b {
				continue
			}
			if links(b).has[c] {
				add(a, b, c)
				continue
			}
			if maxSize < 4 {
				continue
			}
			for _, d := range links(b).order {
				if d != a && d != c && links(c).has[d] {
					add(a, b, d, c)
				}
			}
		}
	}
	return added
}

func (s *modelSession) runFileOrIntersection(w *editWindow, name string, args []string) error {
	if name == "intersectall" {
		if len(args) != 0 {
			return fmt.Errorf("select the line elements to intersect")
		}
		if len(w.SelectedElems()) < 2 {
			return nil
		}
		done := make(chan struct{})
		go func() {
			defer close(done)
			for range w.frame.Lapch {
			}
		}()
		defer func() {
			close(w.frame.Lapch)
			<-done
			w.frame.Lapch = make(chan int)
		}()
		return w.frame.IntersectAll(w.SelectedElems(), w.EPS())
	}
	switch name {
	case "insert":
		if len(args) != 2 || len(w.SelectedNodes()) != 1 {
			return fmt.Errorf("use :insert relative.inp angle_degrees with one selected anchor node")
		}
		angle, err := finiteArg(args[1])
		if err != nil {
			return err
		}
		path, err := inpPath(args[0])
		if err != nil {
			return err
		}
		data, err := s.readEditInput(path)
		if err != nil {
			return err
		}
		return w.frame.ReadInpString(string(data), s.parsePath, w.SelectedNodes()[0].Coord, angle*math.Pi/180, false)
	case "read":
		if len(args) != 2 {
			return fmt.Errorf("use :read result|kijun|weight relative_path")
		}
		kind := strings.ToLower(args[0])
		path := args[1]
		ext := strings.ToLower(filepath.Ext(path))
		switch kind {
		case "result":
			if !slices.Contains([]string{".otl", ".otx", ".oty", ".ohx", ".ohy"}, ext) {
				return fmt.Errorf("expected an ST result file (.otl, .otx, .oty, .ohx or .ohy)")
			}
			return s.withEditFile(path, func(name string) error { return w.frame.ReadResult(name, st.UpdateResult) })
		case "kijun":
			if ext != ".kjn" {
				return fmt.Errorf("expected .kjn")
			}
			return s.withEditFile(path, w.frame.ReadKjn)
		case "weight":
			if ext != ".wgt" {
				return fmt.Errorf("expected .wgt")
			}
			return s.withEditFile(path, w.frame.ReadWgt)
		default:
			return fmt.Errorf("use :read result|kijun|weight relative_path")
		}
	case "propsect", "weightcopy":
		if len(args) != 1 {
			return fmt.Errorf("specify one root-relative input path")
		}
		suffix := ".inp"
		if name == "weightcopy" {
			suffix = ".wgt"
		}
		if !filepath.IsLocal(args[0]) || !strings.EqualFold(filepath.Ext(args[0]), suffix) {
			return fmt.Errorf("expected a root-relative %s file", suffix)
		}
		if name == "propsect" {
			return s.withEditFile(args[0], func(path string) error {
				title := w.frame.Title
				err := w.frame.AddPropAndSect(path, true)
				w.frame.Title = title
				return err
			})
		}
		return s.withEditFile(args[0], w.frame.ReadWgt)
	}
	return fmt.Errorf("unknown import command")
}

func (s *modelSession) withEditFile(path string, f func(string) error) error {
	data, err := s.readEditInput(path)
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(s.parsePath), "edit-*"+strings.ToLower(filepath.Ext(path)))
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	_, writeErr := tmp.Write(data)
	err = tmp.Close()
	if writeErr != nil {
		return writeErr
	}
	if err != nil {
		return err
	}
	return f(tmp.Name())
}
func (s *modelSession) readEditInput(path string) ([]byte, error) {
	if err := s.requireRoot(); err != nil {
		return nil, err
	}
	if !filepath.IsLocal(path) {
		return nil, fmt.Errorf("expected a path within the server root")
	}
	f, err := s.root.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	const limit = 64 << 20
	if !info.Mode().IsRegular() || info.Size() > limit {
		return nil, fmt.Errorf("expected regular input file up to 64 MiB")
	}
	data, err := io.ReadAll(io.LimitReader(f, limit+1))
	if len(data) > limit {
		return nil, fmt.Errorf("input exceeds 64 MiB")
	}
	return data, err
}
