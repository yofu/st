package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
)

type tool struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Input       json.RawMessage `json:"inputSchema"`
}

const pathSchema = `{"type":"object","required":["path"],"properties":{"path":{"type":"string"}},"additionalProperties":false}`
const querySchema = `{"type":"object","properties":{"ids":{"type":"array","items":{"type":"integer"},"maxItems":200},"after":{"type":"integer"},"limit":{"type":"integer","minimum":1,"maximum":200}},"additionalProperties":false}`

const jobSchema = `{"type":"object","required":["job_id"],"properties":{"job_id":{"type":"integer","minimum":1}},"additionalProperties":false}`

var catalog = []tool{
	{"list_model_editors", "List available interactive, legacy GUI and model editing ex commands.", json.RawMessage(`{"type":"object","additionalProperties":false}`)},
	{"preview_gui_edits", "Run ordered GUI model editing commands on a private copy of the current model or pending proposal. Provide selections by ID and interactive inputs in GUI order. Returns before/after INP records and a proposal ID for save or analysis.", json.RawMessage(guiEditSchema)},
	{"start_analysis_comparison", "Start an asynchronous comparison of the current model and pending proposal using linear static, vibration, buckling (Bclng001), or material nonlinear static analysis (Arclm101). Both runs linear static and vibration. Arclm101 requires node_ids, steps and increment, with optional preload_l for X/Y. Set preview_dir to publish converged edited-model steps for the IUP display.", json.RawMessage(`{"type":"object","required":["proposal_id"],"properties":{"proposal_id":{"type":"integer","minimum":1},"analysis":{"enum":["static","vibration","both","buckling","arclm101"]},"period":{"enum":["L","X","Y"]},"node_ids":{"type":"array","items":{"type":"integer","minimum":1},"maxItems":200,"uniqueItems":true},"modes":{"type":"integer","minimum":1,"maximum":100},"steps":{"type":"integer","minimum":1,"maximum":10000},"increment":{"type":"number"},"preload_l":{"type":"boolean"},"preview_dir":{"type":"string","minLength":1},"preview_every":{"type":"integer","minimum":1,"maximum":10000}},"additionalProperties":false}`)},
	{"analysis_status", "Return the selected job state, current before or after phase, progress detail and elapsed seconds.", json.RawMessage(jobSchema)},
	{"cancel_analysis", "Stop the selected analysis worker and discard partial results.", json.RawMessage(jobSchema)},
	{"analysis_results", "Return the completed job's before and after results and signed differences, with modal values matched by ascending rank and Arclm101 curves by load step.", json.RawMessage(jobSchema)},
	{"open_model", "Open an existing INP model by a path relative to the configured root. Replaces the current session model and clears its pending proposal.", json.RawMessage(pathSchema)},
	{"query_nodes", "List nodes by ID or by ascending ID page. Coordinates are X/Y/Z in m. Fixed and load vectors follow X,Y,Z,RX,RY,RZ. Load uses the input model's tf and tf*m units.", json.RawMessage(querySchema)},
	{"query_elements", "List elements by ID or by ascending ID page, with element type, section number, and connectivity.", json.RawMessage(querySchema)},
	{"query_sections", "List available sections by ID or by ascending ID page, with name and line or plate compatibility.", json.RawMessage(querySchema)},
	{"preview_edits", "Return all before/after values and a proposal ID. An element edit reassigns an existing section. A node edit replaces the entire fixed and/or load vector. No file is written.", json.RawMessage(`{"type":"object","required":["edits"],"properties":{"edits":{"type":"array","minItems":1,"maxItems":100,"items":{"type":"object","properties":{"node":{"type":"integer","minimum":1},"element":{"type":"integer","minimum":1},"section":{"type":"integer","minimum":1},"fixed":{"type":"array","minItems":6,"maxItems":6,"items":{"type":"boolean"}},"load":{"type":"array","minItems":6,"maxItems":6,"items":{"type":"number"}}},"additionalProperties":false}}},"additionalProperties":false}`)},
	{"save_model_as", "Save the current reviewed proposal to a new root-relative .inp file. An existing file is never replaced.", json.RawMessage(`{"type":"object","required":["proposal_id","path"],"properties":{"proposal_id":{"type":"integer","minimum":1},"path":{"type":"string"}},"additionalProperties":false}`)},
}

