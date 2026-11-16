package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"

	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"

	"golang.org/x/text/encoding/japanese"
)

const fig2PathSchema = `{"type":"object","required":["path"],"properties":{"path":{"type":"string"}},"additionalProperties":false}`
const fig2PreviewSchema = `{"type":"object","properties":{
  "template":{"type":"string","description":"Root-relative existing .fig2 template."},
  "content":{"type":"string","description":"Complete Fig2 text, including advanced constructs such as for/endfor or multiple."},
  "replace":{"type":"array","items":{"type":"object","required":["find","text"],"properties":{"find":{"type":"string","minLength":1},"text":{"type":"string"},"all":{"type":"boolean"}},"additionalProperties":false}},
  "common":{"type":"array","items":{"type":"string"}},
  "pages":{"type":"array","items":{"type":"object","properties":{"paper":{"enum":["a4tate","a4yoko","a3tate","a3yoko"]},"title":{"type":"string"},"page_title":{"type":"string"},"commands":{"type":"array","items":{"type":"string"}}},"additionalProperties":false}}
},"additionalProperties":false}`

var fig2Catalog = []tool{
	{"list_fig2", "List .fig2 files in a root-relative directory; omit directory to list the model root.", json.RawMessage(`{"type":"object","properties":{"directory":{"type":"string"}},"additionalProperties":false}`)},
	{"read_fig2", "Read an existing .fig2 file as Unicode, including legacy Windows Japanese files.", json.RawMessage(fig2PathSchema)},
	{"preview_fig2", "Draft a Fig2 file from complete text, structured pages, or a copied template with literal replacements and optional appended pages. Returns the content and a draft ID before saving.", json.RawMessage(fig2PreviewSchema)},
	{"save_fig2_as", "Save the reviewed draft as a new .fig2 file within the model root. Never overwrites an existing file.", json.RawMessage(`{"type":"object","required":["path","draft_id"],"properties":{"path":{"type":"string"},"draft_id":{"type":"integer","minimum":1}},"additionalProperties":false}`)},
}

type fig2Draft struct {
	id   uint64
	data []byte
}
type fig2Page struct {
	Paper     string   `json:"paper"`
	Title     string   `json:"title"`
	PageTitle string   `json:"page_title"`
	Commands  []string `json:"commands"`
}
type fig2Replace struct {
	Find string `json:"find"`
	Text string `json:"text"`
	All  bool   `json:"all"`
}
type fig2Preview struct {
	Template string        `json:"template"`
	Content  *string       `json:"content"`
	Replace  []fig2Replace `json:"replace"`
	Common   []string      `json:"common"`
	Pages    []fig2Page    `json:"pages"`
}

func fig2Path(path string) (string, error) {
	if !filepath.IsLocal(path) || !strings.EqualFold(filepath.Ext(path), ".fig2") {
		return "", fmt.Errorf("expected a .fig2 path within the model root")
	}
	return filepath.Clean(path), nil
}

func (s *modelSession) readFig2(path string) ([]byte, error) {
	name, err := fig2Path(path)
	if err != nil {
		return nil, err
	}
	f, err := s.root.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	const maxFig2 = 256 << 10
	if !info.Mode().IsRegular() || info.Size() > maxFig2 {
		return nil, fmt.Errorf("expected a regular Fig2 file of up to 256 KiB")
	}
	data, err := io.ReadAll(io.LimitReader(f, maxFig2+1))
	if len(data) > maxFig2 {
		return nil, fmt.Errorf("Fig2 exceeds 256 KiB")
	}
	return data, err
}

func fig2Text(data []byte) (text, encoding string, err error) {
	if bytes.HasPrefix(data, []byte{0xef, 0xbb, 0xbf}) {
		return string(data[3:]), "utf-8-bom", nil
	}
	if utf8.Valid(data) {
		return string(data), "utf-8", nil
	}
	text, err = japanese.ShiftJIS.NewDecoder().String(string(data))
	if err != nil {
		return "", "", err
	}
	return text, "cp932", nil
}

