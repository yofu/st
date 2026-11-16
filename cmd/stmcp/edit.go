package main

import (
	"bytes"

	"errors"
	"fmt"
	"math"

	"slices"
	"strconv"
	"strings"

	st "github.com/yofu/st/stlib"
)

const guiEditSchema = `{"type":"object","required":["commands"],"properties":{"proposal_id":{"type":"integer","minimum":1},"commands":{"type":"array","minItems":1,"maxItems":100,"items":{"type":"object","required":["command"],"properties":{"command":{"type":"string"},"node_ids":{"type":"array","items":{"type":"integer","minimum":1},"uniqueItems":true},"element_ids":{"type":"array","items":{"type":"integer","minimum":1},"uniqueItems":true},"confirm":{"type":"boolean"},"inputs":{"type":"array","items":{"type":"object","required":["kind"],"properties":{"kind":{"enum":["node","element","axis","click","finish","fence","text","key","modifier","wheel_up","wheel_down"]},"id":{"type":"integer"},"point":{"type":"array","minItems":3,"maxItems":3,"items":{"type":"number"}},"end_point":{"type":"array","minItems":3,"maxItems":3,"items":{"type":"number"}},"xy":{"type":"array","minItems":2,"maxItems":2,"items":{"type":"integer"}},"end":{"type":"array","minItems":2,"maxItems":2,"items":{"type":"integer"}},"text":{"type":"string"},"shift":{"type":"boolean"},"ctrl":{"type":"boolean"}},"additionalProperties":false}}},"additionalProperties":false}}},"additionalProperties":false}`

type guiEdit struct {
	Command string      `json:"command"`
	Nodes   *[]int      `json:"node_ids"`
	Elems   *[]int      `json:"element_ids"`
	Confirm bool        `json:"confirm"`
	Inputs  []editInput `json:"inputs"`
}

type guiEditArgs struct {
	Proposal uint64    `json:"proposal_id"`
	Commands []guiEdit `json:"commands"`
}

var modelExCommands = strings.Fields(`
	hkyou hweak cross rpipe cpipe tkyou tweak ckyou cweak plate angle
	check nodeduplication elemduplication nodenoreference mergenode nodesort intersectall
	setboundary unchain coord node nodeset conf phase pile scale xscale yscale zscale
	pload cmq lock unlock elem fence filter bond section+ cang setcang axis2cang invert
	prestress thermal join divide dividearc section thick add move copy arraycopy set erase
	elimallpin camber parabola skip insert propsect addnode kunst read
`)

var legacyEditors = strings.Fields(`
	DELETE_SELECTED EDITLINEELEM ADDPLATEALL EDITWRECT DUPLICATEELEM PINCHNODE
	ELEMSAMENODE PRUNEENOD CATBYNODE CATINTERMEDIATENODE JOINPLATEELEM
	BONDPIN BONDRIGID CONFFIX CONFPIN CONFXYROLLER CONFFREE
	CONNECTED ONNODE SELECTNODE SELECTELEM SELECTSECT SELECTCONFED SELECTCOLUMNBASE SELECTCHILDREN
	NODENOREFERENCE NODEDUPLICATION ELEMDUPLICATION NODESORT MERGENODE
	DIVIDEATONS DIVIDEATMID DIVIDEINN DIVIDEATELEM INTERSECTALL INSERT WEIGHTCOPY
DELETE_SECTION SECTION_COLOR ADD_SECTION_FIG EDIT_SECTION_FIG LOCK_NODES UNLOCK_NODES UNDO REDO
`)

func sharedEditor(name string) bool {
	// These commands produce DXF/clipboard reports.
	return st.Commands[name] != nil && name != "SPLINE" && name != "NOTICE1459"
}

func listModelEditors() any {
	shared := make([]string, 0, len(st.Commands))
	for name := range st.Commands {
		if sharedEditor(name) {
			shared = append(shared, name)
		}
	}
	slices.Sort(shared)
	return map[string]any{"interactive": shared, "additional": legacyEditors, "ex": modelExCommands, "project": projectEditors}
}

func selectIDs[T comparable](ids []int, items map[int]T, kind string) ([]T, error) {
	result := make([]T, 0, len(ids))
	seen := make(map[int]bool, len(ids))
	for _, id := range ids {
		item, ok := items[id]
		if !ok || id <= 0 {
			return nil, fmt.Errorf("unknown %s %d", kind, id)
		}
		if seen[id] {
			return nil, fmt.Errorf("duplicate %s %d", kind, id)
		}
		result = append(result, item)
		seen[id] = true
	}
	return result, nil
}

