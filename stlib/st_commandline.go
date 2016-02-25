package st

import (
	"fmt"
	"github.com/yofu/complete"
	"strings"
)

type CommandLine struct {
	words       []string
	completing  bool
	completes   []string
	completepos int
	comp        *complete.Complete
}

func NewCommandLine() *CommandLine {
	return &CommandLine{
		words:       []string{""},
		completing:  false,
		completes:   make([]string, 0),
		completepos: 0,
		comp:        nil,
	}
}

func (c *CommandLine) ClearCommandLine() {
	c.words = []string{""}
	c.completing = false
	c.completes = make([]string, 0)
	c.completepos = 0
}

func (c *CommandLine) CommandLineString() string {
	if c.completing {
		return strings.Join(append(c.words[:len(c.words)-1], c.completes[c.completepos]), " ")
	} else {
		return strings.Join(c.words, " ")
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
}

func (c *CommandLine) TypeCommandLine(s string) {
	if c.completing {
		c.EndCompletion()
	}
	if len(c.words) == 0 {
		c.words = append(c.words, s)
	} else {
		if s == " " {
			c.words = append(c.words, "")
		} else {
			c.words[len(c.words)-1] = fmt.Sprintf("%s%s", c.words[len(c.words)-1], s)
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
	}
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
