package stgio

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	st "github.com/yofu/st/stlib"
)

type commandDraft struct {
	text  string
	caret int
}

// Alias maps are window-local; loading a user's file must not change Commands
// or the aliases of another window.
func newWindowAlias() *st.Alias {
	a := st.NewAlias()
	a.ClearCommandAlias()
	for name, command := range st.Commands {
		a.AddCommandAlias(name, command)
	}
	return a
}

// LoadAliases runs after SetHome and before Run, like LoadHistory.
func (w *Window) LoadAliases() error {
	if w.Home() == "" {
		return nil
	}
	err := w.ReadPgp(filepath.Join(w.Home(), ".st", "st.pgp"))
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

var pgpCounter = regexp.MustCompile(`@[0-9-]*`)

// ReadPgp is also called by :read pgp. Call with the model lock held once Run
// has started. Parse first so a read error leaves the current aliases intact.
func (w *Window) ReadPgp(filename string) error {
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	aliases := make(map[string]func(st.Commander) chan bool)
	var warnings []string
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 4096), 1024*1024)
	line := 0
	for scanner.Scan() {
		line++
		text := scanner.Text()
		if line == 1 {
			text = strings.TrimPrefix(text, "\ufeff")
		}
		if !utf8.ValidString(text) {
			return fmt.Errorf("%s:%d: UTF-8ではありません", filename, line)
		}
		text = strings.TrimSpace(text)
		if text == "" || strings.HasPrefix(text, "#") {
			continue
		}
		words := strings.Fields(text)
		if len(words) < 2 {
			continue
		}
		name := strings.ToUpper(words[0])
		if command, ok := st.Commands[strings.ToUpper(words[1])]; ok {
			aliases[name] = command
			continue
		}
		if !strings.HasPrefix(words[1], ":") && !strings.HasPrefix(words[1], "'") {
			warnings = append(warnings, fmt.Sprintf("%s:%d: %s: Gio未対応のコマンド %s", filename, line, name, words[1]))
			continue
		}
		command := strings.Join(words[1:], " ")
		if pos := strings.IndexByte(command, '_'); pos >= 0 {
			draft := commandDraft{text: strings.ReplaceAll(command, "_", ""), caret: utf8.RuneCountInString(command[:pos])}
			aliases[name] = func(st.Commander) chan bool {
				copy := draft
				w.commandDraft.Store(&copy)
				w.QueueRedrawAll()
				return nil
			}
			continue
		}
		token := ""
		value := 0
		// IUP increments counters only for Ex aliases; Fig2 aliases stay literal.
		if strings.HasPrefix(command, ":") {
			token = pgpCounter.FindString(command)
			if token != "" {
				value, _ = strconv.Atoi(token[1:])
			}
		}
		aliases[name] = func(host st.Commander) chan bool {
			current := command
			if token != "" {
				current = strings.ReplaceAll(current, token, strconv.Itoa(value))
				value++
			}
			var err error
			if strings.HasPrefix(current, ":") {
				if ew, ok := host.(st.ExModer); ok {
					err = st.ExMode(ew, current)
				}
			} else {
				if fw, ok := host.(st.Fig2Moder); ok {
					err = st.Fig2Mode(fw, current)
				}
			}
			if err != nil {
				st.ErrorMessage(host, err, st.ERROR)
			}
			return nil
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	w.ClearCommandAlias()
	for name, command := range st.Commands {
		w.AddCommandAlias(name, command)
	}
	for name, command := range aliases {
		w.AddCommandAlias(name, command)
	}
	for _, warning := range warnings {
		w.History(warning)
	}
	w.History(fmt.Sprintf("st.pgp: %d個のエイリアスを読み込みました (%s)", len(aliases), filename))
	return nil
}
