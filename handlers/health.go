package handlers

import (
	"fmt"
	"net/http"
	"time"
)

// HealthResponse represents the health check response
type HealthResponse struct {
	Status string `json:"status" example:"healthy"`
	Time   string `json:"time" example:"2025-09-24T23:56:42+02:00"`
}

// HandleHealth handles health check requests
// @Summary Health check endpoint
// @Description Returns the current health status of the server
// @Tags System
// @Accept json
// @Produce json
// @Success 200 {object} HealthResponse "Server is healthy"
// @Router /health [get]
func HandleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	response := HealthResponse{
		Status: "healthy",
		Time:   time.Now().Format(time.RFC3339),
	}
	fmt.Fprintf(w, `{"status":"%s","time":"%s"}`, response.Status, response.Time)
}
