package st

type GUIInputPrompt struct {
	ID      uint64   `json:"prompt_id"`
	Message string   `json:"message"`
	Kinds   []string `json:"kinds"`
}
type GUICommandInput struct {
	Prompt uint64    `json:"prompt_id"`
	Kind   string    `json:"kind"`
	ID     int       `json:"id,omitempty"`
	Text   string    `json:"text,omitempty"`
	Point  []float64 `json:"point,omitempty"`
}
