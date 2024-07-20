package Loan_Applicants

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/lib/pq"
)

type Loan_applicants struct {
	Applicant_id     int
	First_name       string
	Last_name        string
	Address          string
	Phone            string
	Email            string
	Applicant_Status string
	Created_at       string
	Updated_at       string
}

func connectLMS_LoanApplicantsDB() (*sql.DB, error) {
	connStr := "postgres://Admin:Password@localhost:5432/LMS_LoanApplicantsDB?sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("error connecting to the database: %v", err)
	}
	if err = db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("error pinging the database: %v", err)
	}
	return db, nil
}

func SetupDatabase() {
	db, err := connectLMS_LoanApplicantsDB()
	if err != nil {
		log.Fatal("Error connecting to the database:", err)
	}
	defer db.Close()

	// Create the loan_applicants table if it doesn't exist
	if err := createLoanApplicantTable(db); err != nil {
		log.Fatal("Error creating loan_applicants table:", err)
	}

	// Read data from JSON file
	loan_applicant, err := readCustomersFromFile("json/Applicants.json")
	if err != nil {
		log.Fatal(err)
	}

	// Insert each applicants into the database
	for _, loan_applicant := range loan_applicant {
		pk := InsertLoanApplicant(db, loan_applicant)
		fmt.Printf("Inserted loan applicant ID = %d\n", pk)
	}
}

func createLoanApplicantTable(db *sql.DB) error {
	query := `CREATE TABLE IF NOT EXISTS loan_applicants (
		applicant_id SERIAL PRIMARY KEY,
		first_name VARCHAR(50) NOT NULL,
		last_name VARCHAR(50) NOT NULL,
		address VARCHAR(100) NOT NULL,
		phone VARCHAR(15) NOT NULL,
		email VARCHAR(100) NOT NULL UNIQUE,
		applicant_status VARCHAR(15) NOT NULL CHECK (applicant_status IN ('newBorrower', 'currentBorrower')),
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`

	_, err := db.Exec(query)
	if err != nil {
		return fmt.Errorf("error creating loan_applicants table: %v", err)
	}
	return nil
}

func InsertLoanApplicant(db *sql.DB, applicant Loan_applicants) int {
	query := `INSERT INTO loan_applicants (first_name, last_name, address, phone, email, applicant_status)
		VALUES ($1, $2, $3, $4, $5, $6) RETURNING applicant_id`

	var pk int
	err := db.QueryRow(query, applicant.First_name, applicant.Last_name, applicant.Address, applicant.Phone, applicant.Email, applicant.Applicant_Status).Scan(&pk)
	if err != nil {
		log.Fatal(err)
	}
	return pk
}

func readCustomersFromFile(filename string) ([]Loan_applicants, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("error opening file: %v", err)
	}
	defer file.Close()

	var applicant []Loan_applicants
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&applicant); err != nil {
		return nil, fmt.Errorf("error decoding JSON: %v", err)
	}

	return applicant, nil
}

func GetApplicants(w http.ResponseWriter, r *http.Request) {
	db, err := connectLMS_LoanApplicantsDB()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer db.Close()

	// Query from the loan_applicants table
	rows, err := db.Query("SELECT applicant_id, first_name, last_name, address, phone, email, applicant_status, created_at, updated_at FROM loan_applicants")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var loanApplicants []Loan_applicants
	for rows.Next() {
		var loanApplicant Loan_applicants
		if err := rows.Scan(&loanApplicant.Applicant_id, &loanApplicant.First_name, &loanApplicant.Last_name, &loanApplicant.Address, &loanApplicant.Phone, &loanApplicant.Email, &loanApplicant.Applicant_Status, &loanApplicant.Created_at, &loanApplicant.Updated_at); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		loanApplicants = append(loanApplicants, loanApplicant)
	}
	if err := rows.Err(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Return JSON response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(loanApplicants)
}

func GetApplicantByID(w http.ResponseWriter, r *http.Request) {
	db, err := connectLMS_LoanApplicantsDB()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer db.Close()

	// Get Loan_applicants from URL parameters
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid applicant ID", http.StatusBadRequest)
		return
	}

	// Query database for Loan_applicants with given applicant_id
	var loanApplicant Loan_applicants
	query := `SELECT applicant_id, first_name, last_name, address, phone, email, applicant_status, created_at, updated_at FROM loan_applicants WHERE applicant_id = $1`
	err = db.QueryRow(query, id).Scan(&loanApplicant.Applicant_id, &loanApplicant.First_name, &loanApplicant.Last_name, &loanApplicant.Address, &loanApplicant.Phone, &loanApplicant.Email, &loanApplicant.Applicant_Status, &loanApplicant.Created_at, &loanApplicant.Updated_at)
	if err == sql.ErrNoRows {
		// Return JSON error response if no customer with the given ID exists
		errorResponse := map[string]string{"error": "applicant not found"}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(errorResponse)
		return
	} else if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Return JSON response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(loanApplicant)
}

