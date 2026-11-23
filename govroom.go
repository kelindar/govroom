// Package govroom runs VROOM vehicle routing optimization in WebAssembly.
package govroom

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/experimental"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
	"github.com/tetratelabs/wazero/sys"
)

//go:embed wasm/vroom.wasm
var binary []byte

// Solver reuses a compiled VROOM module across calls. Construct it with New.
// A Solver is safe for concurrent use and must not be copied.
type Solver struct {
	mu      sync.RWMutex
	runtime wazero.Runtime
	module  wazero.CompiledModule
}

// New compiles the embedded VROOM module. Reuse the returned solver for
// multiple requests and call Close when finished.
func New(ctx context.Context) (*Solver, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	// wazero 1.12's compiler can trap while unwinding VROOM C++ exceptions.
	config := wazero.NewRuntimeConfigInterpreter().
		WithCoreFeatures(api.CoreFeaturesV2 | experimental.CoreFeaturesExceptionHandling).
		WithCloseOnContextDone(true)
	runtime := wazero.NewRuntimeWithConfig(ctx, config)
	if _, err := wasi_snapshot_preview1.Instantiate(ctx, runtime); err != nil {
		_ = runtime.Close(context.WithoutCancel(ctx))
		return nil, fmt.Errorf("initialize VROOM WASI: %w", err)
	}

	module, err := runtime.CompileModule(ctx, binary)
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

// Close waits for active solves and releases the runtime. It is safe to call
// more than once.
func (s *Solver) Close(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.runtime == nil {
		return nil
	}

	err := s.runtime.Close(ctx)
	s.runtime, s.module = nil, nil
	return err
}

// Solve accepts a VROOM JSON problem with custom travel matrices and returns
// the VROOM JSON solution. It uses VROOM's default exploration level and one
// guest thread. The returned bytes belong to the caller. Do not modify problem
// until Solve returns. Cancel ctx to interrupt a solve.
func (s *Solver) Solve(ctx context.Context, problem []byte) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.runtime == nil {
		return nil, errors.New("solve with VROOM: solver is closed or uninitialized")
	}

	var output bytes.Buffer
	config := wazero.NewModuleConfig().
		WithName("").
		WithArgs("vroom", "-t", "1").
		WithStdin(bytes.NewReader(problem)).
		WithStdout(&output).
		WithSysWalltime().
		WithSysNanotime()

	module, err := s.runtime.InstantiateModule(ctx, s.module, config)
	if err != nil {
		var exit *sys.ExitError
		var failure Error
		if errors.As(err, &exit) &&
			json.Unmarshal(output.Bytes(), &failure) == nil &&
			failure.Code == exit.ExitCode() && failure.Message != "" {
			return nil, &failure
		}
		return nil, fmt.Errorf("solve with VROOM: %w", err)
	}

	_ = module.Close(context.WithoutCancel(ctx))
	return output.Bytes(), nil
}

// Error is a failure reported by VROOM. Its codes are 1 for internal errors,
// 2 for invalid input, and 3 for routing errors.
type Error struct {
	Code    uint32 `json:"code"`
	Message string `json:"error"`
}

func (e *Error) Error() string {
	return fmt.Sprintf("VROOM error %d: %s", e.Code, e.Message)
}
