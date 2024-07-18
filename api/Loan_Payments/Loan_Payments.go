package Loan_Payments

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

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
	loanpayments, err := readreceiptFromFile("api/Loan_Payments/json/receipts.json")
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
