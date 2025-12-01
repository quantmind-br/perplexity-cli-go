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

const (
	ModelPplxPro              Model = "pplx_pro"
	ModelExperimental         Model = "experimental"
	ModelSonar                Model = "sonar"
	ModelGrok4                Model = "grok4"
	ModelGPT5                 Model = "gpt5"
	ModelClaude45Sonnet       Model = "claude45sonnet"
	ModelGemini2Flash         Model = "gemini2flash"
	ModelGPT5Thinking         Model = "gpt5_thinking"
	ModelClaude45SonnetThink  Model = "claude45sonnetthinking"
)

// Source represents search sources.
type Source string

const (
	SourceWeb     Source = "web"
	SourceScholar Source = "scholar"
	SourceSocial  Source = "social"
)

// AvailableModels contains all valid model names.
var AvailableModels = []Model{
	ModelPplxPro,
	ModelExperimental,
	ModelSonar,
	ModelGrok4,
	ModelGPT5,
	ModelClaude45Sonnet,
	ModelGemini2Flash,
	ModelGPT5Thinking,
	ModelClaude45SonnetThink,
}

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
