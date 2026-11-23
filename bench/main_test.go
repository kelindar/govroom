package main

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/kelindar/govroom"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeliveryProblem(t *testing.T) {
	ctx := context.Background()
	solver, err := vroom.New(ctx)
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, solver.Close(ctx)) })

	for _, tc := range []struct {
		name     string
		vehicles int
	}{
		{"10 jobs", 1},
		{"20 jobs", 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			problem := deliveryProblem(tc.vehicles)
			output, err := solver.Solve(ctx, &problem)
			require.NoError(t, err)
			var solution struct {
				Code    int
				Summary struct {
					Routes     int
					Unassigned int
				}
			}
			require.NoError(t, json.Unmarshal(output, &solution))
			assert.Zero(t, solution.Code)
			assert.Zero(t, solution.Summary.Unassigned)
			assert.Equal(t, tc.vehicles, solution.Summary.Routes)
		})
	}
}