func (s *modelSession) previewGUI(args guiEditArgs) (_ any, err error) {
	defer func() {
		if value := recover(); value != nil {
			err = fmt.Errorf("model edit failed: %v", value)
		}
	}()
	if len(args.Commands) == 0 || len(args.Commands) > 100 {
		return nil, fmt.Errorf("specify 1 to 100 commands")
	}
	source := s.raw
	lst := s.lst
	if args.Proposal != 0 {
		if s.pending == nil || args.Proposal != s.pending.ID {
			return nil, fmt.Errorf("unknown pending proposal %d", args.Proposal)
		}
		source = s.pending.data
		lst = s.proposedLst(s.pending)
	}
	frame, err := readModel(source, s.parsePath)
	if err != nil {
		return nil, err
	}
	before := editorINP(frame)
	w := &editWindow{frame: frame, selection: st.NewSelection()}
	trackHistory := slices.ContainsFunc(args.Commands, func(c guiEdit) bool {
		return strings.EqualFold(strings.TrimSpace(c.Command), "UNDO") || strings.EqualFold(strings.TrimSpace(c.Command), "REDO")
	})
	revisions := [][]byte{source}
	revision := 0
	for i, input := range args.Commands {
		switch strings.ToUpper(strings.TrimSpace(input.Command)) {
		case "UNDO", "REDO":
			if input.Nodes != nil || input.Elems != nil || len(input.Inputs) != 0 {
				return nil, fmt.Errorf("UNDO and REDO take no inputs")
			}
			next := revision + 1
			if strings.EqualFold(strings.TrimSpace(input.Command), "UNDO") {
				next = revision - 1
			}
			if next < 0 || next >= len(revisions) {
				return nil, fmt.Errorf("command %d: no %s state in this batch", i+1, input.Command)
			}
			frame, err = readModel(revisions[next], s.parsePath)
			if err != nil {
				return nil, err
			}
			revision = next
			w.frame = frame
			w.Deselect()
			continue
		}
		if err := s.runGUIEdit(w, input); err != nil {
			return nil, fmt.Errorf("command %d (%s): %w", i+1, input.Command, err)
		}
		if trackHistory {
			snapshot := editorINP(frame)
			current := mergeEditorINP(source, before, snapshot)
			if !bytes.Equal(revisions[revision], current) && (revision > 0 || len(diffEditorINP(before, snapshot)) > 0) {
				revisions = append(revisions[:revision+1], current)
				revision++
			}
		}
	}
	after := editorINP(frame)
	data := mergeEditorINP(source, before, after)
	reloaded, err := readModel(data, s.parsePath)
	if err != nil {
		return nil, fmt.Errorf("edited INP could not be reopened: %w", err)
	}
	diff := diffEditorINP(after, editorINP(reloaded))
	if !sameModelTopology(frame, reloaded) {
		return nil, fmt.Errorf("edited INP changed topology when reopened; check for coincident nodes or dangling references")
	}

	if len(diff) > 0 {
		return nil, fmt.Errorf("edited INP did not retain all values: %+v", diff[:min(len(diff), 3)])
	}
	changes := append(diffEditorINP(editorINP(s.frame), editorINP(reloaded)), diffLst(s.lst, lst)...)
	result := map[string]any{"history": strings.TrimSpace(w.log.String()), "changes": changes, "node_ids": existingIDs(w.SelectedNodes(), frame.Nodes, func(n *st.Node) int { return n.Num }), "element_ids": existingIDs(w.SelectedElems(), frame.Elems, func(e *st.Elem) int { return e.Num })}
	if len(changes) != 0 {
		s.next++
		s.pending = &proposal{ID: s.next, Changes: changes, data: data, lst: lst}
		result["proposal_id"] = s.next
	}
	return result, nil
}

func existingIDs[T comparable](selected []T, available map[int]T, idOf func(T) int) []int {
	var zero T
	ids := make([]int, 0, len(selected))
	for _, item := range selected {
		if item != zero {
			id := idOf(item)
			if available[id] == item {
				ids = append(ids, id)
			}
		}
	}
	return ids
}

func (s *modelSession) runGUIEdit(w *editWindow, input guiEdit) error {
	frame := w.frame
	// Previous commands can delete members without clearing the selection.
	if input.Nodes == nil {
		ids := existingIDs(w.SelectedNodes(), frame.Nodes, func(n *st.Node) int { return n.Num })
		input.Nodes = &ids
	}
	if input.Elems == nil {
		ids := existingIDs(w.SelectedElems(), frame.Elems, func(e *st.Elem) int { return e.Num })
		input.Elems = &ids
	}
	nodes, err := selectIDs(*input.Nodes, frame.Nodes, "node")
	if err != nil {
		return err
	}
	elems, err := selectIDs(*input.Elems, frame.Elems, "element")
	if err != nil {
		return err
	}
	w.SelectNode(nodes)
	w.SelectElem(elems)
	w.confirm = input.Confirm
	if w.frame.View.Gfact == 0 {
		w.frame.View.Gfact = 1
	}
	w.Redraw()
	start := w.log.Len()
	command := strings.TrimSpace(input.Command)
	if strings.HasPrefix(command, ":") {
		if len(input.Inputs) != 0 {
			return fmt.Errorf("ex commands use inline arguments")
		}
		err = s.runModelEx(w, command)
	} else if name := strings.ToUpper(command); sharedEditor(name) {
		err = w.interactive(st.Commands[name], input.Inputs)
	} else {
		err = s.runLegacyEdit(w, strings.Fields(command), input.Inputs)
	}
	if err != nil {
		return err
	}
	log := w.log.String()[start:]
	if strings.Contains(log, "[ERROR]:") || strings.Contains(log, "[CRITICAL]:") {
		return errors.New(strings.TrimSpace(log))
	}
	return nil
}