func fig2Paper(paper string) bool {
	return slices.Contains([]string{"a4tate", "a4yoko", "a3tate", "a3yoko"}, strings.ToLower(paper))
}

var fig2For = regexp.MustCompile(`^for\s+\S+\s+in\s+\S`)

// Check the outer structure used by writepdf. Page command arguments and
// project-specific grid-line commands are interpreted by fig2.exe.
func checkFig2(text string) (int, error) {
	pages := 0
	inLoop := false
	for i, line := range strings.Split(text, "\n") {
		words := strings.Fields(line)
		if len(words) == 0 {
			continue
		}
		switch strings.ToLower(words[0]) {
		case "page", "figure", "figure_v", "figure_h":
			if len(words) < 2 || !fig2Paper(words[1]) && !strings.HasPrefix(words[1], "$") {
				return 0, fmt.Errorf("line %d: specify a4tate, a4yoko, a3tate or a3yoko", i+1)
			}
			pages++
		case "template":
			if len(words) < 2 {
				return 0, fmt.Errorf("line %d: missing template name", i+1)
			}
			pages++
		case "for":
			if inLoop || !fig2For.MatchString(strings.TrimSpace(line)) {
				return 0, fmt.Errorf("line %d: invalid or nested Fig2 for loop", i+1)
			}
			inLoop = true
		case "endfor":
			if !inLoop {
				return 0, fmt.Errorf("line %d: endfor without for", i+1)
			}
			inLoop = false
		}
	}
	if inLoop {
		return 0, fmt.Errorf("missing endfor")
	}
	if pages == 0 {
		return 0, fmt.Errorf("add at least one page or figure")
	}
	return pages, nil
}

func buildFig2(o fig2Preview, base string) (string, error) {
	if o.Template != "" && o.Content != nil {
		return "", fmt.Errorf("choose a template or complete content")
	}
	if len(o.Common) > 0 && (o.Template != "" || o.Content != nil) {
		return "", fmt.Errorf("common is for structured creation; use replace to edit a template")
	}
	if len(o.Pages) > 100 || len(o.Replace) > 100 {
		return "", fmt.Errorf("at most 100 pages or replacements per draft")
	}
	base = strings.ReplaceAll(base, "\r\n", "\n")
	if o.Content != nil {
		base = strings.ReplaceAll(*o.Content, "\r\n", "\n")
	}
	for _, edit := range o.Replace {
		find := strings.ReplaceAll(edit.Find, "\r\n", "\n")
		text := strings.ReplaceAll(edit.Text, "\r\n", "\n")
		if find == "" {
			return "", fmt.Errorf("replacement search text is empty")
		}
		n := strings.Count(base, find)
		if n == 0 || n > 1 && !edit.All {
			return "", fmt.Errorf("replacement text matched %d times; use a unique match or all", n)
		}
		base = strings.ReplaceAll(base, find, text)
		if len(base) > 256<<10 {
			return "", fmt.Errorf("Fig2 exceeds 256 KiB")
		}
	}
	line := func(value string) error {
		if strings.ContainsAny(value, "\r\n\x00") {
			return fmt.Errorf("use one Fig2 directive per structured line")
		}
		return nil
	}
	var b strings.Builder
	b.WriteString(base)
	if len(o.Common) > 0 {
		b.WriteString("common\n")
		for _, cmd := range o.Common {
			if err := line(cmd); err != nil {
				return "", err
			}
			if cmd = strings.TrimSpace(cmd); cmd != "" {
				fmt.Fprintln(&b, "    "+cmd)
			}
		}
	}
	for _, page := range o.Pages {
		paper := page.Paper
		if paper == "" {
			paper = "a4tate"
		}
		if !fig2Paper(paper) {
			return "", fmt.Errorf("unknown paper %q", paper)
		}
		for _, title := range []string{page.PageTitle, page.Title} {
			if err := line(title); err != nil {
				return "", err
			}
		}
		fmt.Fprintf(&b, "\npage %s\n", paper)
		if page.PageTitle != "" {
			fmt.Fprintln(&b, "pagetitle "+page.PageTitle)
		}
		if page.Title != "" {
			fmt.Fprintln(&b, "title "+page.Title)
		}
		for _, cmd := range page.Commands {
			if err := line(cmd); err != nil {
				return "", err
			}
			if cmd = strings.TrimSpace(cmd); cmd != "" {
				fmt.Fprintln(&b, "    "+cmd)
			}
		}
	}
	return strings.TrimLeft(b.String(), "\n"), nil
}

