package vroom_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/kelindar/govroom"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProblem(t *testing.T) {
	depot, first, second := uint16(0), uint16(1), uint16(2)
	window := vroom.TimeWindow{0, 30}
	durations := [][]uint32{{0, 4, 8}, {4, 0, 4}, {8, 4, 0}}
	distances := [][]uint32{{0, 40, 80}, {40, 0, 40}, {80, 40, 0}}
	problem := vroom.Problem{
		Vehicles: []vroom.Vehicle{
			{ID: 31, StartIndex: &depot, EndIndex: &depot, Capacity: []uint32{1}, Skills: []uint32{1}, TimeWindow: &window},
			{ID: 32, Profile: "bike", StartIndex: &depot, EndIndex: &depot, Capacity: []uint32{1}, Skills: []uint32{2}, TimeWindow: &window},
		},
		Jobs: []vroom.Job{
			{ID: 20, LocationIndex: &first, Delivery: []uint32{1}, Skills: []uint32{2}},
			{ID: 21, LocationIndex: &second, Delivery: []uint32{2}},
		},
		Shipments: []vroom.Shipment{{
			Amount:   []uint32{1},
			Skills:   []uint32{1},
			Pickup:   vroom.ShipmentStep{ID: 10, LocationIndex: &first, TimeWindows: []vroom.TimeWindow{{4, 6}}},
			Delivery: vroom.ShipmentStep{ID: 11, LocationIndex: &second, TimeWindows: []vroom.TimeWindow{{8, 10}}},
		}},
		Matrices: map[string]vroom.Matrix{
			"car":  {Durations: durations, Distances: distances},
			"bike": {Durations: durations, Distances: distances, Costs: [][]uint32{{0, 3, 6}, {3, 0, 3}, {6, 3, 0}}},
		},
	}

	fixture, err := os.ReadFile(filepath.Join("testdata", "shipment.json"))
	require.NoError(t, err)
	encoded, err := json.Marshal(problem)
	require.NoError(t, err)
	assert.JSONEq(t, string(fixture), string(encoded))

	ctx := context.Background()
	solver, err := vroom.New(ctx)
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, solver.Close(ctx)) })

	for _, tc := range []struct {
		name  string
		input any
	}{
		{"value", problem},
		{"pointer", &problem},
	} {
		t.Run(tc.name, func(t *testing.T) {
			output, err := solver.Solve(ctx, tc.input)
			require.NoError(t, err)
			var result solution
			require.NoError(t, json.Unmarshal(output, &result))
			assert.Zero(t, result.Code)
			assert.Equal(t, 22, result.Summary.Cost)
			assert.Equal(t, 240, result.Summary.Distance)
			require.Len(t, result.Unassigned, 1)
			assert.Equal(t, 21, result.Unassigned[0].ID)
		})
	}
}

func TestOptionalFields(t *testing.T) {
	zero := uint32(0)
	vehicle := vroom.Vehicle{
		ID:         1,
		End:        &vroom.Coordinates{0, 0},
		MaxTasks:   &zero,
		Costs:      &vroom.VehicleCosts{PerHour: &zero},
		TimeWindow: &vroom.TimeWindow{0, 0},
	}
	encoded, err := json.Marshal(vehicle)
	require.NoError(t, err)
	assert.JSONEq(t, `{"id":1,"end":[0,0],"max_tasks":0,"costs":{"per_hour":0},"time_window":[0,0]}`, string(encoded))
}
