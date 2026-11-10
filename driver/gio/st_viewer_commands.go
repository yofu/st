package stgio

import (
	"fmt"
	"math"
	"os"
	"path"
	"slices"
	"strconv"
	"strings"
	"unicode"

	st "github.com/yofu/st/stlib"
	mobilekey "golang.org/x/mobile/event/key"
)

// The web editor exposes model commands only. Desktop file/process commands
// and commands whose objects cannot be saved in an INP are not offered here.
var viewerCommands = strings.Fields(`ADDLINEELEM ADDPLATEELEM ADDPLATEELEMBYLINE HATCHPLATEELEM
 MOVENODE MOVEELEM COPYELEM MIRROR DIVIDEATPLANE INTERSECT TRIM EXTEND JOINLINEELEM
 ERASE MATCHPROPERTY COPYBOND TOGGLEBOND EDITPLATEELEM AXISTOCANG MOVETOLINE MOVETOPLANE
 ROTATE SCALE SEARCHELEM NODETOELEM ELEMTONODE DISTS DEFORMEDDISTS`)

func ViewerCommandNames() []string {
	names := append([]string(nil), viewerCommands...)
	names = append(names, ":addnode 0 0 0", "UNDO", "REDO", ":.", "'.")
	for _, name := range viewerExCommands {
		names = append(names, ":"+name)
	}
	for _, name := range viewerFig2Commands {
		names = append(names, "'"+name)
	}
	slices.Sort(names)
	return names
}

func (w *Window) SubmitViewerCommand(command string) bool {
	command = strings.TrimSpace(command)
	if w.viewer == nil || command == "" || len(command) > 1024 {
		return false
	}
	return w.tryEnqueueJob(uiJob{interrupt: true, run: func() {
		w.History("> " + command)
		if err := w.runViewerCommand(command); err != nil {
			w.History(err.Error())
		}
		w.Redraw()
	}})
}

func (w *Window) runViewerCommand(command string) (err error) {
	defer func() {
		if p := recover(); p != nil {
			err = fmt.Errorf("コマンドを実行できません: %v", p)
		}
	}()
	if strings.HasPrefix(command, "'") {
		if w.frame.Name == "" {
			return fmt.Errorf("先にINPファイルを開いてください")
		}
		if err := viewerFiniteArgs(strings.Fields(command)[1:]); err != nil {
			return err
		}
		err := st.Fig2Mode(w, command)
		if message, ok := err.(st.Messager); ok {
			w.History(message.Message())
			return nil
		}
		return err
	}
	if strings.HasPrefix(command, ":") && !strings.EqualFold(strings.Fields(command)[0], ":addnode") {
		return w.runViewerExMode(command)
	}
	if w.frame.Name == "" {
		return fmt.Errorf("先にINPファイルを開いてください")
	}
	if w.viewerAnalysisRunning() {
		return fmt.Errorf("解析中です。編集は解析の完了後に実行してください")
	}
	words := strings.Fields(command)
	name := strings.ToUpper(words[0])
	if len(words) == 1 && slices.Contains(viewerCommands, name) {
		w.ExecCommand(name)
		return nil
	}
	switch name {
	case "UNDO", "REDO":
		if len(words) != 1 {
			break
		}
		operation := w.Undo
		if name == "REDO" {
			operation = w.Redo
		}
		f, err := operation()
		if err != nil {
			return err
		}
		if f != nil {
			w.Deselect()
			w.SetFrame(f)
			w.Changed(true)
		}
		return nil
	case ":ADDNODE":
		if len(words) != 4 {
			return fmt.Errorf("使い方: :addnode X Y Z")
		}
		var p [3]float64
		for i := range p {
			v, err := strconv.ParseFloat(words[i+1], 64)
			if err != nil || math.IsNaN(v) || math.IsInf(v, 0) {
				return fmt.Errorf("座標には有限の数値を指定してください")
			}
			p[i] = v
		}
		n := w.frame.AddNode(p[0], p[1], p[2])
		n.Frame = w.frame
		w.Deselect()
		w.SelectNode([]*st.Node{n})
		st.Snapshot(w)
		w.History(fmt.Sprintf("NODE %d を追加しました。", n.Num))
		return nil
	}
	return fmt.Errorf("Web版で使えないコマンドです: %s", command)
}

// Names used by the menu. Execution is handled by the shared parsers.
var viewerExCommands = strings.Fields(`
	# 1459 593_2_3 add agc all alt amountlst amountprop analysis angle animatespeed arclm001
	arclm101 arclm201 arclm202 arclm203 arclm301 arclm401 area arraycopy average axis2cang
	bclng001 bond camber cang check check_london checkout ckyou cmq color conf coord copy
	count cpipe cross crosssection currentvalue cweak divide dividearc dxf edit elem
	elemdistance elemduplication elimallpin empty eps erase explorer extractarclm facts
	factsbysect fence fig2 fig2print filter fitscale fixmove fixrotate floor gohanlst hardcopy
	height height+ height- hide hkyou hweak increment insert intersectall invert jiku join
	kaberyo keisansho kunst length lock max mergenode min mkdir mono move nminteraction node
	nodeduplication nodenoreference nodeset nodesort noundo object paper parabola pdf pdiv
	phase pile plan plate plateweight pload png postscript prestress printrange procs propsect
	quit range reaction read redraw resource resultant rpipe save saveasarclm scale section
	section+ sectionlist set setangle setboundary setcang show skip sleep slope spline srcal
	srcalangle stop storey story sum suspicious svg tag tex thermal thick timehistory tkyou
	tweak unchain undo unlock vibeig view vim weightcopy write writefig2 writekijun writelst
	writeoutput writereaction xscale yscale zoubundisp zoubunreaction zscale
`)

