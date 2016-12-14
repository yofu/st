package stshiny

import (
	"fmt"
	"image"
	"image/color"
	"io"
	"log"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/yofu/abbrev"
	"github.com/yofu/st/stlib"
	"golang.org/x/exp/shiny/screen"
	"golang.org/x/mobile/event/key"
	"golang.org/x/mobile/event/lifecycle"
	"golang.org/x/mobile/event/mouse"
	"golang.org/x/mobile/event/paint"
	"golang.org/x/mobile/event/size"
)

const (
	NORMAL = iota
	VIEWEDIT
)

var (
	prevkey         key.Event
	cline           string
	drawing         bool
	keymode         = NORMAL
	winSize         = image.Point{1024, 1024}
	TypeWriteOffset = 40
)

var (
	altselectnode = true
)

var (
	home    = os.Getenv("HOME")
	pgpfile = filepath.Join(home, ".st/st.pgp")
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
	frame           *st.Frame
	screen          screen.Screen
	window          screen.Window
	history         *Dialog
	buffer          screen.Buffer
	currentPen      color.RGBA
	currentBrush    color.RGBA
	font            *Font
	papersize       uint
	changed         bool
	lastexcommand   string
	lastfig2command string
	lastcommand     func(st.Commander) chan bool
	textBox         map[string]*st.TextBox
	glasses         map[string]*Glass
}

func NewWindow(s screen.Screen, w screen.Window) *Window {
	return &Window{
		DrawOption:      st.NewDrawOption(),
		Directory:       st.NewDirectory("", ""),
		RecentFiles:     st.NewRecentFiles(3),
		UndoStack:       st.NewUndoStack(10),
		TagFrame:        st.NewTagFrame(),
		Selection:       st.NewSelection(),
		CommandBuffer:   st.NewCommandBuffer(),
		CommandLine:     st.NewCommandLine(),
		Alias:           st.NewAlias(),
		frame:           st.NewFrame(),
		screen:          s,
		window:          w,
		history:         nil,
		buffer:          nil,
		currentPen:      color.RGBA{0xff, 0xff, 0xff, 0xff},
		currentBrush:    color.RGBA{0xff, 0xff, 0xff, 0x77},
		font:            basicFont,
		papersize:       st.A4_TATE,
		changed:         false,
		lastexcommand:   "",
		lastfig2command: "",
		lastcommand:     nil,
		textBox:         make(map[string]*st.TextBox),
		glasses:         make(map[string]*Glass),
	}
}

