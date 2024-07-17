package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/SupachotT/Loan_Management_System.git/api/Loan_Applicants"
	"github.com/SupachotT/Loan_Management_System.git/api/Loan_Submits"
	"github.com/gorilla/mux"
)

func main() {
	// Call the function from the imported package
	Loan_Applicants.SetupDatabase()
	Loan_Submits.SetupDatabase()

	// Start server
	router := setupRouter()
	fmt.Println("Server listening on port 8080...")
	log.Fatal(http.ListenAndServe(":8080", router))
}

func setupRouter() *mux.Router {
	router := mux.NewRouter()

	// Define API endpoints
	router.HandleFunc("/loan_applicants/all", Loan_Applicants.GetApplicants).Methods("GET")
	router.HandleFunc("/loan_applicants/{id}", Loan_Applicants.GetApplicantByID).Methods("GET")
	router.HandleFunc("/loan_applicants/create", Loan_Applicants.CreateApplicants).Methods("POST")
	router.HandleFunc("/loan_applicants/update/{id}", Loan_Applicants.UpdateApplicants).Methods("PUT")
	router.HandleFunc("/loan_applicants/delete/{id}", Loan_Applicants.DeleteApplicants).Methods("DELETE")

	return router
}
