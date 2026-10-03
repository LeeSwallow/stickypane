package mcp

import (
	"encoding/json"

	"github.com/LeeSwallow/stickypane/internal/harness"
)

type prompt struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// prompts are the plugin's skills, so a client without the plugin can
// still offer them: the way in first.
func prompts() []prompt {
	var out []prompt
	for _, s := range harness.Skills() {
		out = append(out, prompt{Name: s.Name, Description: s.Description})
	}
	return out
}

// getPrompt is one skill, with the files it points to, as a message to
// follow.
func getPrompt(params json.RawMessage) (any, *rpcError) {
	var p struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, &rpcError{codeParams, "cannot read the prompt request: " + err.Error()}
	}
	s, ok := harness.Lookup(p.Name)
	if !ok {
		return nil, &rpcError{codeParams, "unknown prompt " + p.Name}
	}
	return map[string]any{
		"description": s.Description,
		"messages": []map[string]any{{
			"role":    "user",
			"content": content{Type: "text", Text: harness.Read(s)},
		}},
	}, nil
}