func CreateApplicants(w http.ResponseWriter, r *http.Request) {
	db, err := connectLMS_LoanApplicantsDB()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer db.Close()

	// Decode JSON request body into a Loan_applicants struct
	var newApplicant Loan_applicants
	err = json.NewDecoder(r.Body).Decode(&newApplicant)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Insert query
	query := `INSERT INTO loan_applicants (first_name, last_name, address, phone, email, applicant_status) VALUES ($1, $2, $3, $4, $5, $6) RETURNING applicant_id`
	var newApplicantID int
	err = db.QueryRow(query, newApplicant.First_name, newApplicant.Last_name, newApplicant.Address, newApplicant.Phone, newApplicant.Email, newApplicant.Applicant_Status).Scan(&newApplicantID)
	if err != nil {
		pgErr, ok := err.(*pq.Error)
		if ok && pgErr.Code.Name() == "unique_violation" {
			// If the error is due to duplicate email, return a specific JSON response
			errorResponse := map[string]string{
				"error": fmt.Sprintf("Email '%s' already exists", newApplicant.Email),
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusConflict) // HTTP 409 Conflict
			json.NewEncoder(w).Encode(errorResponse)
			return
		}

		// For other errors, return a generic internal server error
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Prepare success message
	successMessage := map[string]interface{}{
		"message":     "Customer information has been successfully created.",
		"customer_id": newApplicantID,
	}

	// Set Content-Type and return JSON response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated) // HTTP 201 Created
	json.NewEncoder(w).Encode(successMessage)
}

func UpdateApplicants(w http.ResponseWriter, r *http.Request) {
	db, err := connectLMS_LoanApplicantsDB()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer db.Close()

	// Get applicant_id from URL parameters
	vars := mux.Vars(r)
	idStr := vars["id"]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid Applicants ID", http.StatusBadRequest)
		return
	}

	// Decode JSON request body into a Loan_applicants struct
	var updateApplicant Loan_applicants
	err = json.NewDecoder(r.Body).Decode(&updateApplicant)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Validate applicant_status
	if updateApplicant.Applicant_Status != "newBorrower" && updateApplicant.Applicant_Status != "currentBorrower" {
		// Return JSON error response if applicant_status is invalid
		errorResponse := map[string]string{"error": "Invalid applicant status. Allowed values are 'newBorrower' or 'currentBorrower'"}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(errorResponse)
		return
	}

	// Update query
	query := `UPDATE loan_applicants SET first_name = $2, last_name = $3, address = $4, phone = $5, email = $6, applicant_status = $7 WHERE applicant_id = $1`
	result, err := db.Exec(query, id, updateApplicant.First_name, updateApplicant.Last_name, updateApplicant.Address, updateApplicant.Phone, updateApplicant.Email, updateApplicant.Applicant_Status)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Check if any rows were affected
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		// Return JSON error response if no applicant with the given ID was found to update
		errorResponse := map[string]string{"error": "Applicant ID not found or no update performed"}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(errorResponse)
		return
	}

	// Return success message
	w.WriteHeader(http.StatusOK)
	successMessage := map[string]string{"message": fmt.Sprintf("Applicant with ID %d updated successfully", id)}
	json.NewEncoder(w).Encode(successMessage)
}

func DeleteApplicants(w http.ResponseWriter, r *http.Request) {
	db, err := connectLMS_LoanApplicantsDB()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer db.Close()

	// Get applicant_id from URL parameters
	vars := mux.Vars(r)
	idStr := vars["id"]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid applicant ID", http.StatusBadRequest)
		return
	}

	// Delete query
	query := `DELETE FROM loan_applicants WHERE applicant_id = $1`
	result, err := db.Exec(query, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Check if any rows were affected
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		// Return JSON error response if no customer with the given ID was found to delete
		errorResponse := map[string]string{"error": "Applicant ID not found or no delete performed"}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(errorResponse)
		return
	}

	// Return success message
	w.WriteHeader(http.StatusOK)
	successMessage := map[string]string{"message": fmt.Sprintf("Applicant with ID %d deleted successfully", id)}
	json.NewEncoder(w).Encode(successMessage)
}