func (stw *Window) keymap(ev key.Event) key.Event {
	switch ev.Code {
	default:
		return ev
	case key.CodeSemicolon:
		if !stw.CommandLineStringIsEmpty() {
			return ev
		}
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

func (stw *Window) escape() {
	stw.QuitCommand()
	stw.ClearCommandLine()
	stw.Deselect()
	stw.Redraw()
	stw.window.Publish()
}

func (stw *Window) Start(fn string) {
	err := stw.LoadFontFace(filepath.Join(home, ".st/fonts/yumindb.ttf"), 14)
	if err != nil {
		st.ErrorMessage(stw, err, st.ERROR)
	}
	stw.ReadRecent()
	if fn != "" {
		st.OpenFile(stw, fn, true)
	} else {
		st.ShowRecent(stw)
	}
	st.ReadPgp(stw, pgpfile)
	stw.frame.View.Center[0] = 512
	stw.frame.View.Center[1] = 512
	stw.Redraw()
	stw.glasses["TYPEWRITE"] = NewGlass(stw)
	stw.glasses["TYPEWRITE"].SetPosition(TypeWriteOffset, winSize.Y-TypeWriteOffset)
	stw.glasses["TYPEWRITE"].SetSize(winSize.X-2*TypeWriteOffset, stw.font.height.Ceil())
	stw.PopHistoryDialog()
	stw.glasses["HISTORY"].Minimize()
	var sz size.Event
	for {
		e := stw.window.NextEvent()
		switch e := e.(type) {
		case lifecycle.Event:
			if e.To == lifecycle.StageDead {
				return
			}
		case key.Event:
			if stw.Executing() {
				if e.Direction == key.DirRelease && e.Code == key.CodeEscape {
					stw.escape()
					continue
				}
				stw.SendKey(st.Key(e))
				continue
			}
			switch e.Direction {
			case key.DirPress, key.DirNone:
				setprev := true
				if keymode == NORMAL {
					kc := stw.keymap(e)
					typing := true
					switch kc.Code {
					default:
						stw.TypeCommandLine(string(kc.Rune))
					case key.CodeDeleteBackspace:
						stw.BackspaceCommandLine()
						if stw.CommandLineStringIsEmpty() {
							stw.Redraw()
							typing = false
						}
					case key.CodeTab:
						if prevkey.Code == key.CodeTab {
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
						if !stw.AtLast() {
							stw.TypeCommandLine(" ")
						} else if strings.Contains(cl, " ") {
							if lis, ok := stw.ContextComplete(); ok {
								cls := strings.Split(cl, " ")
								cls[len(cls)-1] = lis[0] + " "
								stw.SetCommandLineString(strings.Join(cls, " "))
							} else {
								stw.TypeCommandLine(" ")
							}
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
								stw.History(comp.String())
							} else {
								stw.TypeCommandLine(" ")
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
								stw.History(comp.String())
							} else {
								stw.TypeCommandLine(" ")
							}
						} else {
							stw.TypeCommandLine(" ")
						}
					case key.CodeUnknown:
						setprev = false
						typing = false
					case key.CodeLeftShift:
						setprev = false
						typing = false
					case key.CodeLeftAlt:
						setprev = false
						typing = false
					case key.CodeReturnEnter:
						if e.Modifiers&key.ModControl != 0 {
							fn := filepath.Join(stw.Cwd(), "command.txt")
							if st.FileExists(fn) {
								st.ReadResource(stw, fn)
							} else {
								stw.History(fmt.Sprintf("%s doesn't exist", fn))
							}
						} else {
							stw.FeedCommand()
						}
					case key.CodeEscape:
						stw.escape()
						typing = false
					case key.CodeLeftSquareBracket:
						if e.Modifiers&key.ModControl != 0 {
							stw.escape()
							typing = false
						} else {
							stw.TypeCommandLine("[")
						}
					case key.CodeLeftControl:
						setprev = false
						typing = false
					case key.CodeRightControl:
						setprev = false
						typing = false
					case key.CodeRightArrow:
						stw.SeekForward()
					case key.CodeLeftArrow:
						stw.SeekBackward()
					case key.CodeDownArrow:
						if e.Modifiers&key.ModControl != 0 {
							st.PrevFloor(stw)
							stw.Redraw()
							typing = false
						} else {
							stw.SeekLast()
						}
					case key.CodeUpArrow:
						if e.Modifiers&key.ModControl != 0 {
							st.NextFloor(stw)
							stw.Redraw()
							typing = false
						} else {
							stw.SeekHead()
						}
					case key.CodeW:
						if e.Modifiers&key.ModControl != 0 {
							stw.PopHistoryDialog()
							typing = false
						} else {
							stw.TypeCommandLine(string(kc.Rune))
						}
					case key.CodeM:
						if e.Modifiers&key.ModControl != 0 {
							keymode = VIEWEDIT
							typing = false
							stw.Redraw()
						} else {
							stw.TypeCommandLine(string(kc.Rune))
						}
					case key.CodeD:
						if e.Modifiers&key.ModControl != 0 {
							st.HideNotSelected(stw)
							stw.Redraw()
							typing = false
						} else {
							stw.TypeCommandLine(string(kc.Rune))
						}
					case key.CodeF:
						if e.Modifiers&key.ModControl != 0 {
							st.SetFocus(stw)
							stw.Redraw()
							typing = false
						} else {
							stw.TypeCommandLine(string(kc.Rune))
						}
					case key.CodeA:
						if e.Modifiers&key.ModControl != 0 {
							st.SelectNotHidden(stw)
							stw.Redraw()
							typing = false
						} else {
							stw.TypeCommandLine(string(kc.Rune))
						}
					case key.CodeS:
						if e.Modifiers&key.ModControl != 0 {
							st.ShowAll(stw)
							stw.Redraw()
							typing = false
						} else {
							stw.TypeCommandLine(string(kc.Rune))
						}
					case key.CodeR:
						if e.Modifiers&key.ModControl != 0 {
							st.ReadAll(stw)
							stw.Redraw()
							typing = false
						} else {
							stw.TypeCommandLine(string(kc.Rune))
						}
					case key.CodeP:
						if e.Modifiers&key.ModControl != 0 {
							if !((prevkey.Code == key.CodeP || prevkey.Code == key.CodeN) && prevkey.Modifiers&key.ModControl != 0) {
								cline = stw.CommandLineString()
							}
							stw.PrevCommandHistory(cline)
						} else {
							stw.TypeCommandLine(string(kc.Rune))
						}
					case key.CodeN:
						if e.Modifiers&key.ModControl != 0 {
							if !((prevkey.Code == key.CodeP || prevkey.Code == key.CodeN) && prevkey.Modifiers&key.ModControl != 0) {
								cline = stw.CommandLineString()
							}
							stw.NextCommandHistory(cline)
						} else {
							stw.TypeCommandLine(string(kc.Rune))
						}
					case key.CodeY:
						if e.Modifiers&key.ModControl != 0 {
							f, err := stw.Redo()
							if err != nil {
								st.ErrorMessage(stw, err, st.ERROR)
							} else {
								stw.frame = f
							}
							stw.Redraw()
							typing = false
						} else {
							stw.TypeCommandLine(string(kc.Rune))
						}
					case key.CodeZ:
						if e.Modifiers&key.ModControl != 0 {
							f, err := stw.Undo()
							if err != nil {
								st.ErrorMessage(stw, err, st.ERROR)
							} else {
								stw.frame = f
							}
							stw.Redraw()
							typing = false
						} else {
							stw.TypeCommandLine(string(kc.Rune))
						}
					case key.CodeC:
						if e.Modifiers&key.ModControl != 0 {
							err := st.CopyClipboard(stw)
							if err != nil {
								st.ErrorMessage(stw, err, st.ERROR)
							}
							stw.Redraw()
							typing = false
						} else {
							stw.TypeCommandLine(string(kc.Rune))
						}
					case key.CodeV:
						if e.Modifiers&key.ModControl != 0 {
							err := st.PasteClipboard(stw)
							if err != nil {
								st.ErrorMessage(stw, err, st.ERROR)
							}
							stw.Redraw()
							typing = false
						} else {
							stw.TypeCommandLine(string(kc.Rune))
						}
					case key.CodeX:
						if e.Modifiers&key.ModControl != 0 {
							if h, ok := stw.glasses["HISTORY"]; ok {
								if h.Minimized() {
									h.Maximize()
								} else {
									h.Minimize()
								}
								h.Redraw()
								stw.Redraw()
							}
							typing = false
						} else {
							stw.TypeCommandLine(string(kc.Rune))
						}
					}
					if typing {
						stw.Typewrite(stw.CommandLineStringWithPosition())
					}
					if setprev {
						prevkey = e
					}
				} else if keymode == VIEWEDIT {
					redraw := true
					kc := stw.keymap(e)
					switch kc.Code {
					default:
						redraw = false
					case key.CodeEscape:
						keymode = NORMAL
					case key.CodeLeftSquareBracket:
						if e.Modifiers&key.ModControl != 0 {
							keymode = NORMAL
						}
					case key.CodeM:
						if e.Modifiers&key.ModControl != 0 {
							keymode = NORMAL
						}
					case key.CodeI, key.CodeA, key.CodeC, key.CodeS:
						keymode = NORMAL
					case key.CodeSemicolon:
						if kc.Rune == ':' {
							keymode = NORMAL
							stw.Redraw()
							stw.TypeCommandLine(":")
							stw.Typewrite(stw.CommandLineStringWithPosition())
							redraw = false
						}
					case key.CodeApostrophe:
						if kc.Rune == '\'' {
							keymode = NORMAL
							stw.Redraw()
							stw.TypeCommandLine("'")
							stw.Typewrite(stw.CommandLineStringWithPosition())
							redraw = false
						}
					case key.CodeH:
						dx := -100.0
						if e.Modifiers&key.ModShift != 0 {
							stw.frame.View.Center[0] += dx * stw.CanvasMoveSpeedX()
						} else {
							stw.frame.View.Angle[1] -= dx * stw.CanvasRotateSpeedX()
						}
					case key.CodeJ:
						dy := 100.0
						if e.Modifiers&key.ModShift != 0 {
							stw.frame.View.Center[1] += dy * stw.CanvasMoveSpeedY()
						} else {
							stw.frame.View.Angle[0] += dy * stw.CanvasRotateSpeedY()
						}
					case key.CodeK:
						dy := -100.0
						if e.Modifiers&key.ModShift != 0 {
							stw.frame.View.Center[1] += dy * stw.CanvasMoveSpeedY()
						} else {
							stw.frame.View.Angle[0] += dy * stw.CanvasRotateSpeedY()
						}
					case key.CodeL:
						dx := 100.0
						if e.Modifiers&key.ModShift != 0 {
							stw.frame.View.Center[0] += dx * stw.CanvasMoveSpeedX()
						} else {
							stw.frame.View.Angle[1] -= dx * stw.CanvasRotateSpeedX()
						}
					case key.CodeZ:
						if e.Modifiers&key.ModShift != 0 {
							stw.ZoomOut(stw.frame.View.Center[0], stw.frame.View.Center[1])
						} else {
							stw.ZoomIn(stw.frame.View.Center[0], stw.frame.View.Center[1])
						}
					}
					if redraw {
						stw.Redraw()
					}
				}
			}
		case mouse.Event:
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
						stw.RedrawNode()
						stw.window.Publish()
					}
				} else {
					if tailnodes != nil {
						dx := endX - startX
						dy := endY - startY
						stw.SendPosition(dx, dy)
						if dx&3 == 0 || dy&3 == 0 {
							stw.TailLine()
						}
					}
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
								stw.SendModifier(st.Modifier{
									Shift: e.Modifiers&key.ModShift != 0,
								})
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
								stw.SendModifier(st.Modifier{
									Shift: e.Modifiers&key.ModShift != 0,
								})
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
			case mouse.DirStep:
				endX = int(e.X)
				endY = int(e.Y)
				switch e.Button {
				case mouse.ButtonWheelUp:
					if stw.Executing() {
						sent := stw.SendWheel(st.WheelUp)
						if !sent {
							if e.Modifiers&key.ModControl != 0 {
								stw.ZoomIn(stw.frame.View.Center[0], stw.frame.View.Center[1])
							} else {
								stw.ZoomIn(float64(e.X), float64(e.Y))
							}
							stw.Redraw()
							stw.window.Publish()
						}
					} else {
						if e.Modifiers&key.ModControl != 0 {
							stw.ZoomIn(stw.frame.View.Center[0], stw.frame.View.Center[1])
						} else {
							stw.ZoomIn(float64(e.X), float64(e.Y))
						}
					}
				case mouse.ButtonWheelDown:
					if stw.Executing() {
						sent := stw.SendWheel(st.WheelDown)
						if !sent {
							if e.Modifiers&key.ModControl != 0 {
								stw.ZoomOut(stw.frame.View.Center[0], stw.frame.View.Center[1])
							} else {
								stw.ZoomOut(float64(e.X), float64(e.Y))
							}
							stw.Redraw()
							stw.window.Publish()
						}
					} else {
						if e.Modifiers&key.ModControl != 0 {
							stw.ZoomOut(stw.frame.View.Center[0], stw.frame.View.Center[1])
						} else {
							stw.ZoomOut(float64(e.X), float64(e.Y))
						}
					}
				}
				if !stw.Executing() {
					stw.Redraw()
					stw.window.Publish()
				}
			}
		case paint.Event:
			stw.window.Fill(sz.Bounds(), blue0, screen.Src)
			stw.window.Upload(image.Point{}, stw.buffer, stw.buffer.Bounds())
			stw.window.Publish()
		case size.Event:
			sz = e
			winSize = image.Point{sz.WidthPx, sz.HeightPx}
			stw.glasses["TYPEWRITE"].SetPosition(TypeWriteOffset, winSize.Y-TypeWriteOffset)
			stw.glasses["TYPEWRITE"].SetSize(winSize.X-2*int(TypeWriteOffset), stw.font.height.Ceil())
			stw.glasses["TYPEWRITE"].Redraw()
			if h, ok := stw.glasses["HISTORY"]; ok {
				h.SetPosition(winSize.X-340, winSize.Y-80)
				h.SetSize(300, winSize.Y-100)
				h.Redraw()
			}
			stw.Redraw()
			stw.window.Publish()
		case error:
			log.Print(e)
		}
	}
}

