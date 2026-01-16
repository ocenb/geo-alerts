package geo

import (
	"math"
	"testing"
)

func TestDistance(t *testing.T) {
	tests := []struct {
		name       string
		lat1, lon1 float64
		lat2, lon2 float64
		want       float64
		delta      float64 // measurement error
	}{
		{
			name: "Same point",
			lat1: 55.7558, lon1: 37.6173,
			lat2: 55.7558, lon2: 37.6173,
			want:  0,
			delta: 0.1,
		},
		{
			name: "Moscow to St. Petersburg",
			lat1: 55.75, lon1: 37.62,
			lat2: 59.9386, lon2: 30.3141,
			want:  634000,
			delta: 5000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Distance(tt.lat1, tt.lon1, tt.lat2, tt.lon2)
			if math.Abs(got-tt.want) > tt.delta {
				t.Errorf("Distance() = %v, want %v (+/- %v)", got, tt.want, tt.delta)
			}
		})
	}
}
