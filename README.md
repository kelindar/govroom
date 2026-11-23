<p align="center">
  <img src="https://img.shields.io/badge/Go-1.25%2B-00ADD8" alt="Go 1.25+">
  <a href="https://github.com/VROOM-Project/vroom/tree/v1.15.0"><img src="https://img.shields.io/badge/VROOM-1.15.0-blue" alt="VROOM 1.15.0"></a>
</p>

# Vehicle routing optimization in Go

This package embeds VROOM v1.15.0 as WebAssembly and runs it in-process with wazero. Give it a fleet, a set of jobs or shipments, and a matrix of travel times. VROOM assigns the work to vehicles and chooses the order of their stops.

## Features

- **Embedded solver.** Builds with `CGO_ENABLED=0` and ships the VROOM binary inside the Go application.
- **Vehicle routing constraints.** Model capacities, skills, time windows, and pickup and delivery with VROOM's JSON API.
- **Custom travel matrices.** Supply durations and optional distances or costs for each vehicle profile.
- **Concurrent solves.** Share a solver across goroutines and set a deadline on each request.

## Installation

Requires Go 1.25 or newer.

```sh
go get github.com/kelindar/govroom
```

## Quick start

This example sends one vehicle from the depot at index 0 to two jobs and back. Each location index refers to a row and column in the `car` duration matrix, with travel times in seconds.

```go
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/kelindar/govroom"
)

func main() {
	ctx := context.Background()
	solver, err := govroom.New(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer solver.Close(ctx)

	problem := []byte(`{
		"vehicles": [{"id": 9, "start_index": 0, "end_index": 0}],
		"jobs": [{"id": 1, "location_index": 1}, {"id": 2, "location_index": 2}],
		"matrices": {"car": {"durations": [[0, 1, 8], [8, 0, 2], [3, 8, 0]]}}
	}`)

	request, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	solution, err := solver.Solve(request, problem)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Print(string(solution))
}
```

The solution visits job 1, then job 2, and returns to the depot in 6 seconds of travel time. The response includes route steps, summary totals, and any unassigned tasks. See the [VROOM JSON API](https://github.com/VROOM-Project/vroom/blob/v1.15.0/docs/API.md).

## Using a solver

`New` compiles the embedded module once per solver. Each call to `Solve` runs in a fresh WebAssembly instance. Reuse the solver for repeated or concurrent requests and call `Close` when finished. `Close` waits for active solves to finish.

VROOM failures return a `*govroom.Error` with a code and message. Use `errors.As` to inspect it and `errors.Is` to check for cancellation or a deadline. A job that VROOM cannot assign appears in the solution's `unassigned` list.

The embedded build handles optimization with caller-supplied matrices. Routing backends, route geometry, and GLPK plan validation are outside its scope. Each solve runs on one guest thread at VROOM's default exploration level. The package uses wazero's interpreter for C++ exception handling.

## Building VROOM from source

The module includes the WASM binary. To regenerate it on Linux or WSL, install Git, GNU Make, WASI SDK 34.0, and Binaryen 133. Put `wasm-opt` on your PATH. [WASI SDK releases](https://github.com/WebAssembly/wasi-sdk/releases/tag/wasi-sdk-34) · [Binaryen release](https://github.com/WebAssembly/binaryen/releases/tag/version_133)

```sh
bash wasm/build.sh /path/to/wasi-sdk-34.0
go test ./...
```

The script builds the pinned VROOM revision with a patch that runs matrix preparation and solver searches serially. [WASI patch](wasm/serial.patch) · [Third-party notices](wasm/licenses/)