func (stw *Window) CurrentPointerPosition() []int {
	return []int{endX, endY}
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
	if keymode == VIEWEDIT {
		stw.RedrawNode()
		return
	}
	if drawing {
		return
	}
	if stw.frame == nil {
		return
	}
	drawing = true
	if stw.buffer != nil {
		stw.buffer.Release()
	}
	b, err := stw.screen.NewBuffer(winSize)
	if err != nil {
		log.Fatal(err)
	}
	stw.buffer = b
	st.DrawFrame(stw, stw.frame, stw.frame.Show.ColorMode, true)
	for _, t := range stw.textBox {
		if t.IsHidden(stw.frame.Show) {
			continue
		}
		st.DrawText(stw, t)
	}
	stw.window.Upload(image.Point{}, stw.buffer, stw.buffer.Bounds())
	if stw.Executing() {
		stw.window.Fill(image.Rect(0, 0, 10, 10), color.RGBA{0xff, 0x00, 0x00, 0x22}, screen.Over)
	}
	if h, ok := stw.glasses["HISTORY"]; ok {
		h.Upload()
	}
	drawing = false
}

func (stw *Window) RedrawNode() {
	if drawing {
		return
	}
	if stw.frame == nil {
		return
	}
	drawing = true
	if stw.buffer != nil {
		stw.buffer.Release()
	}
	b, err := stw.screen.NewBuffer(winSize)
	if err != nil {
		log.Fatal(err)
	}
	stw.buffer = b
	st.DrawFrameNode(stw, stw.frame, stw.frame.Show.ColorMode, true)
	stw.window.Upload(image.Point{}, stw.buffer, stw.buffer.Bounds())
	drawing = false
}

