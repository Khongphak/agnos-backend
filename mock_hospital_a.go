//go:build ignore

// Mock server for Hospital A upstream API — for local manual testing only.
// Run: go run mock_hospital_a.go
// Then set HOSPITAL_A_BASE_URL=http://localhost:9001 when starting the backend.
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

func main() {
	http.HandleFunc("/patient/search/", func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Path[len("/patient/search/"):]
		log.Printf("mock hospital-a: GET /patient/search/%s", id)

		switch id {
		case "9999999999999":
			// Simulate 404
			w.WriteHeader(http.StatusNotFound)
		case "error":
			// Simulate 500
			w.WriteHeader(http.StatusInternalServerError)
		default:
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{
				"national_id":   id,
				"passport_id":   "",
				"first_name_th": "สมชาย",
				"last_name_th":  "ใจดี",
				"first_name_en": "Somchai",
				"last_name_en":  "Jaidee",
				"date_of_birth": "1990-01-15",
				"gender":        "male",
				"phone_number":  "0812345678",
				"email":         "somchai@example.com",
			})
		}
	})

	fmt.Println("Mock Hospital A running on :9001")
	fmt.Println("  GET /patient/search/<id>   → 200 with patient data")
	fmt.Println("  GET /patient/search/9999999999999 → 404")
	fmt.Println("  GET /patient/search/error  → 500")
	log.Fatal(http.ListenAndServe(":9001", nil))
}
