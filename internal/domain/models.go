package domain

import (
	"time"

	"github.com/google/uuid"
)

type TripStatus string

const (
	TripStatusActive    TripStatus = "active"
	TripStatusCompleted TripStatus = "completed"
)

type TripInput struct {
	UserID     uuid.UUID
	DriverID   uuid.UUID
	StartPoint Coordinates
	EndPoint   Coordinates
	Price      int64
}

type Trip struct {
	TripInput
	ID         uuid.UUID
	Status     TripStatus
	StartedAt  time.Time
	FinishedAt *time.Time
}

type Coordinates struct {
	Latitude  float64
	Longitude float64
}

type TripStatusHistory struct {
	TripID     uuid.UUID
	FromStatus *TripStatus
	ToStatus   TripStatus
	Reason     *string
	ChangedAt  time.Time
}
