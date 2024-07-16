package Loan_Applicants

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	_ "github.com/lib/pq" // Import the PostgreSQL driver
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

func SetupDatabase() {
	// create variable for connect to postgres name LMS_LoanApplicantsDB
	connStr := "postgres://Admin:Password@localhost:5432/LMS_LoanApplicantsDB?sslmode=disable"

	db, err := sql.Open("postgres", connStr)

	if err != nil {
		log.Fatal("Error connecting to the database:", err)
	}

	if err = db.Ping(); err != nil {
		log.Fatal("Error pinging the database:", err)
	}

	defer db.Close()

	createLoanApplicantTable(db)

	// Read data from JSON file
	loan_applicant, err := readCustomersFromFile("api/Loan_Applicants/json/Applicants.json")
	if err != nil {
		log.Fatal(err)
	}

	// Insert each customer into the database
	for _, loan_applicant := range loan_applicant {
		pk := InsertLoanApplicant(db, loan_applicant)
		fmt.Printf("Inserted loan applicant ID = %d\n", pk)
	}
}

func createLoanApplicantTable(db *sql.DB) {
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
		log.Fatal(err)
	}
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
		return nil, err
	}
	defer file.Close()

	var applicant []Loan_applicants
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&applicant); err != nil {
		return nil, err
	}

	return applicant, nil
}

func GetApplicants(w http.ResponseWriter, r *http.Request) {
	// Connect to the database
	connStr := "postgres://Admin:Password@localhost:5432/LMS_LoanApplicantsDB?sslmode=disable"

	db, err := sql.Open("postgres", connStr)
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
