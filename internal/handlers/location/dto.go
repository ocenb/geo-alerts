package location

// @name CheckLocationRequest
type CheckReq struct {
	UserID    string   `json:"user_id" binding:"required,min=1,max=255"`
	Latitude  *float64 `json:"latitude" binding:"required,latitude"`
	Longitude *float64 `json:"longitude" binding:"required,longitude"`
}
