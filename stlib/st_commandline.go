package st

import (
	"fmt"
	"strings"
)

type CommandLine struct {
	words []string
	completing bool
	completes []string
	completepos int
}

func NewCommandLine() *CommandLine {
	return &CommandLine{
		words: make([]string, 0),
		completing: false,
		completes:  make([]string, 0),
		completepos: 0,
	}
}

func (c *CommandLine) ClearCommandLine() {
	c.words = make([]string, 0)
	c.completing = false
	c.completes = make([]string, 0)
	c.completepos = 0
}

func (c *CommandLine) CommandLineString() string {
	if c.completing {
		return strings.Join(append(c.words, c.completes[c.completepos]), " ")
	} else {
		return strings.Join(c.words, " ")
	}
}

func (c *CommandLine) PopLastWord() string {
	if c.completing {
		c.EndCompletion()
	}
	if len(c.words) == 0 {
		return ""
	}
	rtn := c.words[len(c.words) - 1]
	c.words = c.words[:len(c.words) - 1]
	return rtn
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
			c.words[len(c.words) - 1] = fmt.Sprintf("%s%s", c.words[len(c.words) - 1], s)
		}
	}
}

func (c *CommandLine) BackspaceCommandLine() {
	if c.completing {
		c.EndCompletion()
	}
	if len(c.words) == 0 {
		return
	} else {
		if c.words[len(c.words) - 1] == "" {
			c.words = c.words[:len(c.words) - 1]
		} else {
			c.words[len(c.words) - 1] = c.words[len(c.words) - 1][:len(c.words[len(c.words) - 1]) - 1]
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
	c.completepos = 0
	c.completes = lis
	c.completing = true
}

func (c *CommandLine) EndCompletion() {
	c.completing = false
	c.words = append(c.words, c.completes[c.completepos])
}