func decode(raw json.RawMessage, dst any) error {
	if len(raw) == 0 {
		raw = json.RawMessage("{}")
	}
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || raw[0] != '{' {
		return fmt.Errorf("expected a JSON object")
	}
	// None of the tool schemas accepts null. Go otherwise silently decodes
	// null as zero for scalar array entries, which can change a load or restraint.
	tokens := json.NewDecoder(bytes.NewReader(raw))
	for {
		value, err := tokens.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if value == nil {
			return fmt.Errorf("null is not valid in tool arguments")
		}
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	return d.Decode(dst)
}

func main() {
	dir := flag.String("root", ".", "directory containing models available to the MCP client")
	worker := flag.String("analysis-worker", "", "internal analysis worker directory")
	flag.Parse()
	// ST's interactive parser may write diagnostics to os.Stdout.
	protocol := os.Stdout
	os.Stdout = os.Stderr
	if *worker != "" {
		analysisWorkerMain(*worker, protocol)
		return
	}
	if err := run(*dir, os.Stdin, protocol); err != nil {
		fmt.Fprintln(os.Stderr, "stmcp:", err)
		os.Exit(1)
	}
}

func run(dir string, in io.Reader, out io.Writer) error {
	root, err := os.OpenRoot(dir)
	if err != nil {
		return err
	}
	defer root.Close()
	scratch, err := os.MkdirTemp("", "stmcp-parser-")
	if err != nil {
		return err
	}
	defer os.Remove(scratch)
	// The parser checks for ambient .conf sidecars by filename. Give it a
	// private placeholder path so only the requested INP feeds the model.
	session := modelSession{root: root, parsePath: filepath.Join(scratch, "model.inp")}
	defer session.closeAnalysis()
	scanner := bufio.NewScanner(in)
	scanner.Buffer(make([]byte, 4096), 1<<20)
	writer := json.NewEncoder(out)
	version := ""
	for scanner.Scan() {
		var req struct {
			Version string          `json:"jsonrpc"`
			ID      json.RawMessage `json:"id"`
			Method  string          `json:"method"`
			Params  json.RawMessage `json:"params"`
		}
		id := json.RawMessage("null")
		reply := func(result any, code int, message string) error {
			data := map[string]any{"jsonrpc": "2.0", "id": id}
			if code != 0 {
				data["error"] = map[string]any{"code": code, "message": message}
			} else {
				data["result"] = result
			}
			return writer.Encode(data)
		}
		line := scanner.Bytes()
		if !json.Valid(line) {
			if err := reply(nil, -32700, "Parse error"); err != nil {
				return err
			}
			continue
		}
		if err := json.Unmarshal(line, &req); err != nil || req.Version != "2.0" || req.Method == "" {
			if err := reply(nil, -32600, "Invalid request"); err != nil {
				return err
			}
			continue
		}
		if len(req.ID) == 0 {
			continue
		}
		id = req.ID
		var result any
		var code int
		var message string
		switch {
		case req.Method == "ping":
			result = map[string]any{}
		case req.Method == "initialize":
			var args struct {
				Protocol string `json:"protocolVersion"`
			}
			if err := json.Unmarshal(req.Params, &args); err != nil || args.Protocol == "" || version != "" {
				code, message = -32602, "Invalid initialization"
				break
			}
			version = args.Protocol
			if !slices.Contains([]string{"2024-11-05", "2025-03-26", "2025-06-18", "2025-11-25"}, version) {
				version = "2025-11-25"
			}
			result = map[string]any{"protocolVersion": version, "capabilities": map[string]any{"tools": map[string]any{}}, "serverInfo": map[string]string{"name": "stmcp", "version": "0.1.0"}}
		case version == "":
			code, message = -32000, "Initialize the MCP session first"
		case req.Method == "tools/list":
			result = map[string]any{"tools": catalog}
		case req.Method == "tools/call":
			var args struct {
				Name      string          `json:"name"`
				Arguments json.RawMessage `json:"arguments"`
			}
			// MCP clients can attach protocol metadata such as _meta.progressToken.
			// Each tool validates its own arguments separately.
			if err := json.Unmarshal(req.Params, &args); err != nil || args.Name == "" {
				code, message = -32602, "Invalid tool call"
				break
			}
			if !slices.ContainsFunc(catalog, func(t tool) bool { return t.Name == args.Name }) {
				code, message = -32602, "Unknown tool "+args.Name
				break
			}
			value, callErr := session.call(args.Name, args.Arguments)
			if callErr != nil {
				result = map[string]any{"isError": true, "content": []any{map[string]string{"type": "text", "text": callErr.Error()}}}
				break
			}
			display, err := json.Marshal(value)
			if err != nil {
				code, message = -32603, err.Error()
				break
			}
			content := []any{map[string]string{"type": "text", "text": string(display)}}
			response := map[string]any{"content": content}
			result = response
			// Structured content was introduced in the June 2025 protocol.
			if version >= "2025-06-18" {
				response["structuredContent"] = value
			}
		default:
			code, message = -32601, "Method not found"
		}
		if err := reply(result, code, message); err != nil {
			return err
		}
	}
	return scanner.Err()
}
