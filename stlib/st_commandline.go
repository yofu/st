package st

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/yofu/complete"
)

const (
	historysize = 100
)

var (
	historyfn = filepath.Join(os.Getenv("HOME"), ".st/history.dat")
)

type CommandLine struct {
	words       []string
	position    int
	completing  bool
	completes   []string
	completepos int
	comp        *complete.Complete
	history     []string
	historypos  int
}

func NewCommandLine() *CommandLine {
	return &CommandLine{
		words:       []string{""},
		position:    0,
		completing:  false,
		completes:   make([]string, 0),
		completepos: 0,
		comp:        nil,
		history:     make([]string, historysize),
		historypos:  0,
	}
}

func (c *CommandLine) ClearCommandLine() {
	c.words = []string{""}
	c.position = 0
	c.completing = false
	c.completes = make([]string, 0)
	c.completepos = 0
}

func (c *CommandLine) CommandLineStringIsEmpty() bool {
	if c.completing {
		return false
	}
	for _, s := range c.words {
		if len(s) > 0 {
			return false
		}
	}
	return true
}

func (c *CommandLine) CommandLineString() string {
	if c.completing {
		return strings.Join(append(c.words[:len(c.words)-1], c.completes[c.completepos]), " ")
	} else {
		return strings.Join(c.words, " ")
	}
}

func (c *CommandLine) CommandLineStringWithPosition() string {
	if c.completing {
		return strings.Join(append(c.words[:len(c.words)-1], c.completes[c.completepos]), " ") + "|"
	} else {
		str := strings.Join(c.words, " ")
		if str == "" {
			return ""
		} else {
			return fmt.Sprintf("%s|%s", str[:c.position], str[c.position:])
		}
	}
}

func (c *CommandLine) LastWord() string {
	if c.completing {
		c.EndCompletion()
	}
	if len(c.words) == 0 {
		return ""
	}
	return c.words[len(c.words)-1]
}

func (c *CommandLine) SetCommandLineString(s string) {
	c.ClearCommandLine()
	c.words = strings.Split(s, " ")
	c.SeekLast()
}

func (c *CommandLine) TypeCommandLine(s string) {
	if c.completing {
		c.EndCompletion()
	}
	if len(c.words) == 0 {
		c.words = append(c.words, s)
	} else {
		if c.AtLast() {
			if s == " " {
				c.words = append(c.words, "")
			} else {
				c.words[len(c.words)-1] = fmt.Sprintf("%s%s", c.words[len(c.words)-1], s)
			}
			c.position++
		} else {
			pos := c.position
			for i, w := range c.words {
				l := len(w)
				if pos <= l {
					c.words[i] = fmt.Sprintf("%s%s%s", c.words[i][:pos], s, c.words[i][pos:])
					c.position++
					break
				}
				pos -= l
				pos--
			}
		}
	}
}

func (c *CommandLine) BackspaceCommandLine() {
	if c.completing {
		c.EndCompletion()
	}
	if len(c.words) == 0 {
		return
	}
	if c.position == 0 {
		return
	}
	if c.AtLast() {
		c.position--
		if c.words[len(c.words)-1] == "" {
			if len(c.words) > 1 {
				c.words = c.words[:len(c.words)-1]
			}
		} else {
			c.words[len(c.words)-1] = c.words[len(c.words)-1][:len(c.words[len(c.words)-1])-1]
			if len(c.words) == 1 && len(c.words[0]) == 0 {
				c.ClearCommandLine()
			}
		}
	} else {
		pos := c.position
		c.position--
		for i, w := range c.words {
			l := len(w)
			if pos <= l {
				c.words[i] = fmt.Sprintf("%s%s", c.words[i][:pos-1], c.words[i][pos:])
				return
			}
			pos -= l
			pos--
			if pos == 0 {
				c.words[i] = fmt.Sprintf("%s%s", c.words[i], c.words[i+1])
				for j := i + 1; j < len(c.words)-1; j++ {
					c.words[j] = c.words[j+1]
				}
				c.words = c.words[:len(c.words)-1]
				return
			}
		}
	}
}

func (c *CommandLine) SeekHead() {
	if c.completing {
		c.EndCompletion()
	}
	c.position = 0
}

