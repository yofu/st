package stgio

import (
	"path/filepath"
	"sort"
	"strings"

	"github.com/yofu/complete"
	st "github.com/yofu/st/stlib"
)

func (w *Window) setCommandText(text string) {
	w.ui.command.SetText(text)
	w.ui.command.SetCaret(w.ui.command.Len(), w.ui.command.Len())
	w.ui.commandBefore = text
	if text == "" {
		w.ui.punctuationPending = false
		w.ui.spacePending = false
	}
}

// Apply IUP's character rules to committed edits, leaving IME composition and
// Gio's text editing/selection handling to widget.Editor.
func (w *Window) commandChanged() {
	u := w.ui
	text := u.command.Text()
	before := u.commandBefore
	if text == before {
		return
	} // Editor also reports programmatic SetText changes.
	if before == "" && (u.punctuationPending || len(text) == 1) {
		if strings.HasPrefix(text, ";") {
			text = ":" + text[1:]
		} else if strings.HasPrefix(text, ":") {
			text = ";" + text[1:]
		}
	}
	if before != "" && !strings.Contains(before, " ") && text == before+" " {
		text = expandCommandWord(before) + " "
	}
	if u.spacePending {
		if pos := strings.IndexByte(text, ' '); pos >= 0 {
			text = expandCommandWord(text[:pos]) + text[pos:]
		}
	}
	u.punctuationPending = false
	u.spacePending = false
	if text != u.command.Text() {
		w.setCommandText(text)
	} else {
		u.commandBefore = text
	}
	u.historyBrowsing = false
	u.lastCompletion = false
	u.completionCandidates = nil
}

func expandCommandWord(text string) string {
	if strings.HasPrefix(text, ":") {
		name, bang, usage, _ := st.ExModeComplete(text)
		text = ":" + name
		if bang {
			text += "!"
		}
		if usage {
			text += "?"
		}
	} else if strings.HasPrefix(text, "'") {
		name, usage, _ := st.Fig2KeywordComplete(text)
		text = "'" + name
		if usage {
			text += "?"
		}
	}
	return text
}

func (w *Window) browseHistory(back bool) {
	u := w.ui
	u.historyMu.Lock()
	defer u.historyMu.Unlock()
	if !u.historyBrowsing {
		u.draft = u.command.Text()
		u.historyIndex = len(u.commandHistory)
		u.historyBrowsing = true
	}
	if back {
		for i := u.historyIndex - 1; i >= 0; i-- {
			if strings.HasPrefix(u.commandHistory[i], u.draft) {
				u.historyIndex = i
				w.setCommandText(u.commandHistory[i])
				return
			}
		}
	} else {
		for i := u.historyIndex + 1; i < len(u.commandHistory); i++ {
			if strings.HasPrefix(u.commandHistory[i], u.draft) {
				u.historyIndex = i
				w.setCommandText(u.commandHistory[i])
				return
			}
		}
		u.historyIndex = len(u.commandHistory)
		w.setCommandText(u.draft)
	}
	u.lastCompletion = false
}

func (w *Window) completeCommand() { w.completeCommandDirection(1) }
func (w *Window) completeCommandDirection(direction int) {
	u := w.ui
	current := u.command.Text()
	if !u.lastCompletion || len(u.completionCandidates) == 0 || current != u.completionInput {
		u.completionCandidates = w.commandCandidates(current)
		u.completionIndex = 0
	} else {
		u.completionIndex = (u.completionIndex + direction + len(u.completionCandidates)) % len(u.completionCandidates)
	}
	if len(u.completionCandidates) > 0 {
		text := u.completionCandidates[u.completionIndex]
		w.setCommandText(text)
		u.completionInput = text
	}
	u.lastCompletion = true
	u.historyBrowsing = false
}

func (w *Window) commandCandidates(text string) []string {
	u := w.ui
	if !strings.Contains(text, " ") && (strings.HasPrefix(text, ":") || strings.HasPrefix(text, "'")) {
		pool := commandNames(st.ExAbbrev, ":")
		if strings.HasPrefix(text, "'") {
			pool = commandNames(st.Fig2Abbrev, "'")
		}
		for _, name := range []string{":timehistory", ":arclm001", ":arclm101", ":stop"} {
			pool[name] = true
		}
		result := []string{}
		for name := range pool {
			if strings.HasPrefix(name, text) {
				result = append(result, name)
			}
		}
		sort.Strings(result)
		return result
	}
	words := strings.Split(text, " ")
	last := len(words) - 1
	var spec *complete.Complete
	if strings.HasPrefix(text, ":") {
		_, _, _, spec = st.ExModeComplete(words[0])
	} else if strings.HasPrefix(text, "'") {
		_, _, spec = st.Fig2KeywordComplete(words[0])
	}
	if spec != nil && spec.Context(text) != complete.FileName {
		// Chdir mutates Complete; copy the descriptor to keep global tables immutable.
		local := *spec
		local.Chdir(u.values["CWD"])
		return local.Complete(text)
	}
	percent := u.values["PATH"]
	if percent == "" {
		percent = filepath.Join(u.values["CWD"], "_")
	}
	candidates := st.CompleteFileName(words[last], percent, w.Recent())
	result := []string{}
	seen := make(map[string]bool)
	for _, path := range candidates {
		words[last] = path
		candidate := strings.Join(words, " ")
		if !seen[candidate] {
			result = append(result, candidate)
			seen[candidate] = true
		}
	}
	return result
}