func (s *modelSession) fig2Tool(name string, raw json.RawMessage) (any, error) {
	switch name {
	case "list_fig2":
		var o struct {
			Directory string `json:"directory"`
		}
		if err := decode(raw, &o); err != nil {
			return nil, err
		}
		if o.Directory == "" {
			o.Directory = "."
		}
		if !filepath.IsLocal(o.Directory) {
			return nil, fmt.Errorf("expected a directory within the model root")
		}
		entries, err := fs.ReadDir(s.root.FS(), filepath.ToSlash(filepath.Clean(o.Directory)))
		if err != nil {
			return nil, err
		}
		paths := make([]string, 0)
		for _, entry := range entries {
			if !entry.IsDir() && strings.EqualFold(filepath.Ext(entry.Name()), ".fig2") {
				paths = append(paths, filepath.Join(o.Directory, entry.Name()))
			}
		}
		return map[string]any{"paths": paths}, nil
	case "read_fig2":
		var o struct {
			Path string `json:"path"`
		}
		if err := decode(raw, &o); err != nil {
			return nil, err
		}
		data, err := s.readFig2(o.Path)
		if err != nil {
			return nil, err
		}
		text, enc, err := fig2Text(data)
		if err != nil {
			return nil, err
		}
		return map[string]any{"path": o.Path, "content": text, "encoding": enc}, nil
	case "preview_fig2":
		var o fig2Preview
		if err := decode(raw, &o); err != nil {
			return nil, err
		}
		var data []byte
		base := ""
		if o.Template != "" {
			var err error
			data, err = s.readFig2(o.Template)
			if err != nil {
				return nil, err
			}
			base, _, err = fig2Text(data)
			if err != nil {
				return nil, err
			}
		}
		text, err := buildFig2(o, base)
		if err != nil {
			return nil, err
		}
		if len(text) > 256<<10 || strings.ContainsRune(text, 0) {
			return nil, fmt.Errorf("invalid or oversized Fig2 text")
		}
		count, err := checkFig2(text)
		if err != nil {
			return nil, err
		}
		encoding := "cp932"
		if data != nil && text == strings.ReplaceAll(base, "\r\n", "\n") && !bytes.HasPrefix(data, []byte{0xef, 0xbb, 0xbf}) && !utf8.Valid(data) {
			// Reuse unmodified Windows text byte-for-byte.
		} else {
			encoded, err := japanese.ShiftJIS.NewEncoder().String(strings.ReplaceAll(strings.TrimRight(text, "\n")+"\n", "\n", "\r\n"))
			if err != nil {
				return nil, fmt.Errorf("text cannot be written in Windows Japanese encoding: %w", err)
			}
			data = []byte(encoded)
		}
		s.nextFig2++
		s.fig2 = &fig2Draft{id: s.nextFig2, data: data}
		return map[string]any{"draft_id": s.nextFig2, "content": text, "encoding": encoding, "figure_directives": count}, nil
	case "save_fig2_as":
		var o struct {
			Path  string `json:"path"`
			Draft uint64 `json:"draft_id"`
		}
		if err := decode(raw, &o); err != nil {
			return nil, err
		}
		path, err := fig2Path(o.Path)
		if err != nil {
			return nil, err
		}
		if s.fig2 == nil || o.Draft != s.fig2.id {
			return nil, fmt.Errorf("use the current draft_id returned by preview_fig2")
		}
		if err := writeNew(s.root, path, s.fig2.data); err != nil {
			return nil, err
		}
		return map[string]any{"path": path, "draft_id": o.Draft}, nil
	}
	return nil, fmt.Errorf("unknown Fig2 tool")
}

func init() { catalog = append(catalog, fig2Catalog...) }