func (c *CommandLine) SeekForward() {
	if c.completing {
		c.EndCompletion()
	}
	if c.AtLast() {
		return
	}
	c.position++
}

func (c *CommandLine) SeekBackward() {
	if c.completing {
		c.EndCompletion()
	}
	if c.position == 0 {
		return
	}
	c.position--
}

func (c *CommandLine) SeekLast() {
	if c.completing {
		c.EndCompletion()
	}
	pos := -1
	for _, w := range c.words {
		pos++
		pos += len(w)
	}
	c.position = pos
}

func (c *CommandLine) AtLast() bool {
	pos := -1
	for _, w := range c.words {
		pos++
		pos += len(w)
	}
	return c.position == pos
}

func (c *CommandLine) PrevComplete() {
	if c.completes == nil || len(c.completes) == 0 {
		return
	}
	c.completepos--
	if c.completepos < 0 {
		c.completepos = len(c.completes) - 1
	}
}

func (c *CommandLine) NextComplete() {
	if c.completes == nil || len(c.completes) == 0 {
		return
	}
	c.completepos++
	if c.completepos >= len(c.completes) {
		c.completepos = 0
	}
}

func (c *CommandLine) StartCompletion(lis []string) {
	if len(lis) == 0 {
		return
	}
	c.completepos = 0
	c.completes = lis
	c.completing = true
}

func (c *CommandLine) EndCompletion() {
	c.completing = false
	if c.completes != nil && len(c.completes) > c.completepos {
		c.words[len(c.words)-1] = c.completes[c.completepos]
		c.SeekLast()
	}
	c.completes = make([]string, 0)
	c.completepos = 0
}

func (c *CommandLine) SetComplete(comp *complete.Complete) {
	c.comp = comp
}

func (c *CommandLine) ContextComplete() ([]string, bool) {
	str := c.CommandLineString()
	if c.comp == nil || c.comp.Context(str) == complete.FileName {
		return nil, false
	}
	lis := c.comp.CompleteWord(str)
	if len(lis) == 0 {
		return nil, false
	} else {
		return lis, true
	}
}

func (c *CommandLine) AddCommandHistory(str string) {
	tmp := make([]string, historysize)
	tmp[0] = str
	ind := 0
	for i := 0; i < historysize-1; i++ {
		if c.history[i] == str {
			ind++
			if i+ind >= historysize {
				break
			}
			continue
		}
		tmp[i+1] = c.history[i+ind]
	}
	c.history = tmp
	c.historypos = -1
}

func (c *CommandLine) PrevCommandHistory(str string) {
	last := c.historypos
	for {
		c.historypos++
		if c.historypos >= historysize {
			c.historypos = last
			return
		}
		if strings.HasPrefix(c.history[c.historypos], str) {
			c.SetCommandLineString(c.history[c.historypos])
			return
		}
	}
}

func (c *CommandLine) NextCommandHistory(str string) {
	for {
		c.historypos--
		if c.historypos < 0 {
			c.historypos = 0
			return
		}
		if strings.HasPrefix(c.history[c.historypos], str) {
			c.SetCommandLineString(c.history[c.historypos])
			return
		}
	}
}

func (c *CommandLine) ReadCommandHistory(fn string) error {
	if fn == "" {
		fn = historyfn
	}
	if !FileExists(fn) {
		return fmt.Errorf("file doesn't exist")
	}
	tmp := make([]string, historysize)
	f, err := os.Open(fn)
	if err != nil {
		return err
	}
	s := bufio.NewScanner(f)
	num := 0
	for s.Scan() {
		if com := s.Text(); com != "" {
			tmp[num] = com
			num++
		}
	}
	if err := s.Err(); err != nil {
		return err
	}
	c.history = tmp
	return nil
}

func (c *CommandLine) SaveCommandHistory(fn string) error {
	if fn == "" {
		fn = historyfn
	}
	var otp bytes.Buffer
	for _, com := range c.history {
		if com != "" {
			otp.WriteString(com)
			otp.WriteString("\n")
		}
	}
	w, err := os.Create(fn)
	defer w.Close()
	if err != nil {
		return err
	}
	otp = AddCR(otp)
	otp.WriteTo(w)
	return nil
}
