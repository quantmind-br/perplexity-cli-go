// Package models defines data structures for Perplexity API requests and responses.
package models

// Mode represents the search mode for Perplexity queries.
type Mode string

const (
	ModeFast         Mode = "fast"
	ModePro          Mode = "pro"
	ModeReasoning    Mode = "reasoning"
	ModeDeepResearch Mode = "deep-research"
	ModeDefault      Mode = "default"
)

// Model represents available AI models.
type Model string

// Pro mode models
const (
	ModelPplxPro            Model = "pplx_pro"
	ModelGPT51              Model = "gpt51"
	ModelGrok41NonReasoning Model = "grok41nonreasoning"
	ModelExperimental       Model = "experimental"
	ModelClaude45Sonnet     Model = "claude45sonnet"
)

// Reasoning mode models
const (
	ModelGemini30Pro         Model = "gemini30pro"
	ModelGPT51Thinking       Model = "gpt51_thinking"
	ModelGrok41Reasoning     Model = "grok41reasoning"
	ModelKimiK2Thinking      Model = "kimik2thinking"
	ModelClaude45SonnetThink Model = "claude45sonnetthinking"
)

// Source represents search sources.
type Source string

const (
	SourceWeb     Source = "web"
	SourceScholar Source = "scholar"
	SourceSocial  Source = "social"
)

// AvailableProModels contains models available for Pro mode.
var AvailableProModels = []Model{
	ModelPplxPro,
	ModelGPT51,
	ModelGrok41NonReasoning,
	ModelExperimental,
	ModelClaude45Sonnet,
}

// AvailableReasoningModels contains models available for Reasoning mode.
var AvailableReasoningModels = []Model{
	ModelGemini30Pro,
	ModelGPT51Thinking,
	ModelGrok41Reasoning,
	ModelKimiK2Thinking,
	ModelClaude45SonnetThink,
}

// AvailableModels contains all valid model names.
var AvailableModels = append(AvailableProModels, AvailableReasoningModels...)

// AvailableSources contains all valid source names.
var AvailableSources = []Source{
	SourceWeb,
	SourceScholar,
	SourceSocial,
}

// IsValidModel checks if a model name is valid.
func IsValidModel(m Model) bool {
	for _, valid := range AvailableModels {
		if m == valid {
			return true
		}
	}
	return false
}

// IsValidSource checks if a source name is valid.
func IsValidSource(s Source) bool {
	for _, valid := range AvailableSources {
		if s == valid {
			return true
		}
	}
	return false
}

// IsValidMode checks if a mode is valid.
func IsValidMode(m Mode) bool {
	switch m {
	case ModeFast, ModePro, ModeReasoning, ModeDeepResearch, ModeDefault:
		return true
	}
	return false
}
