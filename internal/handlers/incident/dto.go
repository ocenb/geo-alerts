package incident

// @name ListIncidentsRequest
type ListReq struct {
	Limit  int `form:"limit" binding:"omitempty,min=1"`
	Offset int `form:"offset" binding:"omitempty,min=0"`
}

// @name CreateIncidentRequest
type CreateReq struct {
	Latitude  *float64 `json:"latitude" binding:"required,latitude"`
	Longitude *float64 `json:"longitude" binding:"required,longitude"`
	Radius    int      `json:"radius" binding:"required,min=1"`
}

// @name UpdateIncidentRequest
type UpdateReq struct {
	Latitude  *float64 `json:"latitude" binding:"required,latitude"`
	Longitude *float64 `json:"longitude" binding:"required,longitude"`
	Radius    int      `json:"radius" binding:"required,min=1"`
}
