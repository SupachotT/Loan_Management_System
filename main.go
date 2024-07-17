package main

import (
	"fmt"
	"net/http"

	"github.com/SupachotT/Loan_Management_System.git/api/Loan_Applicants"
	"github.com/SupachotT/Loan_Management_System.git/api/Loan_Submits"
	"github.com/gorilla/mux"
)

func main() {
	// Call the function from the imported package
	Loan_Applicants.SetupDatabase()
	Loan_Submits.SetupDatabase()
	// Loan_Submits.TestReadSubmitFromFile()

	// Start server
	router := mux.NewRouter()
	handleRoutes(router)

	fmt.Println("Server listening on port 8080...")
	fmt.Println(http.ListenAndServe(":8080", router))
}

func handleRoutes(router *mux.Router) {
	// Define API endpoints for Loan Applicants
	applicantsRouter := router.PathPrefix("/loan_applicants").Subrouter()
	applicantsRouter.HandleFunc("/all", Loan_Applicants.GetApplicants).Methods("GET")
	applicantsRouter.HandleFunc("/{id}", Loan_Applicants.GetApplicantByID).Methods("GET")
	applicantsRouter.HandleFunc("/create", Loan_Applicants.CreateApplicants).Methods("POST")
	applicantsRouter.HandleFunc("/update/{id}", Loan_Applicants.UpdateApplicants).Methods("PUT")
	applicantsRouter.HandleFunc("/delete/{id}", Loan_Applicants.DeleteApplicants).Methods("DELETE")

	// Define API endpoints for Loan Submits
	submitsRouter := router.PathPrefix("/loan_submits").Subrouter()
	submitsRouter.HandleFunc("/all", Loan_Submits.GetLoanSubmit).Methods("GET")
	submitsRouter.HandleFunc("/{id}", Loan_Submits.GetLoanSubmitByID).Methods("GET")
	submitsRouter.HandleFunc("/create", Loan_Submits.CreateLoanSubmit).Methods("POST")
}
