package Loan_Submits

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/shopspring/decimal"
)

// ----------------------------- solve problem cannot insert data from file json into database -----------------------------
// Custom date type for parsing dates in YYYY-MM-DD format
type CustomDate struct {
	time.Time
}

const customDateFormat = "2006-01-02"

func (cd *CustomDate) UnmarshalJSON(b []byte) error {
	dateString := string(b)
	if dateString == `null` {
		*cd = CustomDate{time.Time{}}
		return nil
	}
	dateString = dateString[1 : len(dateString)-1]

	t, err := time.Parse(customDateFormat, dateString)
	if err != nil {
		return err
	}
	cd.Time = t
	return nil
}

// Scan implements the sql.Scanner interface.
func (cd *CustomDate) Scan(value interface{}) error {
	if value == nil {
		*cd = CustomDate{Time: time.Time{}}
		return nil
	}

	switch v := value.(type) {
	case time.Time:
		*cd = CustomDate{Time: v}
		return nil
	case string:
		t, err := time.Parse(customDateFormat, v)
		if err != nil {
			return err
		}
		*cd = CustomDate{Time: t}
		return nil
	default:
		return fmt.Errorf("unsupported Scan, storing driver.Value type %T into type %T", value, cd)
	}
}

// Value implements the driver.Valuer interface.
func (cd CustomDate) Value() (driver.Value, error) {
	return cd.Time.Format(customDateFormat), nil
}

//----------------------------- end for solve problem cannot insert data from file json into database -----------------------------

type LoanSubmit struct {
	LoanSubmitID int
	ApplicantID  int
	LoanAmount   decimal.Decimal
	InterestRate decimal.Decimal
	LoanDate     CustomDate // Use CustomDate
	DueDate      CustomDate // Use CustomDate
	LoanStatus   string
	CreatedAt    string
	UpdatedAt    string
}

func connectLoanSubmitDB() (*sql.DB, error) {
	connStr := "postgres://Admin:Password@localhost:5432/LMS_LoanSubmitsDB?sslmode=disable"
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
	db, err := connectLoanSubmitDB()
	if err != nil {
		log.Fatal("Error connecting to the database:", err)
	}
	defer db.Close()

	// Create the loan_submits table if it doesn't exist
	if err := createLoanSubmitTable(db); err != nil {
		log.Fatal("Error creating loan_submits table:", err)
	}

	// Read data from JSON file
	loanSubmits, err := readSubmitFromFile("json/SubmittedApp.json")
	if err != nil {
		log.Fatal(err)
	}

	// Insert each applicant into the database
	for _, Submit := range loanSubmits {
		pk := InsertLoanSubmit(db, Submit)
		fmt.Printf("Inserted loan submit ID = %d\n", pk)
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

func InsertLoanSubmit(db *sql.DB, loanSubmitt LoanSubmit) int {
	query := `INSERT INTO loan_submits (applicant_id, loan_amount, interest_rate, loan_date, due_date, loan_status)
            VALUES ($1, $2, $3, $4, $5, $6) RETURNING loanSubmit_id`

	var loanSubmitID int
	// Format time.Time to PostgreSQL DATE format
	loanDate := loanSubmitt.LoanDate.Format("2006-01-02")
	dueDate := loanSubmitt.DueDate.Format("2006-01-02")

	err := db.QueryRow(query, loanSubmitt.ApplicantID, loanSubmitt.LoanAmount, loanSubmitt.InterestRate, loanDate, dueDate, loanSubmitt.LoanStatus).Scan(&loanSubmitID)
	if err != nil {
		log.Fatal(err)
	}
	return loanSubmitID
}

func readSubmitFromFile(filename string) ([]LoanSubmit, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("error opening file: %v", err)
	}
	defer file.Close()

	var loanSubmits []LoanSubmit
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&loanSubmits); err != nil {
		return nil, fmt.Errorf("error decoding JSON: %v", err)
	}

	return loanSubmits, nil
}

