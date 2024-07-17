package Loan_Submits

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
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
	LoanAmount   float64
	InterestRate float64
	LoanDate     CustomDate // Use CustomDate
	DueDate      CustomDate // Use CustomDate
	LoanStatus   string
	CreatedAt    string
	UpdatedAt    string
}

func connectLoanSubmitDB() (*sql.DB, error) {
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
	db, err := connectLoanSubmitDB()
	if err != nil {
		log.Fatal("Error connecting to the database:", err)
	}
	defer db.Close()

	log.Println("Connected to the database successfully")

	// Create the loan_submits table if it doesn't exist
	if err := createLoanSubmitTable(db); err != nil {
		log.Fatal("Error creating loan_submits table:", err)
	}

	// Read data from JSON file
	loanSubmits, err := readSubmitFromFile("api/Loan_Submits/json/SubmittedApp.json")
	if err != nil {
		log.Fatal(err)
	}

	// Insert each applicant into the database
	for _, loanApplicant := range loanSubmits {
		pk := InsertLoanSubmit(db, loanApplicant)
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

func TestReadSubmitFromFile() {
	filename := "api/Loan_Submits/json/SubmittedApp.json"
	loanSubmits, err := readSubmitFromFile(filename)
	if err != nil {
		log.Fatalf("Error reading file %s: %v", filename, err)
	}

	fmt.Printf("Successfully read %d loan submissions:\n", len(loanSubmits))
	for _, submit := range loanSubmits {
		fmt.Printf("ApplicantID: %d, LoanAmount: %.2f, InterestRate: %.2f, LoanDate: %s, DueDate: %s, LoanStatus: %s\n",
			submit.ApplicantID, submit.LoanAmount, submit.InterestRate, submit.LoanDate.Format("2006-01-02"), submit.DueDate.Format("2006-01-02"), submit.LoanStatus)
	}
}

func GetLoanSubmitByID(w http.ResponseWriter, r *http.Request) {
	// Extract loanSubmit_id from request parameters
	params := mux.Vars(r)
	loanSubmitID := params["id"]

	db, err := connectLoanSubmitDB()
	if err != nil {
		http.Error(w, fmt.Sprintf("Error connecting to the database: %v", err), http.StatusInternalServerError)
		return
	}
	defer db.Close()

	// Query the loan submission by ID
	row := db.QueryRow("SELECT loanSubmit_id, applicant_id, loan_amount, interest_rate, loan_date, due_date, loan_status, created_at, updated_at FROM loan_submits WHERE loanSubmit_id = $1", loanSubmitID)

	var loanSubmit LoanSubmit
	err = row.Scan(&loanSubmit.LoanSubmitID, &loanSubmit.ApplicantID, &loanSubmit.LoanAmount, &loanSubmit.InterestRate,
		&loanSubmit.LoanDate, &loanSubmit.DueDate, &loanSubmit.LoanStatus, &loanSubmit.CreatedAt, &loanSubmit.UpdatedAt)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error retrieving loan submission: %v", err), http.StatusInternalServerError)
		return
	}

	// Return JSON response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(loanSubmit)
}

func AddLoanSubmit(w http.ResponseWriter, r *http.Request) {
	// Parse JSON request body
	var loanSubmit LoanSubmit
	err := json.NewDecoder(r.Body).Decode(&loanSubmit)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error decoding JSON: %v", err), http.StatusBadRequest)
		return
	}

	db, err := connectLoanSubmitDB()
	if err != nil {
		http.Error(w, fmt.Sprintf("Error connecting to the database: %v", err), http.StatusInternalServerError)
		return
	}
	defer db.Close()

	// Insert loan submission into the database
	query := `INSERT INTO loan_submits (applicant_id, loan_amount, interest_rate, loan_date, due_date, loan_status)
            VALUES ($1, $2, $3, $4, $5, $6) RETURNING loanSubmit_id`

	var loanSubmitID int
	// Format time.Time to PostgreSQL DATE format
	loanDate := loanSubmit.LoanDate.Format("2006-01-02")
	dueDate := loanSubmit.DueDate.Format("2006-01-02")

	err = db.QueryRow(query, loanSubmit.ApplicantID, loanSubmit.LoanAmount, loanSubmit.InterestRate, loanDate, dueDate, loanSubmit.LoanStatus).Scan(&loanSubmitID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error inserting loan submission: %v", err), http.StatusInternalServerError)
		return
	}

	// Return success response with inserted loanSubmitID
	response := map[string]int{"loanSubmitID": loanSubmitID}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