func (stw *Window) LoadFontFace(path string, point float64) error {
	font, err := LoadFontFace(path, point)
	if err != nil {
		return err
	}
	stw.font = font
	return nil
}

func (stw *Window) Typewrite(str string) {
	stw.glasses["TYPEWRITE"].Show()
	stw.glasses["TYPEWRITE"].SetText([]string{str})
	stw.glasses["TYPEWRITE"].Redraw()
	stw.glasses["TYPEWRITE"].Hide()
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

func (stw *Window) TailLine() {
	if tailbuffer != nil {
		tailbuffer.Release()
	}
	b, err := stw.screen.NewBuffer(winSize)
	if err != nil {
		log.Fatal(err)
	}
	tailbuffer = b
	cvs := b.RGBA()
	for i := 0; i < len(tailnodes)-1; i++ {
		line(cvs, int(tailnodes[i].Pcoord[0]), int(tailnodes[i].Pcoord[1]), int(tailnodes[i+1].Pcoord[0]), int(tailnodes[i+1].Pcoord[1]), tailColor)
	}
	line(cvs, int(tailnodes[len(tailnodes)-1].Pcoord[0]), int(tailnodes[len(tailnodes)-1].Pcoord[1]), endX, endY, tailColor)
	t, err := stw.screen.NewTexture(winSize)
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
		stw.AddCommandHistory(command)
		stw.ClearCommandLine()
		stw.ExecCommand(command)
		stw.Redraw()
		stw.window.Publish()
	}
}

