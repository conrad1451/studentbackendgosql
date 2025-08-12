package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	// PostgreSQL driver
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"

	// Import the handlers package for CORS middleware
	"github.com/gorilla/handlers"
)

// Student represents a student record in the database.
type Student struct {
	ID        int    `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Email     string `json:"email"`
	Major     string `json:"major"`
}

var db *sql.DB

func databaseChosen(chosenDB string) string{
	var theVal string = ""

	switch chosenDB {
	case "NEON_STUDENT_RECORDS_DB":
		return "Neon DB student records DB chosen"
	case "PROJECT2_DB":
		return "Neon DB project2 DB chosen"
	case "GOOGLE_CLOUD_SQL":
		return "Google Cloud SQL DB chosen"
	case "GOOGLE_VM_HOSTED_SQL":
		return "Google VM hosted DB chosen"		
	default:
		return "some DB chosen" 
	}
}

var listOfDBConnections = []string{"NEON_STUDENT_RECORDS_DB", "PROJECT2_DB", "GOOGLE_CLOUD_SQL", "GOOGLE_VM_HOSTED_SQL"}

func main() {
	// Initialize database connection
	var err error
	theChosenDB := listOfDBConnections[3]
	dbConnStr := os.Getenv(theChosenDB)
	if dbConnStr == "" {
		log.Fatal("DATABASE_URL environment variable not set.")
	}

	db, err = sql.Open("postgres", dbConnStr)
	if err != nil {
		log.Fatalf("Error opening database: %v", err)
	}
	defer db.Close()

	err = db.Ping()
	if err != nil {
		log.Fatalf("Error connecting to the database: %v", err)
	}
	fmt.Println("Successfully connected to the database!")
	fmt.Println( databaseChosen(theChosenDB) )

	// Initialize the router
	router := mux.NewRouter()

	// Define API routes
	router.HandleFunc("/godbstudents", createStudent).Methods("POST")
	router.HandleFunc("/godbstudents/{id}", getStudent).Methods("GET")
	router.HandleFunc("/godbstudents", getAllgodbstudents).Methods("GET")
	router.HandleFunc("/godbstudents/{id}", updateStudent).Methods("PUT")
	router.HandleFunc("/godbstudents/{id}", deleteStudent).Methods("DELETE")

	// --- CORS Setup ---
	// Create a list of allowed origins (e.g., your front-end URL)
	// For production, you should replace "*" with your specific front-end domain.
	allowedOrigins := handlers.AllowedOrigins([]string{"*"})
	
	// Create a list of allowed methods (GET, POST, etc.)
	allowedMethods := handlers.AllowedMethods([]string{"GET", "POST", "PUT", "DELETE", "OPTIONS"})

	// Create a list of allowed headers, including Content-Type
	allowedHeaders := handlers.AllowedHeaders([]string{"Content-Type", "Authorization"})

	// Wrap your router with the CORS handler
	corsRouter := handlers.CORS(allowedOrigins, allowedMethods, allowedHeaders)(router)
	// --- End of CORS Setup ---

	// Start the HTTP server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" // Default port
	}
	fmt.Printf("Server listening on port %s...\n", port)
	
	// Pass the corsRouter to ListenAndServe
	log.Fatal(http.ListenAndServe(":"+port, corsRouter))
}

// createStudent handles POST requests to create a new student record.
func createStudent(w http.ResponseWriter, r *http.Request) {
	var student Student
	err := json.NewDecoder(r.Body).Decode(&student)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	query := `INSERT INTO godbstudents (first_name, last_name, email, major) VALUES ($1, $2, $3, $4) RETURNING id`
	err = db.QueryRow(query, student.FirstName, student.LastName, student.Email, student.Major).Scan(&student.ID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error creating student: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(student)
}

// getStudent handles GET requests to retrieve a single student by ID.
func getStudent(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid student ID", http.StatusBadRequest)
		return
	}

	var student Student
	query := `SELECT id, first_name, last_name, email, major FROM godbstudents WHERE id = $1`
	row := db.QueryRow(query, id)

	err = row.Scan(&student.ID, &student.FirstName, &student.LastName, &student.Email, &student.Major)
	if err == sql.ErrNoRows {
		http.Error(w, "Student not found", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, fmt.Sprintf("Error retrieving student: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(student)
}

// getAllgodbstudents handles GET requests to retrieve all student records.
func getAllgodbstudents(w http.ResponseWriter, r *http.Request) {
	var godbstudents []Student
	query := `SELECT id, first_name, last_name, email, major FROM godbstudents ORDER BY id`
	rows, err := db.Query(query)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error retrieving godbstudents: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var student Student
		err := rows.Scan(&student.ID, &student.FirstName, &student.LastName, &student.Email, &student.Major)
		if err != nil {
			log.Printf("Error scanning student row: %v", err)
			continue
		}
		godbstudents = append(godbstudents, student)
	}

	if err = rows.Err(); err != nil {
		http.Error(w, fmt.Sprintf("Error iterating over student rows: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(godbstudents)
}

// updateStudent handles PUT requests to update an existing student record.
func updateStudent(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid student ID", http.StatusBadRequest)
		return
	}

	var student Student
	err = json.NewDecoder(r.Body).Decode(&student)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if student.ID != 0 && student.ID != id {
		http.Error(w, "ID in URL and request body do not match", http.StatusBadRequest)
		return
	}
	student.ID = id

	query := `UPDATE godbstudents SET first_name = $1, last_name = $2, email = $3, major = $4 WHERE id = $5`
	result, err := db.Exec(query, student.FirstName, student.LastName, student.Email, student.Major, student.ID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error updating student: %v", err), http.StatusInternalServerError)
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		http.Error(w, fmt.Sprintf("Error checking rows affected: %v", err), http.StatusInternalServerError)
		return
	}
	if rowsAffected == 0 {
		http.Error(w, "Student not found or no changes made", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Student updated successfully"})
}

// deleteStudent handles DELETE requests to delete a student record by ID.
func deleteStudent(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid student ID", http.StatusBadRequest)
		return
	}

	query := `DELETE FROM godbstudents WHERE id = $1`
	result, err := db.Exec(query, id)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error deleting student: %v", err), http.StatusInternalServerError)
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		http.Error(w, fmt.Sprintf("Error checking rows affected: %v", err), http.StatusInternalServerError)
		return
	}
	if rowsAffected == 0 {
		http.Error(w, "Student not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Student deleted successfully"})
}