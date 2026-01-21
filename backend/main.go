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
	ID            string `json:"id"`
	Username      string `json:"username"`
	UsernameLower string `json:"-"` // Pre-computed lowercase
	Rating        int    `json:"rating"`
	Rank          int    `json:"rank"`
}

type UserStore struct {
	// 1. TWO DATA STRUCTURES:
	// - Map for O(1) lookups by ID/username
	// - Sorted slice for leaderboard (cached)
	usersByID     map[string]*User
	usersByName   map[string]*User
	sortedUsers   []*User // Cached sorted version by rating
	
	// 2. OPTIMIZATION: For search - alphabetically sorted slice
	sortedByName  []*User // Sorted by UsernameLower
	
	// 3. OPTIMIZATION: First-character bucketing
	firstCharBuckets map[byte][]*User // Map first char -> users
	
	// 4. SYNC.RWMUTEX for concurrent reads
	mu sync.RWMutex
	
	// 5. CACHE for leaderboard pages
	cache      map[string]cacheEntry
	cacheMutex sync.RWMutex
	cacheTTL   time.Duration
	
	// 6. LAZY SORTING control
	needsSorting  bool
	updateCount   int64
	sortThreshold int // Sort after N updates
	
	// 7. PARTIAL UPDATES tracking
	updatedUsers map[string]bool // Track which users changed
	
	// 8. Stats
	totalUsers int64
	lastUpdate time.Time
}

type cacheEntry struct {
	data      []User
	timestamp time.Time
}

func NewUserStore() *UserStore {
	return &UserStore{
		usersByID:         make(map[string]*User),
		usersByName:       make(map[string]*User),
		sortedUsers:       make([]*User, 0),
		sortedByName:      make([]*User, 0),
		firstCharBuckets:  make(map[byte][]*User),
		cache:             make(map[string]cacheEntry),
		cacheTTL:          1 * time.Second,
		sortThreshold:     50, // Sort every 50 updates
		updatedUsers:      make(map[string]bool),
	}
}

func (s *UserStore) generateUsers(count int) {
	log.Printf("Generating %d users...", count)
	
	firstNames := []string{"Alex", "Aaron", "Alice", "Amy", "Andrew", "Anna", "Anthony", "Ashley",
		"Zack", "Zara", "Zane", "Zoe", "Zachary", "Zelda", "Zander", "Zuri",
		"Rahul", "Priya", "Amit", "Neha", "Vikas", "Sonia", "Raj", "Meera",
		"John", "Jane", "Mike", "Emma", "David", "Lisa", "Tom", "Sarah"}
	
	lastNames := []string{"Sharma", "Kumar", "Verma", "Patel", "Singh", "Reddy", "Joshi", "Das",
		"Smith", "Johnson", "Williams", "Brown", "Jones", "Garcia", "Miller", "Davis"}
	
	s.mu.Lock()
	defer s.mu.Unlock()
	
	// Clear buckets
	s.firstCharBuckets = make(map[byte][]*User)
	
	for i := 0; i < count; i++ {
		firstName := firstNames[rand.Intn(len(firstNames))]
		lastName := lastNames[rand.Intn(len(lastNames))]
		username := fmt.Sprintf("%s_%s%d", strings.ToLower(firstName), strings.ToLower(lastName), i+1)
		userID := fmt.Sprintf("user_%d", i+1)
		rating := 100 + rand.Intn(4901)
		
		user := &User{
			ID:            userID,
			Username:      username,
			UsernameLower: strings.ToLower(username), // Pre-compute lowercase
			Rating:        rating,
		}
		
		s.usersByID[userID] = user
		s.usersByName[username] = user
		s.sortedUsers = append(s.sortedUsers, user)
		s.sortedByName = append(s.sortedByName, user)
		
		// Add to first-character bucket
		if len(username) > 0 {
			firstChar := username[0]
			s.firstCharBuckets[firstChar] = append(s.firstCharBuckets[firstChar], user)
		}
	}
	
	// Initial sort by rating
	s.sortUsersLocked()
	
	// Sort alphabetically
	sort.Slice(s.sortedByName, func(i, j int) bool {
		return s.sortedByName[i].UsernameLower < s.sortedByName[j].UsernameLower
	})
	
	// Sort each bucket alphabetically
	for char, bucket := range s.firstCharBuckets {
		sort.Slice(bucket, func(i, j int) bool {
			return bucket[i].UsernameLower < bucket[j].UsernameLower
		})
		s.firstCharBuckets[char] = bucket
	}
	
	atomic.StoreInt64(&s.totalUsers, int64(count))
	s.lastUpdate = time.Now()
	
	// Log bucket distribution
	log.Printf("Generated %d users", count)
	log.Printf("Bucket distribution:")
	for char := byte('a'); char <= 'z'; char++ {
		if bucket, exists := s.firstCharBuckets[char]; exists {
			log.Printf("  %c: %d users", char, len(bucket))
		}
	}
}