func (stw *Window) ExecCommand(command string) {
	if stw.frame == nil {
		if strings.HasPrefix(command, ":") {
			err := st.ExMode(stw, command)
			if err != nil {
				if _, ok := err.(st.NotRedraw); ok {
					return
				} else {
					st.ErrorMessage(stw, err, st.ERROR)
				}
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
		if c, ok := stw.CommandAlias(strings.ToUpper(command)); ok {
			stw.lastcommand = c
			stw.Execute(c(stw))
		} else {
			stw.History(fmt.Sprintf("command doesn't exist: %s", command))
		}
	case strings.HasPrefix(command, ":"):
		err := st.ExMode(stw, command)
		if err != nil {
			if _, ok := err.(st.NotRedraw); ok {
				return
			} else {
				st.ErrorMessage(stw, err, st.ERROR)
			}
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

func (stw *Window) PopHistoryDialog() {
	if h, ok := stw.glasses["HISTORY"]; ok {
		h.show = !h.show
		if h.show {
			h.Redraw()
		} else {
			stw.Redraw()
		}
		return
	}
	g := NewGlass(stw)
	g.SetPosition(winSize.X-340, winSize.Y-80)
	g.SetSize(300, winSize.Y-100)
	g.Redraw()
	stw.glasses["HISTORY"] = g
}

func (stw *Window) History(str string) {
	if str == "" {
		return
	}
	if stw.history != nil {
		stw.history.TypeString(str)
		stw.history.Redraw()
	} else if h, ok := stw.glasses["HISTORY"]; ok {
		h.TypeString(strings.TrimSuffix(str, "\n"))
		h.Redraw()
	} else {
		if strings.HasSuffix(str, "\n") {
			fmt.Printf(str)
		} else {
			fmt.Println(str)
		}
	}
}

func (stw *Window) HistoryWriter() io.Writer {
	if stw.history != nil {
		return stw.history
	} else if h, ok := stw.glasses["HISTORY"]; ok {
		return h
	} else {
		return os.Stdout
	}
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
	return winSize.X, winSize.Y
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

func (stw *Window) ShapeData(sh st.Shape) {
	stw.History(fmt.Sprintf("%s\n", sh.String()))
	stw.History(fmt.Sprintf("A   = %10.4f [cm2]\n", sh.A()))
	stw.History(fmt.Sprintf("Asx = %10.4f [cm2]\n", sh.Asx()))
	stw.History(fmt.Sprintf("Asy = %10.4f [cm2]\n", sh.Asy()))
	stw.History(fmt.Sprintf("Ix  = %10.4f [cm4]\n", sh.Ix()))
	stw.History(fmt.Sprintf("Iy  = %10.4f [cm4]\n", sh.Iy()))
	stw.History(fmt.Sprintf("J   = %10.4f [cm4]\n", sh.J()))
	stw.History(fmt.Sprintf("Zx  = %10.4f [cm3]\n", sh.Zx()))
	stw.History(fmt.Sprintf("Zy  = %10.4f [cm3]\n", sh.Zy()))
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

func (stw *Window) TextBox(name string) *st.TextBox {
	if _, tok := stw.textBox[name]; !tok {
		stw.textBox[name] = st.NewTextBox(stw.font)
	}
	return stw.textBox[name]
}

func (stw *Window) TextBoxes() []*st.TextBox {
	rtn := make([]*st.TextBox, len(stw.textBox))
	i := 0
	for _, t := range stw.textBox {
		rtn[i] = t
		i++
	}
	return rtn
}

func (stw *Window) SetAngle(phi, theta float64) {
	view := st.CanvasCenterView(stw, []float64{phi, theta})
	st.Animate(stw, view)
}

func (stw *Window) SetPaperSize(name uint) {
	stw.papersize = name
}

func (stw *Window) PaperSize() uint {
	return stw.papersize
}

func (stw *Window) SetPeriod(string) {
}

func (stw *Window) Pivot() bool {
	return false
}

func (stw *Window) DrawPivot([]*st.Node, chan int, chan int) {
}

func (stw *Window) SetColorMode(mode uint) {
	stw.frame.Show.ColorMode = mode
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
	return stw.lastfig2command
}

func (stw *Window) SetLastFig2Command(c string) {
	stw.lastfig2command = c
}

func (stw *Window) ShowCenter() {
	stw.SetAngle(stw.frame.View.Angle[0], stw.frame.View.Angle[1])
}

func (stw *Window) EnableLabel(string) {
}

func (stw *Window) DisableLabel(string) {
}

func (stw *Window) SetLabel(k, v string) {
}
