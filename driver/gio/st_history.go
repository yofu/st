package stgio

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	st "github.com/yofu/st/stlib"
)

const commandHistoryLimit = 100

// LoadHistory is called before Run, so startup data never races with widgets.
// File formats and ordering match IUP: newest first, UTF-8, one entry per line.
func (w *Window) LoadHistory(directory string) error {
	w.historyDirectory = directory
	commands, commandErr := readHistoryLines(filepath.Join(directory, "history.dat"), commandHistoryLimit)
	models, recentErr := readHistoryLines(filepath.Join(directory, "recent.dat"), 3)
	w.historyReadError = commandErr
	w.recentReadError = recentErr
	w.ui.historyMu.Lock()
	w.ui.commandHistory = nil
	// The UI stores oldest first for navigation; the disk format is newest first.
	for i := len(commands) - 1; i >= 0; i-- {
		w.addUICommandHistoryLocked(commands[i])
	}
	w.ui.historyIndex = -1
	w.ui.historyMu.Unlock()
	r := st.NewRecentFiles(3)
	for i := len(models) - 1; i >= 0; i-- {
		r.AddRecent(models[i])
	}
	w.recentMu.Lock()
	w.RecentFiles = r
	w.recentMu.Unlock()
	if len(models) > 0 {
		st.ShowRecent(w)
	}
	return errors.Join(commandErr, recentErr)
}

func readHistoryLines(path string, limit int) ([]string, error) {
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()
	s := bufio.NewScanner(f)
	s.Buffer(make([]byte, 4096), 1024*1024)
	lines := []string{}
	for s.Scan() {
		line := s.Text()
		if len(lines) == 0 {
			line = strings.TrimPrefix(line, "\ufeff")
		}
		if line == "" {
			continue
		}
		if !utf8.ValidString(line) {
			return nil, fmt.Errorf("%s: UTF-8ではない履歴です", path)
		}
		lines = append(lines, line)
		if len(lines) >= limit {
			break
		}
	}
	return lines, s.Err()
}

// Replace atomically in the same directory. A failed write keeps the old file.
func writeHistoryLines(path string, lines []string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".st-history-*")
	if err != nil {
		return err
	}
	temporary := f.Name()
	defer os.Remove(temporary)
	for _, line := range lines {
		if strings.ContainsAny(line, "\r\n") {
			f.Close()
			return fmt.Errorf("history entry contains a newline")
		}
		if _, err = f.WriteString(line + "\r\n"); err != nil {
			f.Close()
			return err
		}
	}
	if err = f.Close(); err != nil {
		return err
	}
	return os.Rename(temporary, path)
}

func (w *Window) addUICommandHistoryLocked(command string) {
	u := w.ui
	entries := u.commandHistory[:0]
	for _, old := range u.commandHistory {
		if old != command {
			entries = append(entries, old)
		}
	}
	u.commandHistory = append(entries, command)
	if len(u.commandHistory) > commandHistoryLimit {
		u.commandHistory = u.commandHistory[len(u.commandHistory)-commandHistoryLimit:]
	}
}

func (w *Window) SaveCommandHistory(path string) error {
	if path == "" {
		if w.historyReadError != nil {
			return fmt.Errorf("history.dat was not loaded; preserving existing file: %w", w.historyReadError)
		}
		if w.historyDirectory == "" {
			return nil
		}
		path = filepath.Join(w.historyDirectory, "history.dat")
	}
	w.persistenceMu.Lock()
	defer w.persistenceMu.Unlock()
	w.ui.historyMu.Lock()
	lines := make([]string, 0, len(w.ui.commandHistory))
	for i := len(w.ui.commandHistory) - 1; i >= 0; i-- {
		lines = append(lines, w.ui.commandHistory[i])
	}
	w.ui.historyMu.Unlock()
	return writeHistoryLines(path, lines)
}

func (w *Window) Recent() []string {
	w.recentMu.Lock()
	defer w.recentMu.Unlock()
	return append([]string(nil), w.RecentFiles.Recent()...)
}

func (w *Window) AddRecent(path string) {
	w.recentMu.Lock()
	w.RecentFiles.AddRecent(path)
	w.recentMu.Unlock()
	if err := w.SaveRecent(); err != nil {
		w.History("recent.dat: " + err.Error())
	}
}

func (w *Window) SaveRecent() error {
	if w.recentReadError != nil {
		return fmt.Errorf("recent.dat was not loaded; preserving existing file: %w", w.recentReadError)
	}
	if w.historyDirectory == "" {
		return nil
	}
	w.persistenceMu.Lock()
	defer w.persistenceMu.Unlock()
	return writeHistoryLines(filepath.Join(w.historyDirectory, "recent.dat"), w.Recent())
}
