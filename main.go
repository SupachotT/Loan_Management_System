package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/SupachotT/Loan_Management_System.git/api/Loan_Applicants"
	"github.com/gorilla/mux"
)

func main() {
	// Call the function from the imported package
	Loan_Applicants.SetupDatabase()

	router := mux.NewRouter()

	// Define API endpoints
	router.HandleFunc("/applicants/all", Loan_Applicants.GetApplicants).Methods("GET")

	// Start server
	fmt.Println("Server listening on port 8080...")
	log.Fatal(http.ListenAndServe(":8080", router))
}
