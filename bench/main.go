package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/kelindar/bench"
	"github.com/kelindar/govroom"
)

func main() {
	ctx := context.Background()
	solver, err := vroom.New(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer solver.Close(ctx)

	bench.Run(func(b *bench.B) {
		for _, vehicles := range []int{1, 2} {
			problem := deliveryProblem(vehicles)
			raw, err := json.Marshal(problem)
			if err != nil {
				log.Fatal(err)
			}

			for _, input := range []struct {
				format string
				value  any
			}{
				{"json", raw},
				{"typed", &problem},
			} {
				name := fmt.Sprintf("solve/%d-jobs/%s", len(problem.Jobs), input.format)
				b.Run(name, func(int) {
					if _, err := solver.Solve(ctx, input.value); err != nil {
						panic(fmt.Errorf("%s: %w", name, err))
					}
				})
			}
		}
	}, bench.WithSamples(20), bench.WithDuration(100*time.Millisecond))
}

func deliveryProblem(vehicles int) vroom.Problem {
	const stopsPerVehicle = 10
	depot := uint16(0)
	problem := vroom.Problem{
		Vehicles: make([]vroom.Vehicle, vehicles),
		Jobs:     make([]vroom.Job, vehicles*stopsPerVehicle),
	}
	for i := range problem.Vehicles {
		problem.Vehicles[i] = vroom.Vehicle{
			ID: uint64(i + 1), StartIndex: &depot, EndIndex: &depot,
			Capacity: []uint32{stopsPerVehicle},
		}
	}
	for i := range problem.Jobs {
		stop := uint16(i + 1)
		window := vroom.TimeWindow{0, 240}
		if i%2 == 1 {
			window = vroom.TimeWindow{100, 480}
		}
		problem.Jobs[i] = vroom.Job{
			ID: uint64(stop), LocationIndex: &stop, Delivery: []uint32{1},
			Service: 5, TimeWindows: []vroom.TimeWindow{window},
		}
	}

	// Use Manhattan travel times on an eight-column grid with the depot at index 0.
	const columns = 8
	size := len(problem.Jobs) + 1
	durations := make([][]uint32, size)
	for i := range durations {
		durations[i] = make([]uint32, size)
		row, column := i/columns, i%columns
		for j := range durations[i] {
			otherRow, otherColumn := j/columns, j%columns
			vertical := max(row, otherRow) - min(row, otherRow)
			horizontal := max(column, otherColumn) - min(column, otherColumn)
			durations[i][j] = uint32(5 * (vertical + horizontal))
		}
	}
	problem.Matrices = map[string]vroom.Matrix{"car": {Durations: durations}}
	return problem
}
