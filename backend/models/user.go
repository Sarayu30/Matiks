package models

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
