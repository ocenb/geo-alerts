package models

import (
	"time"
)

type CheckLocationParams struct {
	UserID    string
	Latitude  float64
	Longitude float64
}

// @name CheckLocationResult
type CheckLocationResult struct {
	UserID    string          `json:"user_id"`
	Latitude  float64         `json:"latitude"`
	Longitude float64         `json:"longitude"`
	HasDanger bool            `json:"has_danger"`
	Dangers   []IncidentShort `json:"dangers"`
	CreatedAt time.Time       `json:"created_at"`
}