func GetLoanSubmit(w http.ResponseWriter, r *http.Request) {
	db, err := connectLoanSubmitDB()
	if err != nil {
		log.Fatal("Error connecting to the database:", err)
	}
	defer db.Close()

	// Query from the loan_submits table
	rows, err := db.Query("SELECT loanSubmit_id, applicant_id, loan_amount, interest_rate, loan_date, due_date, loan_status, created_at, updated_at FROM loan_submits")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var loan_Submits []LoanSubmit
	for rows.Next() {
		var loanSubmit LoanSubmit
		if err := rows.Scan(&loanSubmit.LoanSubmitID, &loanSubmit.ApplicantID, &loanSubmit.LoanAmount, &loanSubmit.InterestRate,
			&loanSubmit.LoanDate, &loanSubmit.DueDate, &loanSubmit.LoanStatus, &loanSubmit.CreatedAt, &loanSubmit.UpdatedAt); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		loan_Submits = append(loan_Submits, loanSubmit)
	}
	if err := rows.Err(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Return JSON response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(loan_Submits)
}

func GetLoanSubmitByID(w http.ResponseWriter, r *http.Request) {
	db, err := connectLoanSubmitDB()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer db.Close()

	// Extract loanSubmit_id from request parameters
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid Loan Submit ID", http.StatusBadRequest)
		return
	}

	// Query database for loan_submits with given loanSubmit_id
	var loanSubmit LoanSubmit
	query := `SELECT loanSubmit_id, applicant_id, loan_amount, interest_rate, loan_date, due_date, loan_status, created_at, updated_at FROM loan_submits WHERE loanSubmit_id = $1`
	err = db.QueryRow(query, id).Scan(&loanSubmit.LoanSubmitID, &loanSubmit.ApplicantID, &loanSubmit.LoanAmount, &loanSubmit.InterestRate,
		&loanSubmit.LoanDate, &loanSubmit.DueDate, &loanSubmit.LoanStatus, &loanSubmit.CreatedAt, &loanSubmit.UpdatedAt)
	if err == sql.ErrNoRows {
		// Return JSON error response if no loan submit with the given ID exists
		errorResponse := map[string]string{"error": "loan_submits data not found"}
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
	json.NewEncoder(w).Encode(loanSubmit)
}

func CreateLoanSubmit(w http.ResponseWriter, r *http.Request) {
	db, err := connectLoanSubmitDB()
	if err != nil {
		http.Error(w, fmt.Sprintf("Error connecting to the database: %v", err), http.StatusInternalServerError)
		return
	}
	defer db.Close()

	// Parse JSON request body
	var loanSubmit LoanSubmit
	err = json.NewDecoder(r.Body).Decode(&loanSubmit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Insert loan submission into the database
	query := `INSERT INTO loan_submits (applicant_id, loan_amount, interest_rate, loan_date, due_date, loan_status)
            VALUES ($1, $2, $3, $4, $5, $6) RETURNING loanSubmit_id`

	var loanSubmitID int
	// Format time.Time to PostgreSQL DATE format
	loanDate := loanSubmit.LoanDate.Format("2006-01-02")
	dueDate := loanSubmit.DueDate.Format("2006-01-02")

	err = db.QueryRow(query, loanSubmit.ApplicantID, loanSubmit.LoanAmount, loanSubmit.InterestRate, loanDate, dueDate, loanSubmit.LoanStatus).Scan(&loanSubmitID)
	if err != nil {
		if loanSubmit.LoanStatus != "ongoing" && loanSubmit.LoanStatus != "completed" {
			// If loan status is invalid, return a specific JSON response
			errorMessage := map[string]string{
				"error": fmt.Sprintf("Invalid loan status: '%s'. Allowed values are 'ongoing' or 'completed'.", loanSubmit.LoanStatus),
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest) // HTTP 400 Bad Request
			json.NewEncoder(w).Encode(errorMessage)
			return
		}

		// For other errors, return a generic internal server error
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Prepare success message
	successMessage := map[string]interface{}{
		"message":       "Loan submission information has been successfully created.",
		"loanSubmit_id": loanSubmitID,
	}

	// Set Content-Type and return JSON response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated) // HTTP 201 Created
	json.NewEncoder(w).Encode(successMessage)
}

func UpdateLoanSubmit(w http.ResponseWriter, r *http.Request) {
	db, err := connectLoanSubmitDB()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer db.Close()

	// Extract loanSubmit_id from request parameters
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid Loan Submit ID", http.StatusBadRequest)
		return
	}

	// Parse JSON request body
	var updateloanSubmit LoanSubmit
	err = json.NewDecoder(r.Body).Decode(&updateloanSubmit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Validate LoanStatus
	if updateloanSubmit.LoanStatus != "ongoing" && updateloanSubmit.LoanStatus != "completed" {
		// Return JSON error response if applicant_status is invalid
		errorResponse := map[string]string{"error": "Invalid Loan Submit status. Allowed values are 'ongoing' or 'completed'"}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(errorResponse)
		return
	}

	// Update query
	query := `UPDATE loan_submits 
			  SET applicant_id = $1, loan_amount = $2, interest_rate = $3, loan_date = $4, due_date = $5, loan_status = $6, updated_at = CURRENT_TIMESTAMP
              WHERE loanSubmit_id = $7`

	// Format time.Time to PostgreSQL DATE format
	loanDate := updateloanSubmit.LoanDate.Format("2006-01-02")
	dueDate := updateloanSubmit.DueDate.Format("2006-01-02")

	result, err := db.Exec(query, updateloanSubmit.ApplicantID, updateloanSubmit.LoanAmount, updateloanSubmit.InterestRate, loanDate, dueDate, updateloanSubmit.LoanStatus, updateloanSubmit.LoanSubmitID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Check if any rows were affected
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		// Return JSON error response if no Loan Submit with the given ID was found to update
		errorResponse := map[string]string{"error": "Loan Submit ID not found or no update performed"}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(errorResponse)
		return
	}

	// Return success message
	w.WriteHeader(http.StatusOK)
	successMessage := map[string]string{"message": fmt.Sprintf("Loan submission with ID %d updated successfully", id)}
	json.NewEncoder(w).Encode(successMessage)
}

func DeleteLoanSubmit(w http.ResponseWriter, r *http.Request) {
	db, err := connectLoanSubmitDB()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer db.Close()

	// Extract loanSubmit_id from request parameters
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid Loan Submit ID", http.StatusBadRequest)
		return
	}

	// Delete query
	query := `DELETE FROM loan_submits WHERE loanSubmit_id = $1`
	result, err := db.Exec(query, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Check if any rows were affected
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		// Return JSON error response if no Loan Submit ID with the given ID was found to delete
		errorResponse := map[string]string{"error": "Loan Submit ID not found or no delete performed"}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(errorResponse)
		return
	}

	// Return success message
	w.WriteHeader(http.StatusOK)
	successMessage := map[string]string{"message": fmt.Sprintf("Loan submission with ID %d deleted successfully", id)}
	json.NewEncoder(w).Encode(successMessage)
}