func (s *modelSession) runModelEx(w *editWindow, command string) error {
	stages := strings.Split(command, "|")
	for i, stage := range stages {
		stage = strings.TrimSpace(stage)
		args := strings.Fields(stage)
		if len(args) == 0 || !strings.HasPrefix(args[0], ":") || strings.ContainsAny(stage, "\t\r\n") {
			return fmt.Errorf("use :command in every pipeline stage")
		}
		name, _, _, _ := st.ExModeComplete(args[0])
		if !slices.Contains(modelExCommands, name) {
			return fmt.Errorf("%q is not a model-editing ex command", name)
		}
		switch name {
		case "addnode":
			if len(stages) != 1 || len(args) != 4 {
				return fmt.Errorf("use :addnode X Y Z")
			}
			var p [3]float64
			for j := range p {
				v, err := finiteArg(args[j+1])
				if err != nil {
					return err
				}
				p[j] = v
			}
			if w.frame.SearchNode(p[0], p[1], p[2]) != nil {
				return fmt.Errorf("a node already exists at these coordinates")
			}
			n := w.frame.AddNode(p[0], p[1], p[2])
			n.Frame = w.frame
			w.Deselect()
			w.SelectNode([]*st.Node{n})
			return nil
		case "setboundary":
			if len(args) >= 2 {
				if _, err := positiveCount(args[1], len(w.frame.Nodes)); err != nil {
					return err
				}
			}
		case "divide", "dividearc":
			if len(args) >= 3 && strings.EqualFold(args[1], "n") {
				if _, err := positiveCount(args[2], 10000); err != nil {
					return err
				}
			}
		case "arraycopy":
			var counts []string
			if len(args) >= 8 && strings.EqualFold(args[1], "rect") {
				counts = args[5:8]
			}
			if len(args) >= 6 && strings.EqualFold(args[1], "polar") {
				counts = args[5:6]
			}
			total := 1
			for _, val := range counts {
				n, err := positiveCount(val, 10000)
				if err != nil {
					return err
				}
				total *= n
				if total > 10000 {
					return fmt.Errorf("at most 10000 array positions per command")
				}
			}
		case "kunst":
			if len(args) < 2 || !slices.Contains([]string{"slab", "line", "spline"}, args[1]) {
				return fmt.Errorf("use :kunst slab, line or spline")
			}
			for _, arg := range args[2:] {
				if value, ok := strings.CutPrefix(strings.ToUpper(arg), "-N="); ok {
					if _, err := positiveCount(value, 10000); err != nil {
						return err
					}
				}
			}
		case "copy":
			// The other subcommands write to the system clipboard.
			if len(args) < 2 || !slices.Contains([]string{"elem", "sec", "sect"}, strings.ToLower(args[1])) {
				return fmt.Errorf("use :copy elem or :copy sect")
			}
			for _, arg := range args[2:] {
				if strings.HasPrefix(arg, "-") && len(arg) > 1 && (arg[1] == '-' || arg[1] >= 'A' && arg[1] <= 'z') {
					return fmt.Errorf(":copy does not accept options in MCP")
				}
			}
		case "insert", "propsect", "intersectall", "read":
			if len(stages) != 1 {
				return fmt.Errorf(":%s runs as a standalone command", name)
			}
			return s.runFileOrIntersection(w, name, args[1:])
		}
		stages[i] = stage
	}
	err := st.ExMode(w, strings.Join(stages, "|"))
	var info st.NotRedraw
	if errors.As(err, &info) {
		return nil
	}
	return err
}

func positiveCount(value string, max int) (int, error) {
	n, err := strconv.Atoi(value)
	if err != nil || n < 1 || n > max {
		return 0, fmt.Errorf("expected an integer from 1 to %d", max)
	}
	return n, nil
}

func finiteArg(value string) (float64, error) {
	n, err := strconv.ParseFloat(value, 64)
	if err != nil || math.IsNaN(n) || math.IsInf(n, 0) {
		return 0, fmt.Errorf("expected a finite number, got %q", value)
	}
	return n, nil
}