var viewerFig2Commands = strings.Fields(`
	alias angle anonymous arc axonometric cmqline conf data deformation deformed dfact disp
	dists draw drift eaxis eccentric elem elem+ elem- elemcode energy enormal fence fit focus
	gaxis gfact gray grey height inputfile kijun measure mfact momentfigure ncolor noaxis
	nocaption nodecode nolegend nomomentvalue noshearvalue outputfile pagetitle period
	period++ period-- perpl perspective pilecode pload position prestress qfact rfact sectcode
	section section+ section- sheararrow size skip srcancolor srcanrate stiff stress tensile
	text title unit viewpoint weight width x y z zcoord
`)

func (w *Window) runViewerExMode(command string) (err error) {
	if command == ":." {
		command = w.LastExCommand()
		if command == "" || command == ":." {
			return fmt.Errorf("繰り返すexコマンドがありません")
		}
	}

	// Check every stage before allowing any part of the pipeline to edit.
	stages := strings.Split(command, "|")
	workerCommand := false
	for i, stage := range stages {
		words := strings.Fields(stage)
		if len(words) == 0 || (i == 0 && !strings.HasPrefix(words[0], ":")) || strings.HasPrefix(words[0], "-") {
			return fmt.Errorf("exコマンドの書式を確認してください")
		}
		name, bang, usage, _ := st.ExModeComplete(words[0])
		if len(stages) == 1 && usage {
			switch name {
			case "edit":
				w.History(":edit [作業フォルダのファイル名] または :edit でファイルを選択")
				return nil
			case "write", "save":
				w.History(":write [ファイル名.inp] または :save [ファイル名.inp] で書き出し")
				return nil
			}
		}
		if !slices.Contains(viewerExCommands, name) {
			return fmt.Errorf("Web版で使えないコマンドです: %s", words[0])
		}
		if !usage && name == "stop" && w.viewer.workerJob != 0 && len(stages) == 1 {
			w.viewer.stopWorker(w.viewer.workerJob)
			return nil
		}
		if !usage && w.viewerAnalysisRunning() {
			switch name {
			case "stop", "explorer", "redraw", "count", "view":
			default:
				return fmt.Errorf("解析中です。完了を待つか :stop で停止してください")
			}
		}
		if !usage && w.viewer.startWorker != nil && st.WorkerAnalysisCommand(name) {
			workerCommand = true
		}
		if name == "divide" && !usage && len(words) > 1 {
			if strings.HasPrefix(words[1], "-") {
				return fmt.Errorf(":divide の分割方法を先に指定してください")
			}
			if strings.EqualFold(words[1], "n") {
				if len(words) < 3 {
					return fmt.Errorf(":divide n の分割数を指定してください")
				}
				if n, parseErr := strconv.Atoi(words[2]); parseErr != nil || n < 2 || n > 1000 {
					return fmt.Errorf(":divide n の分割数は2～1000を指定してください")
				}
			}
		}
		if err := viewerFiniteArgs(words[1:]); err != nil {
			return err
		}
		words[0] = ":" + name
		if bang {
			words[0] += "!"
		}
		if usage {
			words[0] += "?"
		}
		stages[i] = strings.Join(words, " ")
	}
	if w.frame.Name == "" {
		switch name, _, _, _ := st.ExModeComplete(strings.Fields(stages[0])[0]); name {
		case "edit", "#", "resource", "mkdir", "vim", "explorer", "quit":
		default:
			return fmt.Errorf("先にINPファイルを開いてください")
		}
	}

	if workerCommand {
		if len(stages) != 1 {
			return fmt.Errorf("解析コマンドはパイプから分けて実行してください")
		}
		return w.startViewerWorker(stages[0])
	}
	err = st.ExMode(w, strings.Join(stages, " | "))
	if _, alreadyLogged := err.(st.NotRedraw); alreadyLogged {
		return nil
	}
	return err
}

func (w *Window) viewerAnalysisRunning() bool {
	if w.viewer.workerJob != 0 {
		return true
	}
	w.ui.mu.Lock()
	progress := w.ui.progress
	w.ui.mu.Unlock()
	if strings.HasPrefix(progress, "Calculating...") || strings.HasPrefix(progress, "Time history...") {
		return true
	}
	for _, frame := range w.frame.Arclms {
		if frame != nil && frame.Running() {
			return true
		}
	}
	return false
}

