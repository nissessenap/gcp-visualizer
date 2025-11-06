package cli

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

// contextKey is a custom type for context keys to avoid collisions
type contextKey string

const testContextKey contextKey = "test"

func TestCLI_Context(t *testing.T) {
	tests := []struct {
		name string
		ctx  context.Context
	}{
		{
			name: "background context",
			ctx:  context.Background(),
		},
		{
			name: "context with value",
			ctx:  context.WithValue(context.Background(), testContextKey, "value"),
		},
		{
			name: "cancelled context",
			ctx:  func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx
			}(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cli := &CLI{ctx: tt.ctx}

			// Test that Context() returns the same context
			result := cli.Context()
			assert.Equal(t, tt.ctx, result)
		})
	}
}

func TestCLI_ContextAccessFromCommands(t *testing.T) {
	// This test verifies that commands can access the context
	// via the getter method without needing to access unexported fields
	ctx := context.WithValue(context.Background(), testContextKey, "value")
	cli := &CLI{ctx: ctx}

	// Simulate what a command would do
	commandCtx := cli.Context()

	assert.NotNil(t, commandCtx)
	assert.Equal(t, "value", commandCtx.Value(testContextKey))
}
