package handlers

import (
    "net/http"
)

func LeaderboardHandler(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    w.Write([]byte(`{"message": "Leaderboard endpoint - use main service"}`))
}