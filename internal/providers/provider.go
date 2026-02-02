package providers

import (
	"sync"

	"github.com/erg0nix/kontekst/internal/core"
)

type Provider interface {
	GenerateChat(
		messages []core.Message,
		tools []core.ToolDef,
		sampling *core.SamplingConfig,
		model string,
		useToolRole bool,
	) (core.ChatResponse, error)
	CountTokens(text string) (int, error)
	ConcurrencyLimit() int
}

type ProviderRouter interface {
	Provider
}

type SingleProviderRouter struct {
	Provider Provider
	once     sync.Once
	limiter  *semaphore
}

func (r *SingleProviderRouter) GenerateChat(
	messages []core.Message,
	tools []core.ToolDef,
	sampling *core.SamplingConfig,
	model string,
	useToolRole bool,
) (core.ChatResponse, error) {
	if r.Provider == nil {
		return core.ChatResponse{}, nil
	}

	if concurrencyLimiter := r.getLimiter(); concurrencyLimiter != nil {
		concurrencyLimiter.acquire()
		defer concurrencyLimiter.release()
	}

	return r.Provider.GenerateChat(messages, tools, sampling, model, useToolRole)
}

func (r *SingleProviderRouter) CountTokens(text string) (int, error) {
	if r.Provider == nil {
		return 0, nil
	}

	return r.Provider.CountTokens(text)
}

func (r *SingleProviderRouter) ConcurrencyLimit() int {
	if r.Provider == nil {
		return 0
	}

	return r.Provider.ConcurrencyLimit()
}

func (r *SingleProviderRouter) getLimiter() *semaphore {
	r.once.Do(func() {
		concurrencyLimit := r.ConcurrencyLimit()

		if concurrencyLimit > 0 {
			r.limiter = newSemaphore(concurrencyLimit)
		}
	})
	return r.limiter
}

type semaphore struct {
	ch chan struct{}
}

func newSemaphore(limit int) *semaphore {
	return &semaphore{ch: make(chan struct{}, limit)}
}

func (s *semaphore) acquire() {
	s.ch <- struct{}{}
}

func (s *semaphore) release() {
	<-s.ch
}
