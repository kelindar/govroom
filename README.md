<p align="center">
  <img width="300" height="100" src=".github/logo.png" alt="vroom">
  <br>
  <img src="https://img.shields.io/badge/Go-1.25%2B-00ADD8" alt="Go 1.25+">
  <a href="https://pkg.go.dev/github.com/kelindar/govroom"><img src="https://pkg.go.dev/badge/github.com/kelindar/govroom.svg" alt="Go reference"></a>
  <a href="https://github.com/VROOM-Project/vroom/tree/v1.15.0"><img src="https://img.shields.io/badge/VROOM-1.15.0-blue" alt="VROOM 1.15.0"></a>
</p>

# Vehicle routing optimization in Go

This package runs VROOM v1.15.0 in a Go process. Give it some vehicles, the work they need to do, and the travel times between stops. VROOM decides which vehicle should do each job and in what order.

The solver is embedded as WebAssembly and runs through wazero. An application can build with `CGO_ENABLED=0` and ship as a single binary.

## Features

- **Jobs and shipments.** Assign individual stops or pair a pickup and delivery on the same vehicle.
- **Vehicle constraints.** Set capacities, skills, time windows, breaks, and route limits.
- **Custom matrices.** Use different travel times, distances, or costs for cars, bikes, and other vehicle profiles.
- **Go or JSON input.** Assemble a request with Go structs or send existing VROOM JSON.
- **Concurrent solves.** Reuse one solver across requests and give each solve its own deadline.

## Installation

Requires Go 1.25 or newer.

```sh
go get github.com/kelindar/govroom
```

Import `github.com/kelindar/govroom` and use it as `vroom` in Go.

## Quick start

Let's send one vehicle from a depot to two jobs and back.

1. Put the depot at index 0 and the jobs at indices 1 and 2.
2. Build the travel-time matrix in that same order. The entry at row 0, column 1 is the time from the depot to job 1.
3. Create a solver, pass it the problem, and read the solution.

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/kelindar/govroom"
)

