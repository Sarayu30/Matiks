package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type User struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Rating   int    `json:"rating"` // 100-5000
	Rank     int    `json:"rank"`
}

type UserStore struct {
	users       []User
	usersLock   sync.RWMutex
	
	// Indexes
	userIDIndex    map[int]*User
	usernameIndex  map[string]*User
	ratingIndex    map[int][]*User
	
	// Cache
	sortedUsers    []*User
	
	// Stats
	totalUsers     int64
	updateCounter  int64
	lastUpdateTime int64 // Unix timestamp
	
	// Update control
	updateInterval time.Duration
	updateTicker   *time.Ticker
	stopUpdates    chan bool
}

func NewUserStore(userCount int) *UserStore {
	store := &UserStore{
		userIDIndex:    make(map[int]*User),
		usernameIndex:  make(map[string]*User),
		ratingIndex:    make(map[int][]*User),
		updateInterval: 3 * time.Second,
		stopUpdates:    make(chan bool),
	}
	
	store.generateUsers(userCount)
	store.rebuildIndexes()
	
	// Start auto updates
	go store.startAutoUpdates()
	
	return store
}

func (s *UserStore) generateUsers(count int) {
	s.users = make([]User, count)
	
	firstNames := []string{
		"Alex", "Aaron", "Alice", "Amy", "Andrew", "Anna", "Anthony", "Ashley",
		"Zack", "Zara", "Zane", "Zoe", "Zachary", "Zelda", "Zander", "Zuri",
		"Rahul", "Priya", "Amit", "Neha", "Vikas", "Sonia", "Raj", "Meera",
		"John", "Jane", "Mike", "Emma", "David", "Lisa", "Tom", "Sarah",
		"Bob", "Charlie", "Diana", "Eve", "Frank", "Grace", "Henry", "Ivy",
	}
	
	lastNames := []string{
		"Sharma", "Kumar", "Verma", "Patel", "Singh", "Reddy", "Joshi", "Das",
		"Smith", "Johnson", "Williams", "Brown", "Jones", "Garcia", "Miller", "Davis",
		"Anderson", "Thomas", "Jackson", "White", "Harris", "Martin", "Thompson", "Moore",
		"Wilson", "Taylor", "Clark", "Lewis", "Walker", "Hall", "Allen", "Young",
	}
	
	usernameCounter := make(map[string]int)
	
	for i := 0; i < count; i++ {
		firstName := firstNames[rand.Intn(len(firstNames))]
		lastName := lastNames[rand.Intn(len(lastNames))]
		
		baseUsername := strings.ToLower(firstName) + "_" + strings.ToLower(lastName)
		counter := usernameCounter[baseUsername] + 1
		usernameCounter[baseUsername] = counter
		
		username := fmt.Sprintf("%s%d", baseUsername, counter)
		
		// Generate realistic rating distribution
		rating := 100 + rand.Intn(4901) // 100-5000
		
		s.users[i] = User{
			ID:       i + 1,
			Username: username,
			Rating:   rating,
		}
	}
	
	atomic.StoreInt64(&s.totalUsers, int64(count))
}

func (s *UserStore) rebuildIndexes() {
	s.usersLock.Lock()
	defer s.usersLock.Unlock()
	
	s.userIDIndex = make(map[int]*User)
	s.usernameIndex = make(map[string]*User)
	s.ratingIndex = make(map[int][]*User)
	
	for i := range s.users {
		user := &s.users[i]
		s.userIDIndex[user.ID] = user
		s.usernameIndex[user.Username] = user
		s.ratingIndex[user.Rating] = append(s.ratingIndex[user.Rating], user)
	}
	
	// Sort users
	s.sortedUsers = make([]*User, len(s.users))
	for i := range s.users {
		s.sortedUsers[i] = &s.users[i]
	}
	
	sort.Slice(s.sortedUsers, func(i, j int) bool {
		if s.sortedUsers[i].Rating == s.sortedUsers[j].Rating {
			return s.sortedUsers[i].ID < s.sortedUsers[j].ID
		}
		return s.sortedUsers[i].Rating > s.sortedUsers[j].Rating
	})
	
	// Assign ranks with ties
	currentRank := 1
	for i := 0; i < len(s.sortedUsers); {
		currentRating := s.sortedUsers[i].Rating
		
		j := i
		for j < len(s.sortedUsers) && s.sortedUsers[j].Rating == currentRating {
			s.sortedUsers[j].Rank = currentRank
			j++
		}
		
		currentRank += (j - i)
		i = j
	}
	
	atomic.StoreInt64(&s.lastUpdateTime, time.Now().Unix())
}

