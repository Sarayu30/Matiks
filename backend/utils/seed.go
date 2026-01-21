package utils

import (
    "math/rand"
    "time"
)

func GenerateUsername(firstNames []string, lastNames []string) string {
    rand.Seed(time.Now().UnixNano())
    
    firstName := firstNames[rand.Intn(len(firstNames))]
    if rand.Intn(100) < 50 {
        return firstName
    }
    
    lastName := lastNames[rand.Intn(len(lastNames))]
    return firstName + "_" + lastName
}