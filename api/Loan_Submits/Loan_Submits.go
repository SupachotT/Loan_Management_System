package Loan_Submits

import (
	"database/sql"
	"fmt"
	"log"
)

func connectDB() (*sql.DB, error) {
	connStr := "postgres://Admin:Password@localhost:5432/LMS_LoanSubmitDB?sslmode=disable"
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
	db, err := connectDB()
	if err != nil {
		log.Fatal("Error connecting to the database:", err)
	}
	defer db.Close()

	log.Println("Connected to the database successfully")

	// Create the loan_submits table if it doesn't exist
	if err := createLoanSubmitTable(db); err != nil {
		log.Fatal("Error creating loan_submits table:", err)
	}
}

func createLoanSubmitTable(db *sql.DB) error {
	query := `CREATE TABLE IF NOT EXISTS loan_submits (
		loanSubmit_id SERIAL PRIMARY KEY,
		applicant_id INT NOT NULL,
		loan_amount DECIMAL(15, 2) NOT NULL,
		interest_rate DECIMAL(5, 2) NOT NULL,
		loan_date DATE NOT NULL,
		due_date DATE NOT NULL,
		email VARCHAR(100) NOT NULL UNIQUE,
		loan_status VARCHAR(15) NOT NULL CHECK (loan_status IN ('ongoing', 'completed')),
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`

	_, err := db.Exec(query)
	if err != nil {
		return fmt.Errorf("error creating loan_submits table: %v", err)
	}
	return nil
}
