package Loan_Applicants

import (
	"database/sql"
	"log"

	_ "github.com/lib/pq" // Import the PostgreSQL driver
)

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
}

func createLoanApplicantTable(db *sql.DB) {
	query := `CREATE TABLE IF NOT EXISTS customers (
		customer_id SERIAL PRIMARY KEY,
		first_name VARCHAR(50) NOT NULL,
		last_name VARCHAR(50) NOT NULL,
		address VARCHAR(100) NOT NULL,
		phone VARCHAR(15) NOT NULL,
		email VARCHAR(100) NOT NULL UNIQUE,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`

	_, err := db.Exec(query)
	if err != nil {
		log.Fatal(err)
	}
}
