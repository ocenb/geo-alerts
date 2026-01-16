package models

import (
	"time"
)

type CreateIncidentParams struct {
	Latitude  float64
	Longitude float64
	Radius    int
}

type UpdateIncidentParams struct {
	ID        int64
	Latitude  float64
	Longitude float64
	Radius    int
}

// @name Incident
type Incident struct {
	ID        int64     `json:"id"`
	Latitude  float64   `json:"latitude"`
	Longitude float64   `json:"longitude"`
	Radius    int       `json:"radius"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// @name IncidentShort
type IncidentShort struct {
	ID        int64   `json:"id"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Radius    int     `json:"radius"`
}

// @name Stats
type Stats struct {
	IncidentID int64   `json:"incident_id"`
	UserCount  int     `json:"user_count"`
	Latitude   float64 `json:"latitude"`
	Longitude  float64 `json:"longitude"`
}
