package agents

import (
	"errors"
	"fmt"
	"os"

	"github.com/erg0nix/kontekst/internal/core"
)

func Validate(cfg *AgentConfig) error {
	if cfg == nil {
		return errors.New("agent config is nil")
	}

	if cfg.Name == "" {
		return errors.New("agent name is required")
	}

	if cfg.Provider.Endpoint == "" {
		return errors.New("provider.endpoint is required")
	}

	if cfg.Provider.Model != "" {
		if _, err := os.Stat(cfg.Provider.Model); err != nil {
			if os.IsNotExist(err) {
				return &ModelNotFoundError{Model: cfg.Provider.Model}
			}
			return err
		}
	}

	if cfg.ContextSize <= 0 {
		return errors.New("context_size must be greater than 0")
	}

	if cfg.Sampling != nil {
		if err := validateSampling(cfg.Sampling); err != nil {
			return err
		}
	}

	return nil
}

func validateSampling(s *core.SamplingConfig) error {
	if s.Temperature != nil {
		if *s.Temperature < 0 || *s.Temperature > 2 {
			return &SamplingRangeError{Param: "temperature", Value: *s.Temperature, Min: 0, Max: 2}
		}
	}
	if s.TopP != nil {
		if *s.TopP < 0 || *s.TopP > 1 {
			return &SamplingRangeError{Param: "top_p", Value: *s.TopP, Min: 0, Max: 1}
		}
	}
	if s.TopK != nil {
		if *s.TopK < 0 {
			return fmt.Errorf("sampling.top_k must be non-negative, got %d", *s.TopK)
		}
	}
	if s.RepeatPenalty != nil {
		if *s.RepeatPenalty < 0 || *s.RepeatPenalty > 2 {
			return &SamplingRangeError{Param: "repeat_penalty", Value: *s.RepeatPenalty, Min: 0, Max: 2}
		}
	}
	return nil
}

type ModelNotFoundError struct {
	Model string
}

func (e *ModelNotFoundError) Error() string {
	return fmt.Sprintf("model file not found: %s", e.Model)
}

type SamplingRangeError struct {
	Param string
	Value float64
	Min   float64
	Max   float64
}

func (e *SamplingRangeError) Error() string {
	return fmt.Sprintf("sampling.%s value %.2f is out of range [%.1f, %.1f]", e.Param, e.Value, e.Min, e.Max)
}