func main() {
	ctx := context.Background()
	solver, err := vroom.New(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer solver.Close(ctx)

	depot, first, second := uint16(0), uint16(1), uint16(2)
	problem := vroom.Problem{
		Vehicles: []vroom.Vehicle{
			{ID: 9, StartIndex: &depot, EndIndex: &depot},
		},
		Jobs: []vroom.Job{
			{ID: 1, LocationIndex: &first},
			{ID: 2, LocationIndex: &second},
		},
		Matrices: map[string]vroom.Matrix{
			"car": {Durations: [][]uint32{
				{0, 1, 8},
				{8, 0, 2},
				{3, 8, 0},
			}},
		},
	}

	result, err := solver.Solve(ctx, problem)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Print(string(result))
}
```

The vehicle visits job 1, then job 2. The route takes 6 seconds of travel time. In the next examples, `ctx` and `solver` refer to the values created above.

## Building a problem

A `Job` is a single stop. Use its `Delivery` and `Pickup` fields for quantities that a vehicle drops off or collects there. A `Shipment` pairs two stops and requires the same vehicle to make the pickup before the delivery.

Vehicle capacities and task quantities use the same dimensions in the same order. For example, if your dimensions are parcels and kilograms, a capacity of `[]uint32{10, 50}` can carry up to 10 parcels and 50 kilograms. Any skill listed on a job or shipment must also be listed on the vehicle. Time windows are pairs of seconds and constrain when service can begin.

### Pickup and delivery with two vehicle profiles

This fleet has a car and a bike. Skill 1 restricts the shipment to the car. Skill 2 restricts the single job to the bike. The pickup can start between seconds 4 and 6, and the delivery between seconds 8 and 10.

```go
depot, pickup, dropoff := uint16(0), uint16(1), uint16(2)
problem := vroom.Problem{
	Vehicles: []vroom.Vehicle{
		{ID: 31, StartIndex: &depot, EndIndex: &depot, Capacity: []uint32{1}, Skills: []uint32{1}},
		{ID: 32, Profile: "bike", StartIndex: &depot, EndIndex: &depot, Capacity: []uint32{1}, Skills: []uint32{2}},
	},
	Jobs: []vroom.Job{
		{ID: 20, LocationIndex: &pickup, Delivery: []uint32{1}, Skills: []uint32{2}},
	},
	Shipments: []vroom.Shipment{
		{
			Amount: []uint32{1},
			Skills: []uint32{1},
			Pickup: vroom.ShipmentStep{
				ID: 10, LocationIndex: &pickup, TimeWindows: []vroom.TimeWindow{{4, 6}},
			},
			Delivery: vroom.ShipmentStep{
				ID: 11, LocationIndex: &dropoff, TimeWindows: []vroom.TimeWindow{{8, 10}},
			},
		},
	},
	Matrices: map[string]vroom.Matrix{
		"car":  {Durations: [][]uint32{{0, 4, 8}, {4, 0, 4}, {8, 4, 0}}},
		"bike": {Durations: [][]uint32{{0, 5, 9}, {5, 0, 5}, {9, 5, 0}}},
	},
}

result, err := solver.Solve(ctx, &problem)
if err != nil {
	log.Fatal(err)
}
fmt.Print(string(result))
```

### Location indices and matrices

Give each vehicle profile a square duration matrix. The default profile is `car`. Rows are origins, columns are destinations, and durations are in seconds. The matrices can be asymmetric, for example when a one-way street makes the return trip longer. IDs identify vehicles and tasks in the result, while the indices refer to matrix entries. Multiple tasks can use the same index.

An index of zero is valid, so the Go models use pointers for optional indices. Use `&depot` for index 0 and `nil` to omit the field. For example, a vehicle with a start index and no end index can finish at its last assigned stop. You can also include coordinates as `vroom.Coordinates{longitude, latitude}` alongside an index.

Use `Distances` to supply distances in meters and `Costs` to supply a separate objective matrix. Give them the same layout as `Durations`. Include distance matrices for all used profiles if you supply any, and include distances for a profile if its vehicles use `VehicleCosts.PerKm` or `MaxDistance`. With a custom cost matrix, leave `VehicleCosts.PerHour` at its default. See the [VROOM input reference](https://github.com/VROOM-Project/vroom/blob/v1.15.0/docs/API.md#input) for the full schemas, including vehicle breaks, initial routes, and per-type service times.

## Sending JSON

If you already have a VROOM request from a file, an HTTP body, or a fixture, pass it directly. `Solve` accepts JSON text as a `[]byte`, `json.RawMessage`, or `string`. This example sends the same problem in all three forms. In an application, use whichever form you already have.

```go
raw := []byte(`{
	"vehicles": [{"id": 9, "start_index": 0, "end_index": 0}],
	"jobs": [{"id": 1, "location_index": 1}],
	"matrices": {"car": {"durations": [[0, 4], [5, 0]]}}
}`)

for _, input := range []any{raw, json.RawMessage(raw), string(raw)} {
	result, err := solver.Solve(ctx, input)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Print(string(result))
}
```

For other Go values, `Solve` uses `json.Marshal`. Use your own struct with VROOM JSON tags or a map when you build the request dynamically. Both requests below use the vehicles, jobs, and matrices from the quick start.

```go
type deliveryRequest struct {
	Fleet  []vroom.Vehicle         `json:"vehicles"`
	Stops  []vroom.Job             `json:"jobs"`
	Travel map[string]vroom.Matrix `json:"matrices"`
}
typed := deliveryRequest{
	Fleet: problem.Vehicles, Stops: problem.Jobs, Travel: problem.Matrices,
}
dynamic := map[string]any{
	"vehicles": problem.Vehicles,
	"jobs":     problem.Jobs,
	"matrices": problem.Matrices,
}

for _, input := range []any{typed, dynamic} {
	result, err := solver.Solve(ctx, input)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Print(string(result))
}
```

## Reading the solution

`Solve` returns VROOM's response as JSON bytes. Print or forward them as-is, or decode just the fields your application needs. This example prints the vehicle ID and arrival time for each job in the quick-start result.

```go
var solution struct {
	Routes []struct {
		Vehicle uint64 `json:"vehicle"`
		Steps   []struct {
			Type    string `json:"type"`
			ID      uint64 `json:"id"`
			Arrival uint64 `json:"arrival"`
		} `json:"steps"`
	} `json:"routes"`
	Unassigned []struct {
		ID uint64 `json:"id"`
	} `json:"unassigned"`
}

if err := json.Unmarshal(result, &solution); err != nil {
	log.Fatal(err)
}
for _, route := range solution.Routes {
	for _, step := range route.Steps {
		if step.Type == "job" {
			fmt.Printf("vehicle %d visits job %d at %ds\n", route.Vehicle, step.ID, step.Arrival)
		}
	}
}
fmt.Printf("%d unassigned tasks\n", len(solution.Unassigned))
```

Check `unassigned` even when `Solve` succeeds. For example, a job can be left out if it requires a skill that none of the vehicles have. The [VROOM output reference](https://github.com/VROOM-Project/vroom/blob/v1.15.0/docs/API.md#output) also covers route totals, shipment steps, arrival times, and vehicle loads.

## Deadlines and errors

Create a context with a deadline for each request. `errors.Is` distinguishes deadlines and cancellation. A failure reported by VROOM is available as a `*vroom.Error`. It has a code and message, with code 1 for internal errors, 2 for invalid input, and 3 for routing errors.

```go
request, cancel := context.WithTimeout(ctx, 2*time.Second)
defer cancel()

result, err := solver.Solve(request, problem)
var failure *vroom.Error
switch {
case errors.Is(err, context.DeadlineExceeded):
	log.Print("route optimization timed out")
case errors.Is(err, context.Canceled):
	log.Print("route optimization canceled")
case errors.As(err, &failure):
	log.Printf("VROOM error %d: %s", failure.Code, failure.Message)
case err != nil:
	log.Print(err)
default:
	fmt.Print(string(result))
}
```

## Reusing a solver

Call `New` once for a service or batch of requests. Each solve gets a fresh WASM instance and shares the compiled module, so several goroutines can call `Solve` at the same time. Keep each request's data read-only until its solve returns. A canceled solve leaves the solver available for later requests. Call `Close` when finished. It waits for active solves and releases the runtime.

Each solve runs on one guest thread at VROOM's default exploration level. The embedded build covers matrix-based optimization. Routing backends, route geometry, and GLPK plan validation require other VROOM builds.

## Building VROOM from source

The repository includes the WASM binary. To regenerate it on Linux or WSL, install Git, GNU Make, WASI SDK 34.0, and Binaryen 133. Put `wasm-opt` on your PATH. [WASI SDK](https://github.com/WebAssembly/wasi-sdk/releases/tag/wasi-sdk-34) · [Binaryen](https://github.com/WebAssembly/binaryen/releases/tag/version_133)

```sh
bash wasm/build.sh /path/to/wasi-sdk-34.0
go test ./...
```

The script checks out VROOM v1.15.0, runs matrix preparation and solver searches serially via the WASI patch, and translates C++ exception instructions for wazero's interpreter. [WASI patch](wasm/serial.patch) · [Third-party notices](wasm/licenses/)
