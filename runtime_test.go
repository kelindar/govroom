package vroom

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/experimental"
)

func TestRuntime(t *testing.T) {
	ctx := context.Background()

	t.Run("invalid module", func(t *testing.T) {
		solver, err := newSolver(ctx, []byte("\x00asm\x01"))
		assert.ErrorContains(t, err, "compile VROOM:")
		assert.Nil(t, solver)
	})

	t.Run("cancel during compilation", func(t *testing.T) {
		compilation, cancel := context.WithCancel(ctx)
		defer cancel()
		compilation = experimental.WithFunctionListenerFactory(compilation, experimental.FunctionListenerFactoryFunc(
			func(def api.FunctionDefinition) experimental.FunctionListener {
				if def.GoFunction() == nil {
					cancel()
				}
				return nil
			},
		))

		solver, err := newSolver(compilation, []byte(trappingGuest))
		assert.ErrorIs(t, err, context.Canceled)
		assert.Nil(t, solver)
	})

	t.Run("guest trap", func(t *testing.T) {
		solver, err := newSolver(ctx, []byte(trappingGuest))
		require.NoError(t, err)
		t.Cleanup(func() { assert.NoError(t, solver.Close(ctx)) })

		for range 2 {
			output, err := solver.Solve(ctx, nil)
			assert.ErrorContains(t, err, "solve with VROOM:")
			assert.ErrorContains(t, err, "unreachable")
			assert.Nil(t, output)
		}
	})
}

// (module (func (export "_start") unreachable))
const trappingGuest = "\x00asm\x01\x00\x00\x00" +
	"\x01\x04\x01\x60\x00\x00" +
	"\x03\x02\x01\x00" +
	"\x07\x0a\x01\x06_start\x00\x00" +
	"\x0a\x05\x01\x03\x00\x00\x0b"
