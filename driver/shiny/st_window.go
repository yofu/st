package stshiny

import (
	"fmt"
	"github.com/golang/freetype/truetype"
	"github.com/yofu/abbrev"
	"github.com/yofu/st/stlib"
	"golang.org/x/exp/shiny/screen"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
	"golang.org/x/mobile/event/key"
	"golang.org/x/mobile/event/lifecycle"
	"golang.org/x/mobile/event/mouse"
	"golang.org/x/mobile/event/paint"
	"golang.org/x/mobile/event/size"
	"image"
	"image/color"
	"io/ioutil"
	"log"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

var (
	prevkey key.Code
)

var (
	altselectnode = true
)

var (
	blue0 = color.RGBA{0x00, 0x00, 0x1f, 0xff}

	startX = 0
	startY = 0
	endX   = 0
	endY   = 0

	pressed = 0

	commandbuffer  screen.Buffer
	commandtexture screen.Texture
	tailbuffer     screen.Buffer
	tailtexture    screen.Texture
	tailnodes      []*st.Node

	tailColor        = color.RGBA{0xff, 0xff, 0x00, 0x22}
	selectleftColor  = color.RGBA{0x00, 0x00, 0x77, 0x22}
	selectrightColor = color.RGBA{0x00, 0x77, 0x00, 0x22}
)

var (
	Commands = map[string]func(st.Commander) chan bool{
		"D": st.Dists,
		"Q": st.MatchProperty,
		"J": st.JoinLineElem,
		"E": st.Erase,
		"A": st.AddLineElem,
		"H": st.HatchPlateElem,
		"Z": st.Trim,
	}
)

const (
	ButtonLeft = 1 << iota
	ButtonMiddle
)

type Window struct {
	*st.DrawOption
	*st.Directory
	*st.RecentFiles
	*st.UndoStack
	*st.TagFrame
	*st.Selection
	*st.CommandBuffer
	*st.CommandLine
	*st.Alias
	frame         *st.Frame
	screen        screen.Screen
	window        screen.Window
	buffer        screen.Buffer
	currentPen    color.RGBA
	currentBrush  color.RGBA
	fontFace      font.Face
	fontHeight    fixed.Int26_6
	fontColor     color.RGBA
	changed       bool
	lastexcommand string
	lastcommand   func(st.Commander) chan bool
}

func NewWindow(s screen.Screen) *Window {
	return &Window{
		DrawOption:    st.NewDrawOption(),
		Directory:     st.NewDirectory("", ""),
		RecentFiles:   st.NewRecentFiles(3),
		UndoStack:     st.NewUndoStack(10),
		TagFrame:      st.NewTagFrame(),
		Selection:     st.NewSelection(),
		CommandBuffer: st.NewCommandBuffer(),
		CommandLine:   st.NewCommandLine(),
		Alias:         st.NewAlias(),
		frame:         st.NewFrame(),
		screen:        s,
		window:        nil,
		buffer:        nil,
		currentPen:    color.RGBA{0xff, 0xff, 0xff, 0xff},
		currentBrush:  color.RGBA{0xff, 0xff, 0xff, 0x77},
		fontFace:      basicfont.Face7x13,
		fontHeight:    13,
		fontColor:     color.RGBA{0xff, 0xff, 0xff, 0xff},
		changed:       false,
		lastexcommand: "",
		lastcommand:   nil,
	}
}

func keymap(ev key.Event) key.Event {
	switch ev.Code {
	default:
		return ev
	case key.CodeSemicolon:
		r := ev.Rune
		if ev.Modifiers&key.ModShift != 0 {
			r = ';'
		} else {
			r = ':'
		}
		return key.Event{
			Rune:      r,
			Code:      ev.Code,
			Modifiers: ev.Modifiers ^ key.ModShift,
			Direction: ev.Direction,
		}
	}
}

func (stw *Window) Start() {
	w, err := stw.screen.NewWindow(nil)
	if err != nil {
		log.Fatal(err)
	}
	stw.window = w
	defer stw.window.Release()
	err = stw.LoadFontFace(filepath.Join(os.Getenv("HOME"), ".st/fonts/GenShinGothic-Regular.ttf"), 12)
	if err != nil {
		st.ErrorMessage(stw, err, st.ERROR)
	}
	stw.ReadRecent()
	st.ShowRecent(stw)
	stw.frame.View.Center[0] = 512
	stw.frame.View.Center[1] = 512
	stw.Redraw()
	var sz size.Event
	for {
		e := stw.window.NextEvent()
		switch e := e.(type) {
		case lifecycle.Event:
			if e.To == lifecycle.StageDead {
				return
			}
		case key.Event:
			if e.Direction == key.DirPress {
				kc := keymap(e)
				switch kc.Code {
				default:
					stw.TypeCommandLine(string(kc.Rune))
				case key.CodeDeleteBackspace:
					stw.BackspaceCommandLine()
				case key.CodeTab:
					if prevkey == key.CodeTab {
						if e.Modifiers&key.ModShift != 0 {
							stw.PrevComplete()
						} else {
							stw.NextComplete()
						}
					} else {
						stw.Complete()
					}
				case key.CodeSpacebar:
					stw.EndCompletion()
					cl := stw.CommandLineString()
					if strings.Contains(cl, " ") {
						stw.TypeCommandLine(" ")
					} else if strings.HasPrefix(cl, ":") {
						c, bang, usage, comp := st.ExModeComplete(cl)
						var b, u string
						if bang {
							b = "!"
						} else {
							b = ""
						}
						if usage {
							u = "?"
						} else {
							u = ""
						}
						if comp != nil {
							str := fmt.Sprintf(":%s%s%s ", c, b, u)
							stw.SetCommandLineString(str)
							comp.Chdir(stw.Cwd())
							stw.SetComplete(comp)
						}
					} else if strings.HasPrefix(cl, "'") {
						c, usage, comp := st.Fig2KeywordComplete(cl)
						var u string
						if usage {
							u = "?"
						} else {
							u = ""
						}
						if comp != nil {
							str := fmt.Sprintf("'%s%s ", c, u)
							stw.SetCommandLineString(str)
							comp.Chdir(stw.Cwd())
							stw.SetComplete(comp)
						}
					} else {
						stw.TypeCommandLine(" ")
					}
				case key.CodeLeftShift:
				case key.CodeLeftAlt:
				case key.CodeReturnEnter:
					stw.FeedCommand()
				case key.CodeEscape:
					stw.EndCommand()
					stw.ClearCommandLine()
					stw.Deselect()
					stw.Redraw()
					stw.window.Publish()
				case key.CodeLeftControl:
				case key.CodeRightControl:
				case key.CodeY:
					if e.Modifiers&key.ModControl != 0 {
						f, err := stw.Redo()
						if err != nil {
							st.ErrorMessage(stw, err, st.ERROR)
						} else {
							stw.frame = f
						}
					}
				case key.CodeZ:
					if e.Modifiers&key.ModControl != 0 {
						f, err := stw.Undo()
						if err != nil {
							st.ErrorMessage(stw, err, st.ERROR)
						} else {
							stw.frame = f
						}
					}
				}
				stw.Typewrite(25, 700, stw.CommandLineString())
			}
			prevkey = e.Code
		case mouse.Event:
			if e.Button == 4 || e.Button == 5 {
				e.Direction = mouse.DirNone
			}
			switch e.Direction {
			case mouse.DirPress:
				startX = int(e.X)
				startY = int(e.Y)
				switch e.Button {
				case mouse.ButtonLeft:
					pressed |= ButtonLeft
				case mouse.ButtonMiddle:
					pressed |= ButtonMiddle
				}
			case mouse.DirNone:
				endX = int(e.X)
				endY = int(e.Y)
				switch e.Button {
				default:
					if pressed&ButtonLeft != 0 {
						var col color.Color
						if endX >= startX {
							col = selectleftColor
						} else {
							col = selectrightColor
						}
						stw.window.Upload(image.Point{}, stw.buffer, stw.buffer.Bounds())
						stw.window.Fill(image.Rect(startX, startY, endX, endY), col, screen.Over)
						stw.window.Publish()
					} else if pressed&ButtonMiddle != 0 {
						dx := endX - startX
						dy := endY - startY
						if dx&7 == 0 || dy&7 == 0 {
							if e.Modifiers&key.ModShift != 0 {
								stw.frame.View.Center[0] += float64(dx) * stw.CanvasMoveSpeedX()
								stw.frame.View.Center[1] += float64(dy) * stw.CanvasMoveSpeedY()
							} else {
								stw.frame.View.Angle[0] += float64(dy) * stw.CanvasRotateSpeedY()
								stw.frame.View.Angle[1] -= float64(dx) * stw.CanvasRotateSpeedX()
							}
							stw.Redraw()
							stw.window.Publish()
						}
					} else {
						if tailnodes != nil {
							dx := endX - startX
							dy := endY - startY
							if dx&3 == 0 || dy&3 == 0 {
								for i := 0; i< len(tailnodes) - 1; i++ {
									stw.TailLine(int(tailnodes[i].Pcoord[0]), int(tailnodes[i].Pcoord[1]), int(tailnodes[i+1].Pcoord[0]), int(tailnodes[i+1].Pcoord[1]))
								}
								stw.TailLine(int(tailnodes[len(tailnodes)-1].Pcoord[0]), int(tailnodes[len(tailnodes)-1].Pcoord[1]), endX, endY)
							}
						}
					}
				case mouse.ButtonWheelUp, 4:
					stw.ZoomIn(float64(e.X), float64(e.Y))
					stw.Redraw()
					stw.window.Publish()
				case mouse.ButtonWheelDown, 5:
					stw.ZoomOut(float64(e.X), float64(e.Y))
					stw.Redraw()
					stw.window.Publish()
				}
			case mouse.DirRelease:
				endX = int(e.X)
				endY = int(e.Y)
				switch e.Button {
				case mouse.ButtonLeft:
					pressed &= ^ButtonLeft
					if stw.Executing() {
						stw.SendClick(st.ClickLeft(endX, endY))
					}
					if (e.Modifiers&key.ModAlt == 0) == altselectnode {
						els, picked := st.PickElem(stw, startX, startY, endX, endY)
						if stw.Executing() {
							if !picked {
								stw.SendElem(nil)
							} else {
								for _, el := range els {
									stw.SendElem(el)
								}
							}
						} else {
							if !picked {
								stw.DeselectElem()
							} else {
								st.MergeSelectElem(stw, els, e.Modifiers&key.ModShift != 0)
							}
						}
					} else {
						ns, picked := st.PickNode(stw, startX, startY, endX, endY)
						if stw.Executing() {
							if !picked {
								stw.SendNode(nil)
							} else {
								for _, n := range ns {
									stw.SendNode(n)
								}
							}
						} else {
							if !picked {
								stw.DeselectNode()
							} else {
								st.MergeSelectNode(stw, ns, e.Modifiers&key.ModShift != 0)
							}
						}
					}
				case mouse.ButtonMiddle:
					pressed &= ^ButtonMiddle
				case mouse.ButtonRight:
					if stw.Executing() {
						stw.SendClick(st.ClickRight(endX, endY))
					} else {
						if stw.CommandLineString() != "" {
							stw.FeedCommand()
						} else if stw.lastcommand != nil {
							stw.Execute(stw.lastcommand(stw))
						}
					}
				}
				stw.Redraw()
				stw.window.Publish()
			}
		case paint.Event:
			stw.window.Fill(sz.Bounds(), blue0, screen.Src)
			stw.window.Upload(image.Point{}, stw.buffer, stw.buffer.Bounds())
			stw.window.Publish()
		case size.Event:
			sz = e
		case error:
			log.Print(e)
		}
	}
}

func (stw *Window) ZoomIn(x, y float64) {
	stw.Zoom(1.0, x, y)
}

func (stw *Window) ZoomOut(x, y float64) {
	stw.Zoom(-1.0, x, y)
}

func (stw *Window) Zoom(factor float64, x, y float64) {
	val := math.Pow(2.0, factor/stw.CanvasScaleSpeed())
	stw.frame.View.Center[0] += (val - 1.0) * (stw.frame.View.Center[0] - x)
	stw.frame.View.Center[1] += (val - 1.0) * (stw.frame.View.Center[1] - y)
	if stw.frame.View.Perspective {
		stw.frame.View.Dists[1] *= val
		if stw.frame.View.Dists[1] < 0.0 {
			stw.frame.View.Dists[1] = 0.0
		}
	} else {
		stw.frame.View.Gfact *= val
		if stw.frame.View.Gfact < 0.0 {
			stw.frame.View.Gfact = 0.0
		}
	}
}

func (stw *Window) Frame() *st.Frame {
	return stw.frame
}

func (stw *Window) SetFrame(frame *st.Frame) {
	stw.frame = frame
}

func (stw *Window) Redraw() {
	if stw.frame == nil {
		return
	}
	if stw.buffer != nil {
		stw.buffer.Release()
	}
	winSize := image.Point{1024, 1024}
	b, err := stw.screen.NewBuffer(winSize)
	if err != nil {
		log.Fatal(err)
	}
	stw.buffer = b
	st.DrawFrame(stw, stw.frame, st.ECOLOR_SECT, true)
	stw.window.Upload(image.Point{}, stw.buffer, stw.buffer.Bounds())
}

func (stw *Window) RedrawNode() {
	if stw.frame == nil {
		return
	}
	if stw.buffer != nil {
		stw.buffer.Release()
	}
	winSize := image.Point{1024, 1024}
	b, err := stw.screen.NewBuffer(winSize)
	if err != nil {
		log.Fatal(err)
	}
	stw.buffer = b
	st.DrawFrameNode(stw, stw.frame, st.ECOLOR_SECT, true)
	stw.window.Upload(image.Point{}, stw.buffer, stw.buffer.Bounds())
}

func (stw *Window) LoadFontFace(path string, point float64) error {
	f, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	ttf, err := truetype.Parse(f)
	if err != nil {
		return err
	}
	stw.fontFace = truetype.NewFace(ttf, &truetype.Options{
		Size:    point,
		Hinting: font.HintingFull,
	})
	stw.fontHeight = fixed.Int26_6(int(point*3) >> 2) // * 72/96
	return nil
}

func (stw *Window) Typewrite(x, y float64, str string) {
	if str == "" {
		stw.Redraw()
		return
	}
	if commandbuffer != nil {
		commandbuffer.Release()
	}
	b, err := stw.screen.NewBuffer(image.Point{1024, 1024})
	if err != nil {
		log.Fatal(err)
	}
	commandbuffer = b
	d := &font.Drawer{
		Dst:  commandbuffer.RGBA(),
		Src:  image.NewUniform(stw.fontColor),
		Face: stw.fontFace,
		Dot:  fixed.Point26_6{fixed.Int26_6(x * 64), fixed.Int26_6(y * 64)},
	}
	d.DrawString(str)
	t, err := stw.screen.NewTexture(image.Point{1024, 1024})
	if err != nil {
		log.Fatal(err)
	}
	if commandtexture != nil {
		commandtexture.Release()
	}
	commandtexture = t
	t.Upload(image.Point{}, commandbuffer, commandbuffer.Bounds())
	stw.window.Fill(image.Rect(int(x-5), int(y-float64(stw.fontHeight)-5), 500, int(y+5)), color.RGBA{0x33, 0x33, 0x33, 0xff}, screen.Over)
	stw.window.Copy(image.Point{0, 0}, commandtexture, commandtexture.Bounds(), screen.Over, nil)
	stw.window.Publish()
}

func (stw *Window) AddTail(n *st.Node) {
	if tailnodes == nil {
		tailnodes = []*st.Node{n}
	} else {
		tailnodes = append(tailnodes, n)
	}
}

func (stw *Window) EndTail() {
	tailnodes = nil
}

func (stw *Window) TailLine(x1, y1, x2, y2 int) {
	if tailbuffer != nil {
		tailbuffer.Release()
	}
	b, err := stw.screen.NewBuffer(image.Point{1024, 1024})
	if err != nil {
		log.Fatal(err)
	}
	tailbuffer = b
	line(b.RGBA(), x1, y1, x2, y2, tailColor)
	t, err := stw.screen.NewTexture(image.Point{1024, 1024})
	if err != nil {
		log.Fatal(err)
	}
	if tailtexture != nil {
		tailtexture.Release()
	}
	tailtexture = t
	t.Upload(image.Point{}, tailbuffer, tailbuffer.Bounds())
	stw.window.Upload(image.Point{}, stw.buffer, stw.buffer.Bounds())
	stw.window.Copy(image.Point{0, 0}, tailtexture, tailtexture.Bounds(), screen.Over, nil)
	stw.window.Publish()
}

func (stw *Window) FeedCommand() {
	command := stw.CommandLineString()
	if command != "" {
		stw.ClearCommandLine()
		stw.ExecCommand(command)
		stw.Redraw()
	}
}

func (stw *Window) ExecCommand(command string) {
	if stw.frame == nil {
		if strings.HasPrefix(command, ":") {
			err := st.ExMode(stw, command)
			if err != nil {
				st.ErrorMessage(stw, err, st.ERROR)
			}
		} else if strings.HasPrefix(command, "'") {
			err := st.Fig2Mode(stw, command)
			if err != nil {
				st.ErrorMessage(stw, err, st.ERROR)
			}
		}
		return
	}
	switch {
	default:
		if c, ok := Commands[strings.ToUpper(command)]; ok {
			stw.lastcommand = c
			stw.Execute(c(stw))
		} else {
			stw.History(fmt.Sprintf("command doesn't exist: %s", command))
		}
	case strings.HasPrefix(command, ":"):
		err := st.ExMode(stw, command)
		if err != nil {
			st.ErrorMessage(stw, err, st.ERROR)
		}
	case strings.HasPrefix(command, "'"):
		err := st.Fig2Mode(stw, command)
		if err != nil {
			st.ErrorMessage(stw, err, st.ERROR)
		}
	}
}

func (stw *Window) LastExCommand() string {
	return stw.lastexcommand
}

func (stw *Window) SetLastExCommand(command string) {
	stw.lastexcommand = command
}

func (stw *Window) History(str string) {
	fmt.Println(str)
}

func (stw *Window) Print() {
}

func (stw *Window) Changed(c bool) {
	stw.changed = c
}

func (stw *Window) IsChanged() bool {
	return stw.changed
}

func (stw *Window) Yn(string, string) bool {
	return false
}

func (stw *Window) Yna(string, string, string) int {
	return 0
}

func (stw *Window) SaveAS() {
	st.SaveFile(stw, "hogtxt.inp")
}

func (stw *Window) GetCanvasSize() (int, int) {
	return 1024, 1024
}

func (stw *Window) SaveFileSelected(string) error {
	return nil
}

func (stw *Window) SearchFile(fn string) (string, error) {
	return fn, fmt.Errorf("file not found: %s", fn)
}

func (stw *Window) Close(bang bool) {
	if !bang && stw.changed {
		stw.History("changes are not saved")
		return
	}
	err := stw.SaveRecent()
	if err != nil {
		fmt.Println(err)
	}
	os.Exit(0)
}

func (stw *Window) ReadPgp(string) error {
	return nil
}

func (stw *Window) ShapeData(st.Shape) {
}

func (stw *Window) EPS() float64 {
	return 1e-3
}

func (stw *Window) SetEPS(float64) {
}

func (stw *Window) ToggleFixRotate() {
}

func (stw *Window) ToggleFixMove() {
}

func (stw *Window) ToggleAltSelectNode() {
	altselectnode = !altselectnode
}

func (stw *Window) AltSelectNode() bool {
	return altselectnode
}

func (stw *Window) SetAltSelectNode(a bool) {
	altselectnode = a
}

func (stw *Window) SetShowPrintRange(bool) {
}

func (stw *Window) ToggleShowPrintRange() {
}

func (stw *Window) CurrentLap(string, int, int) {
}

func (stw *Window) SectionData(*st.Sect) {
}

func (stw *Window) TextBox(string) st.TextBox {
	return nil
}

func (stw *Window) AxisRange(int, float64, float64, bool) {
}

func (stw *Window) NextFloor() {
}

func (stw *Window) PrevFloor() {
}

func (stw *Window) SetAngle(phi, theta float64) {
	view := st.CanvasCenterView(stw, []float64{phi, theta})
	st.Animate(stw, view)
}

func (stw *Window) SetPaperSize(uint) {
}

func (stw *Window) PaperSize() uint {
	return st.A4_TATE
}

func (stw *Window) SetPeriod(string) {
}

func (stw *Window) Pivot() bool {
	return false
}

func (stw *Window) DrawPivot([]*st.Node, chan int, chan int) {
}

func (stw *Window) SetColorMode(uint) {
}

func (stw *Window) SetConf([]bool) {
}

func (stw *Window) Complete() string {
	var rtn []string
	str := stw.LastWord()
	switch {
	case strings.HasPrefix(str, ":"):
		i := 0
		rtn = make([]string, len(st.ExAbbrev))
		for ab := range st.ExAbbrev {
			pat := abbrev.MustCompile(ab)
			l := fmt.Sprintf(":%s", pat.Longest())
			if strings.HasPrefix(l, str) {
				rtn[i] = l
				i++
			}
		}
		rtn = rtn[:i]
		sort.Strings(rtn)
	case strings.HasPrefix(str, "'"):
		i := 0
		rtn = make([]string, len(st.Fig2Abbrev))
		for ab := range st.Fig2Abbrev {
			pat := abbrev.MustCompile(ab)
			l := fmt.Sprintf("'%s", pat.Longest())
			if strings.HasPrefix(l, str) {
				rtn[i] = l
				i++
			}
		}
		rtn = rtn[:i]
		sort.Strings(rtn)
	default:
		if lis, ok := stw.ContextComplete(); ok {
			rtn = lis
		} else {
			rtn = st.CompleteFileName(str, stw.frame.Path, stw.Recent())
		}
	}
	if len(rtn) == 0 {
		return str
	} else {
		stw.StartCompletion(rtn)
		return rtn[0]
	}
}

func (stw *Window) LastFig2Command() string {
	return ""
}

func (stw *Window) SetLastFig2Command(string) {
}

func (stw *Window) ShowCenter() {
}

func (stw *Window) EnableLabel(string) {
}

func (stw *Window) DisableLabel(string) {
}

func (stw *Window) SetLabel(k, v string) {
}
