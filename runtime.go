package vroom

import (
	"context"
	"fmt"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

func newSolver(ctx context.Context, guest []byte) (*Solver, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	config := wazero.NewRuntimeConfig().WithCloseOnContextDone(true)
	runtime := wazero.NewRuntimeWithConfig(ctx, config)
	if _, err := wasi_snapshot_preview1.Instantiate(ctx, runtime); err != nil {
		_ = runtime.Close(context.WithoutCancel(ctx))
		return nil, fmt.Errorf("initialize VROOM WASI: %w", err)
	}

	module, err := runtime.CompileModule(ctx, guest)
	if err != nil {
		_ = runtime.Close(context.WithoutCancel(ctx))
		return nil, fmt.Errorf("compile VROOM: %w", err)
	}
	if err := ctx.Err(); err != nil {
		_ = runtime.Close(context.WithoutCancel(ctx))
		return nil, err
	}

	return &Solver{runtime: runtime, module: module}, nil
}
