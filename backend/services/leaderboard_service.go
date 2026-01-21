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
    "time"
)

type User struct {
    ID       string \`json:"id"\`
    Username string \`json:"username"\`
    Rating   int    \`json:"rating"\`
    Rank     int    \`json:"rank"\`
}

type LeaderboardResponse struct {
    Users       []User \`json:"users"\`
    Total       int    \`json:"total"\`
    Page        int    \`json:"page"\`
    PageSize    int    \`json:"pageSize"\`
    TotalPages  int    \`json:"totalPages"\`
    LastUpdated string \`json:"lastUpdated"\`
}

type SearchResponse struct {
    Results []User \`json:"results"\`
    Count   int    \`json:"count"\`
    Query   string \`json:"query"\`
}

type LeaderboardService struct {
    users          []User
    usernameTrie   map[string][]int
    userMap        map[string]*User
    mu             sync.RWMutex
    nameParts      []string
    lastNames      []string
    updateCount    int
    lastUpdated    time.Time
}

func NewLeaderboardService() *LeaderboardService {
    return &LeaderboardService{
        users:        make([]User, 0),
        usernameTrie: make(map[string][]int),
        userMap:      make(map[string]*User),
        nameParts: []string{
            "rahul", "alex", "maria", "john", "sarah", "mike", "lisa", "david",
            "emma", "james", "sophia", "william", "olivia", "benjamin", "chloe",
            "leo", "mia", "daniel", "sophie", "ryan", "priya", "arjun", "ananya",
            "vikram", "neha", "karan", "kriti", "raj", "meera", "aditya",
        },
        lastNames: []string{
            "dev", "sharma", "patel", "kumar", "singh", "reddy", "naidu", "joshi",
            "gupta", "verma", "malhotra", "choudhary", "tiwari", "trivedi", "nair",
            "iyer", "menon", "pillai", "mehta", "bhatt", "desai", "jain", "modi",
            "thakur", "yadav", "das", "bose", "banerjee", "chatterjee", "mukherjee",
        },
        updateCount: 0,
        lastUpdated: time.Now(),
    }
}

func (s *LeaderboardService) SeedUsers(count int) {
    s.mu.Lock()
    defer s.mu.Unlock()
    
    s.users = make([]User, 0, count)
    s.usernameTrie = make(map[string][]int)
    s.userMap = make(map[string]*User)
    
    rand.Seed(time.Now().UnixNano())
    usedUsernames := make(map[string]bool)
    
    for i := 0; i < count; i++ {
        var username string
        
        for {
            firstName := s.nameParts[rand.Intn(len(s.nameParts))]
            
            // 70% chance to add last name/number
            if rand.Intn(100) < 70 {
                lastName := s.lastNames[rand.Intn(len(s.lastNames))]
                
                // Different username formats
                format := rand.Intn(3)
                switch format {
                case 0:
                    username = fmt.Sprintf("%s_%s", firstName, lastName)
                case 1:
                    username = fmt.Sprintf("%s_%s%d", firstName, lastName, rand.Intn(100))
                case 2:
                    username = fmt.Sprintf("%s%d", firstName, rand.Intn(1000))
                }
            } else {
                username = firstName
                if rand.Intn(100) < 30 {
                    username = fmt.Sprintf("%s%d", username, rand.Intn(100))
                }
            }
            
            if !usedUsernames[username] {
                usedUsernames[username] = true
                break
            }
        }
        
        rating := rand.Intn(4900) + 100
        
        user := User{
            ID:       fmt.Sprintf("user_%d", i),
            Username: username,
            Rating:   rating,
        }
        
        s.users = append(s.users, user)
        s.userMap[user.ID] = &s.users[i]
    }
    
    s.calculateRanks()
    s.buildTrie()
    log.Printf("✅ Seeded %d users with unique usernames", count)
}

func (s *LeaderboardService) buildTrie() {
    s.usernameTrie = make(map[string][]int)
    
    for i := range s.users {
        username := strings.ToLower(s.users[i].Username)
        
        // Add full username
        s.usernameTrie[username] = append(s.usernameTrie[username], i)
        
        // Add prefixes (starting from 2 chars)
        for length := 2; length <= len(username); length++ {
            prefix := username[:length]
            s.usernameTrie[prefix] = append(s.usernameTrie[prefix], i)
        }
    }
}

func (s *LeaderboardService) calculateRanks() {
    // Sort by rating descending
    sort.Slice(s.users, func(i, j int) bool {
        if s.users[i].Rating == s.users[j].Rating {
            return s.users[i].Username < s.users[j].Username
        }
        return s.users[i].Rating > s.users[j].Rating
    })
    
    // Calculate ranks with tie handling
    currentRank := 1
    for i := 0; i < len(s.users); i++ {
        if i > 0 && s.users[i].Rating < s.users[i-1].Rating {
            currentRank = i + 1
        }
        s.users[i].Rank = currentRank
    }
}

func (s *LeaderboardService) GetLeaderboardHandler(w http.ResponseWriter, r *http.Request) {
    s.mu.RLock()
    defer s.mu.RUnlock()
    
    // Get query parameters
    page, _ := strconv.Atoi(r.URL.Query().Get("page"))
    pageSize, _ := strconv.Atoi(r.URL.Query().Get("pageSize"))
    
    if page < 1 {
        page = 1
    }
    if pageSize < 1 || pageSize > 100 {
        pageSize = 45
    }
    
    // Calculate pagination
    total := len(s.users)
    totalPages := (total + pageSize - 1) / pageSize
    
    if page > totalPages {
        page = totalPages
    }
    
    start := (page - 1) * pageSize
    end := start + pageSize
    if end > total {
        end = total
    }
    
    // Get slice for current page
    var pageUsers []User
    if start < total {
        pageUsers = s.users[start:end]
    } else {
        pageUsers = []User{}
    }
    
    response := LeaderboardResponse{
        Users:       pageUsers,
        Total:       total,
        Page:        page,
        PageSize:    pageSize,
        TotalPages:  totalPages,
        LastUpdated: s.lastUpdated.Format(time.RFC3339),
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

func (s *LeaderboardService) SearchUsersHandler(w http.ResponseWriter, r *http.Request) {
    s.mu.RLock()
    defer s.mu.RUnlock()
    
    query := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("q")))
    if query == "" {
        json.NewEncoder(w).Encode(SearchResponse{
            Results: []User{},
            Count:   0,
            Query:   query,
        })
        return
    }
    
    // Use trie for prefix search
    seen := make(map[int]bool)
    var resultIndices []int
    
    // Check trie for prefix matches
    if indices, exists := s.usernameTrie[query]; exists {
        for _, idx := range indices {
            if !seen[idx] {
                resultIndices = append(resultIndices, idx)
                seen[idx] = true
            }
        }
    }
    
    // Also check if query is prefix of any username
    for i, user := range s.users {
        if strings.HasPrefix(strings.ToLower(user.Username), query) {
            if !seen[i] {
                resultIndices = append(resultIndices, i)
                seen[i] = true
            }
        }
        if len(resultIndices) >= 100 { // Limit results
            break
        }
    }
    
    // Sort by rank
    sort.Slice(resultIndices, func(i, j int) bool {
        return s.users[resultIndices[i]].Rank < s.users[resultIndices[j]].Rank
    })
    
    // Convert to users
    results := make([]User, 0, len(resultIndices))
    for _, idx := range resultIndices {
        results = append(results, s.users[idx])
    }
    
    response := SearchResponse{
        Results: results,
        Count:   len(results),
        Query:   query,
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

func (s *LeaderboardService) StartScoreUpdates(interval time.Duration) {
    ticker := time.NewTicker(interval)
    defer ticker.Stop()
    
    for range ticker.C {
        s.updateRandomScores()
    }
}

func (s *LeaderboardService) updateRandomScores() {
    s.mu.Lock()
    defer s.mu.Unlock()
    
    s.updateCount++
    
    // Update 1-2% of users each cycle
    updateCount := len(s.users) * (1 + rand.Intn(2)) / 100
    if updateCount < 10 {
        updateCount = 10
    }
    
    updatedUsers := 0
    for i := 0; i < updateCount; i++ {
        idx := rand.Intn(len(s.users))
        
        // Significant rating changes for visibility
        change := rand.Intn(200) - 80 // -80 to +119
        newRating := s.users[idx].Rating + change
        
        // Keep within bounds
        if newRating < 100 {
            newRating = 100
        } else if newRating > 5000 {
            newRating = 5000
        }
        
        if newRating != s.users[idx].Rating {
            s.users[idx].Rating = newRating
            updatedUsers++
        }
    }
    
    // Recalculate ranks
    s.calculateRanks()
    s.lastUpdated = time.Now()
    
    if updatedUsers > 0 {
        log.Printf("🔄 Update #%d: %d users' ratings changed", s.updateCount, updatedUsers)
    }
}
