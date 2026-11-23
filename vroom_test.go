package vroom_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/kelindar/govroom"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSolver(t *testing.T) {
	ctx := context.Background()
	solver, err := vroom.New(ctx)
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, solver.Close(ctx)) })

	t.Run("traveling salesman", func(t *testing.T) {
		t.Parallel()
		result := solve(t, solver, "tsp.json")
		assert.Equal(t, 6, result.Summary.Cost)
		assert.Empty(t, result.Unassigned)
		require.Len(t, result.Routes, 1)
		assert.Equal(t, 9, result.Routes[0].Vehicle)
		require.Len(t, result.Routes[0].Steps, 4)
		assert.Equal(t, 1, result.Routes[0].Steps[1].ID)
		assert.Equal(t, 2, result.Routes[0].Steps[2].ID)
	})

	t.Run("shipment and time windows", func(t *testing.T) {
		t.Parallel()
		result := solve(t, solver, "shipment.json")
		assert.Equal(t, 22, result.Summary.Cost)
		assert.Equal(t, 240, result.Summary.Distance)
		require.Len(t, result.Unassigned, 1)
		assert.Equal(t, 21, result.Unassigned[0].ID)
		require.Len(t, result.Routes, 2)
		for _, route := range result.Routes {
			if route.Vehicle == 31 {
				require.Len(t, route.Steps, 4)
				assert.Equal(t, "pickup", route.Steps[1].Type)
				assert.Equal(t, 10, route.Steps[1].ID)
				assert.Equal(t, 4, route.Steps[1].Arrival)
				assert.Equal(t, "delivery", route.Steps[2].Type)
				assert.Equal(t, 11, route.Steps[2].ID)
				assert.Equal(t, 8, route.Steps[2].Arrival)
			} else {
				assert.Equal(t, 32, route.Vehicle)
				require.Len(t, route.Steps, 3)
				assert.Equal(t, 20, route.Steps[1].ID)
			}
		}
	})

	t.Run("input errors", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name  string
			input string
			code  uint32
		}{
			{"malformed", "{", 2},
			{"missing vehicles", `{"vehicles":[],"jobs":[{"id":1,"location_index":1}]}`, 2},
			{"invalid matrix index", `{"vehicles":[{"id":1,"start_index":0}],"jobs":[{"id":1,"location_index":2}],"matrices":{"car":{"durations":[[0,1],[1,0]]}}}`, 2},
			{"missing matrix", `{"vehicles":[{"id":1,"start":[0,0]}],"jobs":[{"id":1,"location":[1,1]}]}`, 3},
		} {
			t.Run(tc.name, func(t *testing.T) {
				output, err := solver.Solve(ctx, []byte(tc.input))
				var failure *vroom.Error
				require.ErrorAs(t, err, &failure)
				assert.Equal(t, tc.code, failure.Code)
				assert.NotEmpty(t, failure.Message)
				assert.EqualError(t, err, fmt.Sprintf("VROOM error %d: %s", tc.code, failure.Message))
				assert.Nil(t, output)
			})
		}

		assert.Equal(t, 6, solve(t, solver, "tsp.json").Summary.Cost)
	})
}

func TestLifecycle(t *testing.T) {
	ctx := context.Background()
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	_, err := vroom.New(canceled)
	assert.ErrorIs(t, err, context.Canceled)

	solver, err := vroom.New(ctx)
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, solver.Close(ctx)) })

	_, err = solver.Solve(canceled, nil)
	assert.ErrorIs(t, err, context.Canceled)
	assert.Equal(t, 6, solve(t, solver, "tsp.json").Summary.Cost)

	require.NoError(t, solver.Close(ctx))
	_, err = solver.Solve(ctx, nil)
	assert.ErrorContains(t, err, "solver is closed")
}

func TestInputFormats(t *testing.T) {
	ctx := context.Background()
	solver, err := vroom.New(ctx)
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, solver.Close(ctx)) })

	raw, err := os.ReadFile(filepath.Join("testdata", "tsp.json"))
	require.NoError(t, err)
	var object map[string]any
	require.NoError(t, json.Unmarshal(raw, &object))

	for _, tc := range []struct {
		name  string
		input any
	}{
		{"bytes", raw},
		{"raw message", json.RawMessage(raw)},
		{"string", string(raw)},
		{"map", object},
	} {
		t.Run(tc.name, func(t *testing.T) {
			output, err := solver.Solve(ctx, tc.input)
			require.NoError(t, err)
			var result solution
			require.NoError(t, json.Unmarshal(output, &result))
			assert.Zero(t, result.Code)
			assert.Equal(t, 6, result.Summary.Cost)
		})
	}

	t.Run("invalid raw JSON", func(t *testing.T) {
		_, err := solver.Solve(ctx, json.RawMessage("{"))
		var failure *vroom.Error
		require.ErrorAs(t, err, &failure)
		assert.Equal(t, uint32(2), failure.Code)
	})

	t.Run("encoding failure", func(t *testing.T) {
		output, err := solver.Solve(ctx, make(chan int))
		var unsupported *json.UnsupportedTypeError
		assert.ErrorAs(t, err, &unsupported)
		assert.Nil(t, output)
	})

	t.Run("cancel during encoding", func(t *testing.T) {
		request, cancel := context.WithCancel(ctx)
		defer cancel()

		output, err := solver.Solve(request, cancelOnMarshal{cancel: cancel})
		assert.ErrorIs(t, err, context.Canceled)
		assert.Nil(t, output)
		assert.Equal(t, 6, solve(t, solver, "tsp.json").Summary.Cost)
	})
}

type cancelOnMarshal struct {
	cancel context.CancelFunc
}

func (c cancelOnMarshal) MarshalJSON() ([]byte, error) {
	c.cancel()
	return []byte("{}"), nil
}

type solution struct {
	Code    int
	Summary struct {
		Cost     int
		Distance int
	}
	Routes []struct {
		Vehicle int
		Steps   []struct {
			Type    string
			ID      int
			Arrival int
		}
	}
	Unassigned []struct{ ID int }
}

func solve(t *testing.T, solver *vroom.Solver, name string) solution {
	t.Helper()
	input, err := os.ReadFile(filepath.Join("testdata", name))
	require.NoError(t, err)
	output, err := solver.Solve(context.Background(), input)
	require.NoError(t, err)

	var result solution
	require.NoError(t, json.Unmarshal(output, &result))
	require.Zero(t, result.Code)
	return result
}
