package backend
package main

import (
	"fmt"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	password := "correctpassword"
	
	// Generate hash
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		fmt.Println("Error generating hash:", err)
		return
	}
	
	fmt.Printf("Generated hash for '%s': %s\n", password, string(hash))
	
	// Test the hash
	err = bcrypt.CompareHashAndPassword(hash, []byte(password))
	fmt.Println("Password match:", err == nil)
}