// OPTIMIZATION: Lazy sorting - only sort when needed
func (s *UserStore) sortUsersLocked() {
	// Sort by rating descending
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
	
	s.needsSorting = false
	s.updatedUsers = make(map[string]bool) // Clear updated users
	s.updateCount = 0
	s.clearCache() // Clear cache when sorted
}

// OPTIMIZATION: Clear cache (thread-safe)
func (s *UserStore) clearCache() {
	s.cacheMutex.Lock()
	defer s.cacheMutex.Unlock()
	s.cache = make(map[string]cacheEntry)
}

// OPTIMIZATION: Update only affected users (partial update)
func (s *UserStore) updateRandomScores(count int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if len(s.sortedUsers) == 0 {
		return
	}
	
	updated := 0
	sameRating := 0
	
	for i := 0; i < count; i++ {
		// Pick random user
		idx := rand.Intn(len(s.sortedUsers))
		user := s.sortedUsers[idx]
		oldRating := user.Rating
		
		// Generate change
		change := rand.Intn(401) - 200
		newRating := user.Rating + change
		
		// Clamp to 100-5000
		if newRating < 100 {
			newRating = 100
		} else if newRating > 5000 {
			newRating = 5000
		}
		
		if newRating != oldRating {
			user.Rating = newRating
			s.updatedUsers[user.ID] = true
			s.updateCount++
			updated++
		} else {
			sameRating++
		}
	}
	
	// Mark that we need sorting
	if updated > 0 {
		s.needsSorting = true
		s.lastUpdate = time.Now()
		
		// OPTIMIZATION: Only sort if we've reached threshold
		if s.updateCount >= int64(s.sortThreshold) {
			s.sortUsersLocked()
			log.Printf("Update: Attempted=%d, Changed=%d, Unchanged=%d, Triggered sort", 
				count, updated, sameRating)
		} else {
			// Clear cache even without full sort
			s.clearCache()
			log.Printf("Update: Attempted=%d, Changed=%d, Unchanged=%d, Pending sorts=%d", 
				count, updated, sameRating, s.updateCount)
		}
	} else {
		log.Printf("Update: Attempted=%d, Changed=0, Unchanged=%d (no rating changes)", 
			count, sameRating)
	}
}

// OPTIMIZATION: Binary Search + First-Character Bucketing
func (s *UserStore) SearchUsers(query string, page, limit int) ([]User, int, int) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" || len(query) < 2 {
		return []User{}, 0, 0
	}
	
	// OPTIMIZATION: If needs sorting, we need to upgrade to write lock
	if s.needsSorting {
		s.mu.RUnlock()
		s.mu.Lock()
		s.sortUsersLocked()
		s.mu.Unlock()
		s.mu.RLock()
	}
	
	var results []User
	
	// OPTIMIZATION 1: Use first-character bucketing if possible
	firstChar := query[0]
	if bucket, exists := s.firstCharBuckets[firstChar]; exists {
		// We have a bucket for this first character
		startTime := time.Now()
		
		// OPTIMIZATION 2: Binary search within the bucket
		startIdx := sort.Search(len(bucket), func(i int) bool {
			return bucket[i].UsernameLower >= query
		})
		
		// OPTIMIZATION 3: Linear scan only from startIdx within bucket
		for i := startIdx; i < len(bucket); i++ {
			user := bucket[i]
			
			// Since bucket is sorted, we can break early
			if !strings.HasPrefix(user.UsernameLower, query) {
				break
			}
			
			results = append(results, *user)
			
			// Limit for performance
			if len(results) >= 1000 {
				break
			}
		}
		
		log.Printf("Search '%s': bucket size=%d, matches=%d, time=%v", 
			query, len(bucket), len(results), time.Since(startTime))
	} else {
		// No bucket for this character - fallback to binary search on full sorted list
		startTime := time.Now()
		
		// Binary search on full sorted list
		startIdx := sort.Search(len(s.sortedByName), func(i int) bool {
			return s.sortedByName[i].UsernameLower >= query
		})
		
		// Linear scan only from startIdx
		for i := startIdx; i < len(s.sortedByName); i++ {
			user := s.sortedByName[i]
			
			// Check if username starts with query (case-insensitive)
			if strings.HasPrefix(user.UsernameLower, query) {
				results = append(results, *user)
			} else {
				// Since slice is sorted, no more matches possible
				break
			}
			
			// Limit for performance
			if len(results) >= 1000 {
				break
			}
		}
		
		log.Printf("Search '%s': full scan, matches=%d, time=%v", 
			query, len(results), time.Since(startTime))
	}
	
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