func (s *UserStore) updateRandomScores(count int) int {
	s.usersLock.Lock()
	defer s.usersLock.Unlock()
	
	if count <= 0 {
		count = max(1, len(s.users)/50) // 2% of users
	}
	
	updated := 0
	updatedUserIDs := make(map[int]bool)
	
	for i := 0; i < count; i++ {
		// Pick random user, avoiding duplicates
		var user *User
		attempts := 0
		for attempts < 10 {
			idx := rand.Intn(len(s.users))
			if !updatedUserIDs[idx] {
				user = &s.users[idx]
				updatedUserIDs[idx] = true
				break
			}
			attempts++
		}
		
		if user == nil {
			continue
		}
		
		oldRating := user.Rating
		
		// Generate change (-300 to +300)
		change := rand.Intn(601) - 300
		newRating := oldRating + change
		
		// Clamp to 100-5000
		if newRating < 100 {
			newRating = 100
		} else if newRating > 5000 {
			newRating = 5000
		}
		
		if newRating != oldRating {
			// Remove from old rating group
			if oldGroup, exists := s.ratingIndex[oldRating]; exists {
				for j, u := range oldGroup {
					if u.ID == user.ID {
						s.ratingIndex[oldRating] = append(oldGroup[:j], oldGroup[j+1:]...)
						break
					}
				}
			}
			
			// Update user
			user.Rating = newRating
			
			// Add to new rating group
			s.ratingIndex[newRating] = append(s.ratingIndex[newRating], user)
			
			updated++
		}
	}
	
	if updated > 0 {
		// Re-sort only if we have changes
		sort.Slice(s.sortedUsers, func(i, j int) bool {
			if s.sortedUsers[i].Rating == s.sortedUsers[j].Rating {
				return s.sortedUsers[i].ID < s.sortedUsers[j].ID
			}
			return s.sortedUsers[i].Rating > s.sortedUsers[j].Rating
		})
		
		// Update ranks
		currentRank := 1
		for i := 0; i < len(s.sortedUsers); {
			currentRating := s.sortedUsers[i].Rating
			
			j := i
			for j < len(s.sortedUsers) && s.sortedUsers[j].Rating == currentRating {
				s.sortedUsers[j].Rank = currentRank
				j++
			}
			
			currentRank += (j - i)
			i = j
		}
		
		atomic.StoreInt64(&s.lastUpdateTime, time.Now().Unix())
		atomic.AddInt64(&s.updateCounter, int64(updated))
		
		log.Printf("🔄 Updated %d users (Total updates: %d)", 
			updated, atomic.LoadInt64(&s.updateCounter))
	}
	
	return updated
}

func (s *UserStore) startAutoUpdates() {
	s.updateTicker = time.NewTicker(s.updateInterval)
	
	for {
		select {
		case <-s.updateTicker.C:
			// Update 2% of users every tick
			updateCount := max(100, len(s.users)/50)
			s.updateRandomScores(updateCount)
			
		case <-s.stopUpdates:
			s.updateTicker.Stop()
			return
		}
	}
}

