package vroom

// Problem describes a fleet, the work to assign, and travel matrices keyed by
// vehicle profile. Vehicles without a profile use the "car" matrix.
type Problem struct {
	Vehicles  []Vehicle         `json:"vehicles"`
	Jobs      []Job             `json:"jobs,omitempty"`
	Shipments []Shipment        `json:"shipments,omitempty"`
	Matrices  map[string]Matrix `json:"matrices,omitempty"`
}

// Matrix holds travel durations in seconds, optional distances in meters, and
// optional costs for one profile. Location indices select rows and columns.
type Matrix struct {
	Durations [][]uint32 `json:"durations,omitempty"`
	Distances [][]uint32 `json:"distances,omitempty"`
	Costs     [][]uint32 `json:"costs,omitempty"`
}

// Coordinates hold longitude and latitude in that order.
type Coordinates [2]float64

// TimeWindow holds the earliest and latest allowed times in seconds.
type TimeWindow [2]uint32

// Vehicle describes an available vehicle and its route constraints.
// A nil StartIndex or EndIndex leaves that endpoint's matrix index unspecified.
type Vehicle struct {
	ID            uint64        `json:"id"`
	Profile       string        `json:"profile,omitempty"`
	Description   string        `json:"description,omitempty"`
	Start         *Coordinates  `json:"start,omitempty"`
	StartIndex    *uint16       `json:"start_index,omitempty"`
	End           *Coordinates  `json:"end,omitempty"`
	EndIndex      *uint16       `json:"end_index,omitempty"`
	Capacity      []uint32      `json:"capacity,omitempty"`
	Costs         *VehicleCosts `json:"costs,omitempty"`
	Skills        []uint32      `json:"skills,omitempty"`
	Type          string        `json:"type,omitempty"`
	TimeWindow    *TimeWindow   `json:"time_window,omitempty"`
	Breaks        []Break       `json:"breaks,omitempty"`
	SpeedFactor   float64       `json:"speed_factor,omitempty"`
	MaxTasks      *uint32       `json:"max_tasks,omitempty"`
	MaxTravelTime *uint32       `json:"max_travel_time,omitempty"`
	MaxDistance   *uint32       `json:"max_distance,omitempty"`
	Steps         []VehicleStep `json:"steps,omitempty"`
}

// VehicleCosts sets per-vehicle costs. A nil PerHour uses VROOM's default
// travel cost of 3600 per hour.
type VehicleCosts struct {
	Fixed       uint32  `json:"fixed,omitempty"`
	PerHour     *uint32 `json:"per_hour,omitempty"`
	PerTaskHour uint32  `json:"per_task_hour,omitempty"`
	PerKm       uint32  `json:"per_km,omitempty"`
}

// Break describes a vehicle break and the constraints on when it can occur.
type Break struct {
	ID          uint64       `json:"id"`
	TimeWindows []TimeWindow `json:"time_windows,omitempty"`
	Service     uint32       `json:"service,omitempty"`
	Description string       `json:"description,omitempty"`
	MaxLoad     []uint32     `json:"max_load,omitempty"`
}

// VehicleStep describes a step in a vehicle's initial route. ID identifies the
// task or break. Start and end steps can omit it.
type VehicleStep struct {
	Type string  `json:"type"`
	ID   *uint64 `json:"id,omitempty"`
}

// Job is a single stop with optional delivery and pickup quantities.
type Job struct {
	ID             uint64            `json:"id"`
	Description    string            `json:"description,omitempty"`
	Location       *Coordinates      `json:"location,omitempty"`
	LocationIndex  *uint16           `json:"location_index,omitempty"`
	Setup          uint32            `json:"setup,omitempty"`
	Service        uint32            `json:"service,omitempty"`
	SetupPerType   map[string]uint32 `json:"setup_per_type,omitempty"`
	ServicePerType map[string]uint32 `json:"service_per_type,omitempty"`
	Delivery       []uint32          `json:"delivery,omitempty"`
	Pickup         []uint32          `json:"pickup,omitempty"`
	Skills         []uint32          `json:"skills,omitempty"`
	Priority       uint32            `json:"priority,omitempty"`
	TimeWindows    []TimeWindow      `json:"time_windows,omitempty"`
}

// Shipment links a pickup and delivery that must use the same vehicle.
type Shipment struct {
	Pickup   ShipmentStep `json:"pickup"`
	Delivery ShipmentStep `json:"delivery"`
	Amount   []uint32     `json:"amount,omitempty"`
	Skills   []uint32     `json:"skills,omitempty"`
	Priority uint32       `json:"priority,omitempty"`
}

// ShipmentStep describes one stop in a shipment.
type ShipmentStep struct {
	ID             uint64            `json:"id"`
	Description    string            `json:"description,omitempty"`
	Location       *Coordinates      `json:"location,omitempty"`
	LocationIndex  *uint16           `json:"location_index,omitempty"`
	Setup          uint32            `json:"setup,omitempty"`
	Service        uint32            `json:"service,omitempty"`
	SetupPerType   map[string]uint32 `json:"setup_per_type,omitempty"`
	ServicePerType map[string]uint32 `json:"service_per_type,omitempty"`
	TimeWindows    []TimeWindow      `json:"time_windows,omitempty"`
}