func viewerFiniteArgs(args []string) error {
	for _, arg := range args {
		if _, value, found := strings.Cut(arg, "="); found {
			arg = value
		}
		if v, err := strconv.ParseFloat(arg, 64); err == nil && (math.IsNaN(v) || math.IsInf(v, 0)) {
			return fmt.Errorf("有限の数値を指定してください")
		}
	}
	return nil
}

func (w *Window) ViewerAction(action string) bool {
	if w.viewer == nil {
		return false
	}
	switch action {
	case "cancel":
		return w.tryEnqueueJob(uiJob{interrupt: true, run: func() { w.Deselect(); w.Redraw() }})
	case "finish":
		return w.tryEnqueueJob(uiJob{run: func() {
			if s := w.interaction.Load(); s != nil {
				s.offer(commandInput{kind: "click", click: st.ClickRight(0, 0)})
			}
		}})
	}
	if value, ok := strings.CutPrefix(action, "key:"); ok {
		k := st.Key{Direction: mobilekey.DirRelease}
		if value == "Enter" {
			k.Code = mobilekey.CodeReturnEnter
		} else if len(value) == 1 && value[0] >= 'A' && value[0] <= 'Z' {
			k.Rune = rune(value[0])
			k.Code = mobilekey.Code(int(mobilekey.CodeA) + int(value[0]-'A'))
		} else {
			return false
		}
		return w.tryEnqueueJob(uiJob{run: func() {
			if s := w.interaction.Load(); s != nil {
				s.offer(commandInput{kind: "key", key: k})
			}
		}})
	}
	return w.viewerDisplayAction(action)
}

// ExportViewerModel runs against a stable model, after prior queued edits.
// The browser offers a download link after it receives the serialized bytes.
func (w *Window) ExportViewerModel(done func(string, []byte, error)) bool {
	return w.ExportViewerModelAs("", done)
}

// ExportViewerModelAs exports to a browser download with an optional basename.
func (w *Window) ExportViewerModelAs(filename string, done func(string, []byte, error)) bool {
	if w.viewer == nil || done == nil {
		return false
	}
	return w.tryEnqueueJob(uiJob{run: func() {
		if w.Executing() {
			done("", nil, fmt.Errorf("操作を確定または中止してから保存してください"))
			return
		}
		if w.frame.Name == "" {
			done("", nil, fmt.Errorf("保存するモデルがありません"))
			return
		}
		if len(w.frame.Nodes) == 0 {
			done("", nil, fmt.Errorf("節点を追加してから保存してください"))
			return
		}
		name := strings.TrimSpace(filename)
		if name == "" {
			name = w.viewerCopyName()
		} else if strings.ContainsAny(name, "/\\:*?\"<>|") || strings.ContainsFunc(name, unicode.IsControl) || !strings.EqualFold(path.Ext(name), ".inp") || len(name) <= 4 || len(name) > 255 {
			done("", nil, fmt.Errorf("保存名はパスを含まない .inp ファイル名を指定してください"))
			return
		}
		var data []byte
		var err error
		func() {
			defer func() {
				if p := recover(); p != nil {
					err = fmt.Errorf("INPを書き出せません: %v", p)
				}
			}()
			data = w.frame.InpBytes()
		}()
		if err == nil {
			w.History("EXPORT: " + name)
		}
		done(name, data, err)
	}})
}

func (w *Window) viewerCopyName() string {
	return strings.TrimSuffix(w.frame.Name, path.Ext(w.frame.Name)) + "-edited.inp"
}

func (w *Window) saveViewerWorkspaceCopy() error {
	name := w.viewerCopyName()
	if err := os.WriteFile(path.Join(w.Cwd(), name), w.frame.InpBytes(), 0644); err != nil {
		return err
	}
	w.OpenBrowserArtifact("download", name)
	return nil
}

type viewerPromptReply struct {
	id       int
	text     string
	accepted bool
}

func (w *Window) ReplyViewerPrompt(id int, text string, accepted bool) bool {
	if w.viewer == nil || w.viewer.promptReplies == nil || len(text) > 1024 {
		return false
	}
	select {
	case w.viewer.promptReplies <- viewerPromptReply{id, text, accepted}:
		w.QueueRedrawAll()
		return true
	default:
		return false
	}
}

// Like the desktop modal, the browser prompt is owned by the frame goroutine.
func (w *Window) updateViewerPrompt() {
	u, v := w.ui, w.viewer
	if d := u.modal; d != nil && d.owner != w.interaction.Load() {
		if d.cancel != nil {
			d.cancel()
		}
		u.modal = nil
	}
	select {
	case d := <-u.pathRequests:
		if d.textOnly && d.owner != nil && d.owner == w.interaction.Load() {
			u.modal = d
			v.promptID++
		} else if d.cancel != nil {
			d.cancel()
		}
	default:
	}
	select {
	case reply := <-v.promptReplies:
		if d := u.modal; d != nil && v.promptID == reply.id {
			u.modal = nil
			if reply.accepted {
				d.apply(reply.text)
			} else if d.cancel != nil {
				d.cancel()
			}
		}
	default:
	}
}
