package Loan_Payments

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

// ----------------------------- end for solve problem cannot insert data from file json into database -----------------------------

type LoanPayment struct {
	LoanPaymentID int
	LoanSubmitID  int
	PaymentAmount decimal.Decimal
	PaymentDate   CustomDate
	PaymentMethod string
	PaymentStatus string
	CreatedAt     string
	UpdatedAt     string
}

func connectLoanPaymentsDB() (*sql.DB, error) {
	connStr := "postgres://Admin:Password@localhost:5432/LMS_LoanPaymentsDB?sslmode=disable"
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
	db, err := connectLoanPaymentsDB()
	if err != nil {
		log.Fatal("Error connecting to the database:", err)
	}
	defer db.Close()

	// Create the loan_submits table if it doesn't exist
	if err := createLoanPaymentTable(db); err != nil {
		log.Fatal("Error creating loan_submits table:", err)
	}

	// Read data from JSON file
	loanpayments, err := readreceiptFromFile("json/receipts.json")
	if err != nil {
		log.Fatal(err)
	}

	// Insert each applicant into the database
	for _, payments := range loanpayments {
		pk := InsertLoanSubmit(db, payments)
		fmt.Printf("Inserted loan payment ID = %d\n", pk)
	}
}

func createLoanPaymentTable(db *sql.DB) error {
	query := `CREATE TABLE IF NOT EXISTS loan_payments (
		loanPayment_id SERIAL PRIMARY KEY,
		loanSubmit_id INT NOT NULL,
		payment_amount DECIMAL(15, 2) NOT NULL,
		payment_date DATE NOT NULL,
		payment_method VARCHAR(50),
		payment_status VARCHAR(15) NOT NULL CHECK (payment_status IN ('not-complete', 'completed')),
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`

	_, err := db.Exec(query)
	if err != nil {
		return fmt.Errorf("error creating loan_payments table: %v", err)
	}
	return nil
}

func InsertLoanSubmit(db *sql.DB, payments LoanPayment) int {
	query := `INSERT INTO loan_payments (loanSubmit_id, payment_amount, payment_date, payment_method, payment_status)
            VALUES ($1, $2, $3, $4, $5) RETURNING loanPayment_id`

	var PaymentsID int
	// Format time.Time to PostgreSQL DATE format
	PaymentDate := payments.PaymentDate.Format("2006-01-02")

	err := db.QueryRow(query, payments.LoanSubmitID, payments.PaymentAmount, PaymentDate, payments.PaymentMethod, payments.PaymentStatus).Scan(&PaymentsID)
	if err != nil {
		log.Fatal(err)
	}
	return PaymentsID
}

func readreceiptFromFile(filename string) ([]LoanPayment, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("error opening file: %v", err)
	}
	defer file.Close()

	var LoanPayments []LoanPayment
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&LoanPayments); err != nil {
		return nil, fmt.Errorf("error decoding JSON: %v", err)
	}

	return LoanPayments, nil
}

