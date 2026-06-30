package agent

import (
	"context"
	"plaza/models"
)

type Backend interface {
	Name() string
	Generate(ctx context.Context, history []models.ChatMessage, trigger string) (string, error)
}

type EchoBackend struct{}

func (e *EchoBackend) Name() string { return "Eco" }

func (e *EchoBackend) Generate(ctx context.Context, history []models.ChatMessage, trigger string) (string, error) {
	if trigger == "" {
		return "...", nil
	}
	return "Eco: \"" + trigger + "\"", nil
}