// OPTIMIZATION: Cached leaderboard with RLock for concurrent reads
func (s *UserStore) GetLeaderboard(page, limit int) ([]User, int, int, int64) {
	// Check cache first
	cacheKey := fmt.Sprintf("lb:%d:%d", page, limit)
	s.cacheMutex.RLock()
	if entry, exists := s.cache[cacheKey]; exists {
		if time.Since(entry.timestamp) <= s.cacheTTL {
			total := len(s.sortedUsers)
			totalPages := (total + limit - 1) / limit
			s.cacheMutex.RUnlock()
			return entry.data, total, totalPages, s.updateCount
		}
	}
	s.cacheMutex.RUnlock()
	
	// OPTIMIZATION: Use RLock for concurrent reads
	s.mu.RLock()
	
	// If needs sorting, we need to upgrade to write lock
	if s.needsSorting {
		s.mu.RUnlock()
		s.mu.Lock()
		s.sortUsersLocked()
		s.mu.Unlock()
		s.mu.RLock()
	}
	
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 45
	}
	
	total := len(s.sortedUsers)
	start := (page - 1) * limit
	
	if start >= total {
		s.mu.RUnlock()
		return []User{}, total, 0, s.updateCount
	}
	
	end := start + limit
	if end > total {
		end = total
	}
	
	// Copy data while holding read lock
	users := make([]User, end-start)
	for i := start; i < end; i++ {
		users[i-start] = *s.sortedUsers[i]
	}
	
	totalPages := (total + limit - 1) / limit
	updateCount := s.updateCount
	
	s.mu.RUnlock()
	
	// Cache the result
	s.cacheMutex.Lock()
	s.cache[cacheKey] = cacheEntry{
		data:      users,
		timestamp: time.Now(),
	}
	s.cacheMutex.Unlock()
	
	return users, total, totalPages, updateCount
}

func (s *UserStore) GetUserRank(username string) (map[string]interface{}, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	user, exists := s.usersByName[username]
	if !exists {
		return nil, false
	}
	
	// Count ties (users with same rating)
	tieCount := 0
	for _, u := range s.sortedUsers {
		if u.Rating == user.Rating {
			tieCount++
		}
	}
	
	return map[string]interface{}{
		"user":          *user,
		"tieCount":      tieCount,
		"totalUsers":    atomic.LoadInt64(&s.totalUsers),
		"percentile":    float64(user.Rank) / float64(atomic.LoadInt64(&s.totalUsers)) * 100,
		"lastUpdate":    s.lastUpdate.Unix(),
		"needsSorting":  s.needsSorting,
		"pendingSorts":  s.updateCount,
	}, true
}

func (s *UserStore) GetStats() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	// Count A/Z names from buckets
	aCount := 0
	zCount := 0
	
	for char, bucket := range s.firstCharBuckets {
		if char == 'a' {
			aCount = len(bucket)
		}
		if char == 'z' {
			zCount = len(bucket)
		}
	}
	
	// Calculate bucket statistics
	bucketStats := make(map[string]int)
	for char, bucket := range s.firstCharBuckets {
		bucketStats[string(char)] = len(bucket)
	}
	
	return map[string]interface{}{
		"totalUsers":     atomic.LoadInt64(&s.totalUsers),
		"usersWithA":     aCount,
		"usersWithZ":     zCount,
		"pendingSorts":   s.updateCount,
		"needsSorting":   s.needsSorting,
		"updatedUsers":   len(s.updatedUsers),
		"cacheSize":      len(s.cache),
		"lastUpdate":     s.lastUpdate.Unix(),
		"sortThreshold":  s.sortThreshold,
		"bucketStats":    bucketStats,
		"optimizedSearch": "Binary Search + First-Char Bucketing",
		"bucketCount":    len(s.firstCharBuckets),
	}
}