func GetLoanPayment(w http.ResponseWriter, r *http.Request) {
	db, err := connectLoanPaymentsDB()
	if err != nil {
		log.Fatal("Error connecting to the database:", err)
	}
	defer db.Close()

	// Query from the loan_payments table
	rows, err := db.Query("SELECT loanPayment_id, loanSubmit_id, payment_amount, payment_date, payment_method, payment_status, created_at, updated_at FROM loan_payments")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var loanPayments []LoanPayment
	for rows.Next() {
		var payment LoanPayment
		if err := rows.Scan(&payment.LoanPaymentID, &payment.LoanSubmitID, &payment.PaymentAmount, &payment.PaymentDate, &payment.PaymentMethod, &payment.PaymentStatus, &payment.CreatedAt, &payment.UpdatedAt); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		loanPayments = append(loanPayments, payment)
	}
	if err := rows.Err(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Return JSON response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(loanPayments)
}

func GetLoanPaymentByID(w http.ResponseWriter, r *http.Request) {
	db, err := connectLoanPaymentsDB()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer db.Close()

	// Extract loanPayment_id from request parameters
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid Loan Payment ID", http.StatusBadRequest)
		return
	}

	// Query database for loan_payments with given loanPayment_id
	var loanPayment LoanPayment
	query := `SELECT loanPayment_id, loanSubmit_id, payment_amount, payment_date, payment_method, payment_status, created_at, updated_at FROM loan_payments WHERE loanPayment_id = $1`
	err = db.QueryRow(query, id).Scan(&loanPayment.LoanPaymentID, &loanPayment.LoanSubmitID, &loanPayment.PaymentAmount, &loanPayment.PaymentDate, &loanPayment.PaymentMethod, &loanPayment.PaymentStatus, &loanPayment.CreatedAt, &loanPayment.UpdatedAt)
	if err == sql.ErrNoRows {
		// Return JSON error response if no loan payment with the given ID exists
		errorResponse := map[string]string{"error": "loan_payments data not found"}
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
	json.NewEncoder(w).Encode(loanPayment)
}

func CreateLoanPayment(w http.ResponseWriter, r *http.Request) {
	db, err := connectLoanPaymentsDB()
	if err != nil {
		http.Error(w, fmt.Sprintf("Error connecting to the database: %v", err), http.StatusInternalServerError)
		return
	}
	defer db.Close()

	// Parse JSON request body
	var loanPayment LoanPayment
	err = json.NewDecoder(r.Body).Decode(&loanPayment)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Insert loan payments into the database
	query := `INSERT INTO loan_payments (loanSubmit_id, payment_amount, payment_date, payment_method, payment_status)
        VALUES ($1, $2, $3, $4, $5) RETURNING loanPayment_id`

	var loanPaymentID int
	// Format time.Time to PostgreSQL DATE format
	paymentDate := loanPayment.PaymentDate.Format("2006-01-02")

	err = db.QueryRow(query, loanPayment.LoanSubmitID, loanPayment.PaymentAmount, paymentDate, loanPayment.PaymentMethod, loanPayment.PaymentStatus).Scan(&loanPaymentID)
	if err != nil {
		if loanPayment.PaymentStatus != "not-complete" && loanPayment.PaymentStatus != "completed" {
			// If payment status is invalid, return a specific JSON response
			errorMessage := map[string]string{
				"error": fmt.Sprintf("Invalid payment status: '%s'. Allowed values are 'not-complete' or 'completed'.", loanPayment.PaymentStatus),
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

	// Prepare success message with only the relevant ID
	successMessage := map[string]interface{}{
		"message":        "Loan payment information has been successfully created.",
		"loanPayment_id": loanPaymentID, // Use the ID of the newly created payment
	}

	// Set Content-Type and return JSON response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated) // HTTP 201 Created
	json.NewEncoder(w).Encode(successMessage)
}

func UpdateLoanPayment(w http.ResponseWriter, r *http.Request) {
	db, err := connectLoanPaymentsDB()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer db.Close()

	// Extract loanPayment_id from request parameters
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid Loan Payment ID", http.StatusBadRequest)
		return
	}

	// Parse JSON request body
	var updateLoanPayment LoanPayment
	err = json.NewDecoder(r.Body).Decode(&updateLoanPayment)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Validate PaymentStatus
	if updateLoanPayment.PaymentStatus != "not-complete" && updateLoanPayment.PaymentStatus != "completed" {
		// Return JSON error response if payment status is invalid
		errorResponse := map[string]string{"error": "Invalid Payment Status. Allowed values are 'not-complete' or 'completed'"}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(errorResponse)
		return
	}

	// Update query
	query := `UPDATE loan_payments 
			  SET loanSubmit_id = $1, payment_amount = $2, payment_date = $3, payment_method = $4, payment_status = $5, updated_at = CURRENT_TIMESTAMP
              WHERE loanPayment_id = $6`

	// Format time.Time to PostgreSQL DATE format
	paymentDate := updateLoanPayment.PaymentDate.Format("2006-01-02")

	result, err := db.Exec(query, updateLoanPayment.LoanSubmitID, updateLoanPayment.PaymentAmount, paymentDate, updateLoanPayment.PaymentMethod, updateLoanPayment.PaymentStatus, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Check if any rows were affected
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		// Return JSON error response if no Loan Payment with the given ID was found to update
		errorResponse := map[string]string{"error": "Loan Payment ID not found or no update performed"}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(errorResponse)
		return
	}

	// Return success message
	w.WriteHeader(http.StatusOK)
	successMessage := map[string]string{"message": fmt.Sprintf("Loan payment with ID %d updated successfully", id)}
	json.NewEncoder(w).Encode(successMessage)
}

func DeleteLoanPayment(w http.ResponseWriter, r *http.Request) {
	db, err := connectLoanPaymentsDB()
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
	query := `DELETE FROM loan_payments WHERE loanPayment_id = $1`
	result, err := db.Exec(query, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Check if any rows were affected
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if rowsAffected == 0 {
		// Return JSON error response if no Loan Payment with the given ID was found to delete
		errorResponse := map[string]string{"error": "Loan Payment ID not found or no delete performed"}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(errorResponse)
		return
	}

	// Return success message
	w.WriteHeader(http.StatusOK)
	successMessage := map[string]string{"message": fmt.Sprintf("Loan payment with ID %d deleted successfully", id)}
	json.NewEncoder(w).Encode(successMessage)
}