func (s *UserStore) GetLeaderboard(page, limit int) ([]User, int, int, int64) {
	s.usersLock.RLock()
	defer s.usersLock.RUnlock()
	
	// Validate and set defaults
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 45
	}
	
	totalUsers := len(s.sortedUsers)
	
	// Calculate start index
	start := (page - 1) * limit
	
	// If start is beyond total users, return empty
	if start >= totalUsers {
		return []User{}, totalUsers, 1, atomic.LoadInt64(&s.updateCounter)
	}
	
	// Calculate end index
	end := start + limit
	if end > totalUsers {
		end = totalUsers
	}
	
	// Extract page data
	result := make([]User, end-start)
	for i := start; i < end; i++ {
		result[i-start] = *s.sortedUsers[i]
	}
	
	// Calculate total pages
	totalPages := (totalUsers + limit - 1) / limit
	
	return result, totalUsers, totalPages, atomic.LoadInt64(&s.updateCounter)
}

func (s *UserStore) SearchUsers(query string, page, limit int) ([]User, int, int) {
	s.usersLock.RLock()
	defer s.usersLock.RUnlock()
	
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return []User{}, 0, 0
	}
	
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 45
	}
	
	var results []User
	
	// Search all users
	for _, user := range s.users {
		if strings.HasPrefix(strings.ToLower(user.Username), query) {
			results = append(results, user)
		}
	}
	
	// Sort by rank
	sort.Slice(results, func(i, j int) bool {
		return results[i].Rank < results[j].Rank
	})
	
	total := len(results)
	start := (page - 1) * limit
	
	if start >= total {
		return []User{}, total, 0
	}
	
	end := start + limit
	if end > total {
		end = total
	}
	
	return results[start:end], total, (total + limit - 1) / limit
}

func (s *UserStore) GetUserRank(username string) (map[string]interface{}, bool) {
	s.usersLock.RLock()
	defer s.usersLock.RUnlock()
	
	user, exists := s.usernameIndex[username]
	if !exists {
		return nil, false
	}
	
	sameRatingUsers := s.ratingIndex[user.Rating]
	positionInTie := 1
	for _, u := range sameRatingUsers {
		if u.ID == user.ID {
			break
		}
		positionInTie++
	}
	
	result := map[string]interface{}{
		"user":          *user,
		"tieCount":      len(sameRatingUsers),
		"positionInTie": positionInTie,
		"totalUsers":    len(s.users),
		"percentile":    float64(user.Rank) / float64(len(s.users)) * 100,
		"lastUpdate":    atomic.LoadInt64(&s.lastUpdateTime),
	}
	
	return result, true
}

func (s *UserStore) GetStats() map[string]interface{} {
	s.usersLock.RLock()
	defer s.usersLock.RUnlock()
	
	// Count A/Z names
	aCount := 0
	zCount := 0
	for _, user := range s.users {
		lowerUsername := strings.ToLower(user.Username)
		if strings.HasPrefix(lowerUsername, "a") {
			aCount++
		}
		if strings.HasPrefix(lowerUsername, "z") {
			zCount++
		}
	}
	
	// Top 10
	top10 := make([]User, 0, 10)
	for i := 0; i < 10 && i < len(s.sortedUsers); i++ {
		top10 = append(top10, *s.sortedUsers[i])
	}
	
	return map[string]interface{}{
		"totalUsers":      len(s.users),
		"usersWithA":      aCount,
		"usersWithZ":      zCount,
		"topUsers":        top10,
		"lastUpdateTime":  atomic.LoadInt64(&s.lastUpdateTime),
		"totalUpdates":    atomic.LoadInt64(&s.updateCounter),
		"updateInterval":  s.updateInterval.String(),
		"memoryUsageMB":   float64(len(s.users)*32) / 1024 / 1024,
	}
}

func (s *UserStore) Stop() {
	if s.updateTicker != nil {
		s.stopUpdates <- true
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

var userStore *UserStore

func init() {
	rand.Seed(time.Now().UnixNano())
	
	userCount := 20000
	userStore = NewUserStore(userCount)
	
	log.Printf("✅ Leaderboard initialized with %d users", userCount)
	log.Printf("⚡ Auto-updates every 3 seconds")
}

func corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")
		
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		
		next(w, r)
	}
}