//main
var userStore *UserStore

func init() {
	rand.Seed(time.Now().UnixNano())
	userStore = NewUserStore()
	userStore.generateUsers(20000)
	
	// Start auto-updates with random counts and intervals
	go func() {
		for {
			// Random count between 1 and 200 users
			updateCount := 1 + rand.Intn(200)
			userStore.updateRandomScores(updateCount)
			
			// Random interval between 1 and 10 seconds
			sleepSeconds := 1 + rand.Intn(10)
			time.Sleep(time.Duration(sleepSeconds) * time.Second)
		}
	}()
	
	log.Printf("✅ Optimized leaderboard initialized")
	log.Printf(" Users: 20,000")
	log.Printf("⚡ Optimizations:")
	log.Printf("   1. O(1) lookups with map")
	log.Printf("   2. Concurrent reads with RWMutex")
	log.Printf("   3. 1-second cache for leaderboard")
	log.Printf("   4. Lazy sorting (every 50 updates)")
	log.Printf("   5. Binary Search + First-Character Bucketing for search")
	log.Printf("   6. Pre-computed lowercase usernames")
	log.Printf("   7. Random update counts (1-200)")
	log.Printf("   8. Random update intervals (1-10 seconds)")
}

func corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Cache-Control", "no-store")
		
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		
		next(w, r)
	}
}

func leaderboardHandler(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	
	users, total, totalPages, pendingSorts := userStore.GetLeaderboard(page, limit)
	
	response := map[string]interface{}{
		"success":      true,
		"users":        users,
		"total":        total,
		"page":         page,
		"limit":        limit,
		"totalPages":   totalPages,
		"pendingSorts": pendingSorts,
		"timestamp":    time.Now().Unix(),
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func searchHandler(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	
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
		// Random count if not specified
		count = 1 + rand.Intn(200)
	}
	
	userStore.updateRandomScores(count)
	
	response := map[string]interface{}{
		"success":   true,
		"message":   fmt.Sprintf("Updated %d users (lazy sorting)", count),
		"timestamp": time.Now().Unix(),
	}
	
	json.NewEncoder(w).Encode(response)
}

func forceSortHandler(w http.ResponseWriter, r *http.Request) {
	userStore.mu.Lock()
	userStore.sortUsersLocked()
	userStore.mu.Unlock()
	
	response := map[string]interface{}{
		"success":   true,
		"message":   "Forced sort completed",
		"timestamp": time.Now().Unix(),
	}
	
	json.NewEncoder(w).Encode(response)
}

func main() {
	http.HandleFunc("/leaderboard", corsMiddleware(leaderboardHandler))
	http.HandleFunc("/search", corsMiddleware(searchHandler))
	http.HandleFunc("/user/rank", corsMiddleware(userRankHandler))
	http.HandleFunc("/stats", corsMiddleware(statsHandler))
	http.HandleFunc("/update", corsMiddleware(updateHandler))
	http.HandleFunc("/force-sort", corsMiddleware(forceSortHandler))
	http.HandleFunc("/health", corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":       "healthy",
			"users":        atomic.LoadInt64(&userStore.totalUsers),
			"optimization": "Binary Search + First-Char Bucketing",
			"timestamp":    time.Now().Unix(),
		})
	}))
	
	port := ":8080"
	log.Printf(" Optimized Server started on %s", port)
	log.Printf(" Total users: %d", atomic.LoadInt64(&userStore.totalUsers))
	log.Printf("⚡ Optimizations active:")
	log.Printf("   1. O(1) lookups with map")
	log.Printf("   2. Concurrent reads with RWMutex")
	log.Printf("   3. 1-second cache for leaderboard")
	log.Printf("   4. Lazy sorting (every 50 updates)")
	log.Printf("   5. Binary Search + First-Char Bucketing for search")
	log.Printf("   6. Partial updates only")
	log.Printf("   7. Random update counts (1-200 users)")
	log.Printf("   8. Random intervals (1-10 seconds)")
	
	log.Fatal(http.ListenAndServe(port, nil))
}