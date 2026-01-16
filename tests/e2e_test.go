package tests

import (
	"flag"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var apiURL = flag.String("api-url", "http://localhost:8000/api/v1", "URL for the test API")
var apiKey = flag.String("api-key", "secret-operator-key", "API key")

func TestFullFlow(t *testing.T) {
	client := resty.New().
		SetTimeout(5 * time.Second).
		SetRetryCount(3).
		SetRetryWaitTime(1 * time.Second)

	// 1. Create Incident
	var res map[string]any
	resp, err := client.R().
		SetHeader("X-API-Key", *apiKey).
		SetBody(map[string]any{
			"latitude":  55.7558,
			"longitude": 37.6173,
			"radius":    1000,
		}).
		SetResult(&res).
		Post(*apiURL + "/incidents")

	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode())

	incIdVal, exists := res["id"]
	if !exists {
		t.Fatalf("Response JSON does not contain 'id'. Got: %v", res)
	}
	incIdValFloat, ok := incIdVal.(float64)
	if !ok {
		t.Fatalf("'id' is not a float64. Got type: %T, value: %v", incIdVal, incIdVal)
	}
	incidentID := int64(incIdValFloat)
	t.Logf("Created incident ID: %d", incidentID)

	// 2. Check Location (User in Danger)
	res = map[string]any{}
	resp, err = client.R().
		SetBody(map[string]any{
			"user_id":   "u1",
			"latitude":  55.7558,
			"longitude": 37.6173,
		}).
		SetResult(&res).
		Post(*apiURL + "/location/check")

	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode())
	hasDangerVal, exists := res["has_danger"]
	if !exists {
		t.Fatalf("Response JSON does not contain 'has_danger'. Got: %v", res)
	}
	hasDangerBool, ok := hasDangerVal.(bool)
	if !ok {
		t.Fatalf("'has_danger' is not a bool. Got type: %T, value: %v", hasDangerVal, hasDangerVal)
	}
	assert.True(t, hasDangerBool, "User should be in danger")

	// 3. Check Location (User Safe)
	res = map[string]any{}
	resp, err = client.R().
		SetBody(map[string]any{
			"user_id":   "u2",
			"latitude":  0.0,
			"longitude": 0.0,
		}).
		SetResult(&res).
		Post(*apiURL + "/location/check")

	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode())
	hasDangerVal, exists = res["has_danger"]
	if !exists {
		t.Fatalf("Response JSON does not contain 'has_danger'. Got: %v", res)
	}
	hasDangerBool, ok = hasDangerVal.(bool)
	if !ok {
		t.Fatalf("'has_danger' is not a bool. Got type: %T, value: %v", hasDangerVal, hasDangerVal)
	}
	assert.False(t, hasDangerBool, "User should be safe")

	// 4. Get Stats
	require.Eventually(t, func() bool {
		var statsRes []map[string]any

		resp, err := client.R().
			SetHeader("X-API-Key", *apiKey).
			SetResult(&statsRes).
			Get(*apiURL + "/incidents/stats")

		if err != nil || resp.StatusCode() != http.StatusOK {
			return false
		}

		for _, s := range statsRes {
			idVal, exists := s["incident_id"]
			if !exists {
				t.Logf("Response JSON does not contain 'incident_id'. Got: %v", s)
				return false
			}
			idFloat, ok := idVal.(float64)
			if !ok {
				t.Fatalf("'incident_id' is not a float64. Got type: %T, value: %v", idVal, idVal)
			}
			countVal, exists := s["user_count"]
			if !exists {
				t.Fatalf("Response JSON does not contain 'user_count'. Got: %v", s)
			}
			if int64(idFloat) == incidentID {
				count, ok := countVal.(float64)
				if !ok {
					t.Fatalf("'user_count' is not a float64. Got type: %T, value: %v", countVal, countVal)
				}
				return count >= 1.0
			}
		}
		return false
	}, 5*time.Second, 500*time.Millisecond, "Stats should show at least 1 user for the incident")

	// 5. Deactivate Incident
	resp, err = client.R().
		SetHeader("X-API-Key", *apiKey).
		Delete(fmt.Sprintf("%s/incidents/%d", *apiURL, incidentID))

	require.NoError(t, err)
	require.Equal(t, http.StatusNoContent, resp.StatusCode())

	// 6. Verify Deactivation
	res = map[string]any{}
	resp, err = client.R().
		SetBody(map[string]any{
			"user_id":   "u1",
			"latitude":  55.7558,
			"longitude": 37.6173,
		}).
		SetResult(&res).
		Post(*apiURL + "/location/check")

	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode())
	hasDangerVal, exists = res["has_danger"]
	if !exists {
		t.Fatalf("Response JSON does not contain 'has_danger'. Got: %v", res)
	}
	hasDangerBool, ok = hasDangerVal.(bool)
	if !ok {
		t.Fatalf("'has_danger' is not a bool. Got type: %T, value: %v", hasDangerVal, hasDangerVal)
	}
	assert.False(t, hasDangerBool, "Should be safe after deactivation")
}