func leaderboardHandler(w http.ResponseWriter, r *http.Request) {
	// Get query parameters
	query := r.URL.Query()
	pageStr := query.Get("page")
	limitStr := query.Get("limit")
	
	// Parse with defaults
	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}
	
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 {
		limit = 45
	}
	
	// Get data
	users, total, totalPages, updateCount := userStore.GetLeaderboard(page, limit)
	
	response := map[string]interface{}{
		"success":     true,
		"users":       users,
		"total":       total,
		"page":        page,
		"limit":       limit,
		"totalPages":  totalPages,
		"updateCount": updateCount,
		"timestamp":   time.Now().Unix(),
		"lastUpdate":  atomic.LoadInt64(&userStore.lastUpdateTime),
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func searchHandler(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 45
	}
	
	users, total, totalPages := userStore.SearchUsers(query, page, limit)
	
	response := map[string]interface{}{
		"success":    true,
		"users":      users,
		"total":      total,
		"page":       page,
		"limit":      limit,
		"totalPages": totalPages,
		"timestamp":  time.Now().Unix(),
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func userRankHandler(w http.ResponseWriter, r *http.Request) {
	username := r.URL.Query().Get("username")
	
	rankInfo, found := userStore.GetUserRank(username)
	if !found {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "User not found",
		})
		return
	}
	
	response := map[string]interface{}{
		"success": true,
		"data":    rankInfo,
	}
	
	json.NewEncoder(w).Encode(response)
}

func statsHandler(w http.ResponseWriter, r *http.Request) {
	stats := userStore.GetStats()
	
	response := map[string]interface{}{
		"success": true,
		"stats":   stats,
	}
	
	json.NewEncoder(w).Encode(response)
}

func updateHandler(w http.ResponseWriter, r *http.Request) {
	count, _ := strconv.Atoi(r.URL.Query().Get("count"))
	if count <= 0 {
		count = 200
	}
	
	updated := userStore.updateRandomScores(count)
	
	response := map[string]interface{}{
		"success":  true,
		"updated":  updated,
		"message":  fmt.Sprintf("Updated %d users", updated),
		"timestamp": time.Now().Unix(),
	}
	
	json.NewEncoder(w).Encode(response)
}

func demoHandler(w http.ResponseWriter, r *http.Request) {
	// Big update for demo
	updated := userStore.updateRandomScores(1000)
	
	response := map[string]interface{}{
		"success":  true,
		"updated":  updated,
		"message":  "Big demo update! 1000 users modified",
		"timestamp": time.Now().Unix(),
	}
	
	json.NewEncoder(w).Encode(response)
}

func main() {
	defer func() {
		if userStore != nil {
			userStore.Stop()
		}
	}()
	
	// Routes
	http.HandleFunc("/leaderboard", corsMiddleware(leaderboardHandler))
	http.HandleFunc("/search", corsMiddleware(searchHandler))
	http.HandleFunc("/user/rank", corsMiddleware(userRankHandler))
	http.HandleFunc("/stats", corsMiddleware(statsHandler))
	http.HandleFunc("/update", corsMiddleware(updateHandler))
	http.HandleFunc("/demo", corsMiddleware(demoHandler))
	http.HandleFunc("/health", corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":    "healthy",
			"users":     atomic.LoadInt64(&userStore.totalUsers),
			"updates":   atomic.LoadInt64(&userStore.updateCounter),
			"timestamp": time.Now().Unix(),
		})
	}))
	
	port := ":8080"
	log.Printf("🚀 Server started on %s", port)
	log.Printf("📊 Total users: %d", atomic.LoadInt64(&userStore.totalUsers))
	log.Printf("⚡ Auto-updates: Every 3 seconds")
	log.Printf("🎯 Names with A/Z included")
	
	log.Fatal(http.ListenAndServe(port, nil))
}