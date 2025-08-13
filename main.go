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
	TeacherID int    `json:"teacher_id"`
}

var db *sql.DB

// [1]
func databaseChosen(chosenDB string) string {
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
	fmt.Println(databaseChosen(theChosenDB))

	// Initialize the router
	router := mux.NewRouter()

	// Define API routes
	router.HandleFunc("/godbstudents", createStudent).Methods("POST")
	router.HandleFunc("/godbstudents/{id}", getStudent).Methods("GET")
	router.HandleFunc("/godbstudents", getAllgodbstudents).Methods("GET")
	router.HandleFunc("/godbstudents/{id}", updateStudent).Methods("PUT")
	router.HandleFunc("/godbstudents/{id}", deleteStudent).Methods("DELETE")

	// --- CORS Setup ---
	allowedOrigins := handlers.AllowedOrigins([]string{"*"})
	allowedMethods := handlers.AllowedMethods([]string{"GET", "POST", "PUT", "DELETE", "OPTIONS"})
	allowedHeaders := handlers.AllowedHeaders([]string{"Content-Type", "Authorization"})
	corsRouter := handlers.CORS(allowedOrigins, allowedMethods, allowedHeaders)(router)
	// --- End of CORS Setup ---

	// Start the HTTP server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" // Default port
	}
	fmt.Printf("Server listening on port %s...\n", port)
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

	query := `INSERT INTO godbstudents (first_name, last_name, email, major, teacher_id) VALUES ($1, $2, $3, $4, $5) RETURNING id`
	err = db.QueryRow(query, student.FirstName, student.LastName, student.Email, student.Major, student.TeacherID).Scan(&student.ID)
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
	query := `SELECT id, first_name, last_name, email, major, teacher_id FROM godbstudents WHERE id = $1`
	row := db.QueryRow(query, id)

	err = row.Scan(&student.ID, &student.FirstName, &student.LastName, &student.Email, &student.Major, &student.TeacherID)
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
// It now supports an optional `teacherID` query parameter to filter results.
func getAllgodbstudents(w http.ResponseWriter, r *http.Request) {
	var godbstudents []Student
	
	// Get query parameters from the request
	queryParams := r.URL.Query()
	teacherIDStr := queryParams.Get("teacherID")

	var rows *sql.Rows
	var err error

	// If a teacherID is provided, filter the results
	if teacherIDStr != "" {
		teacherID, err := strconv.Atoi(teacherIDStr)
		if err != nil {
			http.Error(w, "Invalid teacherID query parameter", http.StatusBadRequest)
			return
		}
		query := `SELECT id, first_name, last_name, email, major, teacher_id FROM godbstudents WHERE teacher_id = $1 ORDER BY id`
		rows, err = db.Query(query, teacherID)
	} else {
		// Otherwise, retrieve all students
		query := `SELECT id, first_name, last_name, email, major, teacher_id FROM godbstudents ORDER BY id`
		rows, err = db.Query(query)
	}

	if err != nil {
		http.Error(w, fmt.Sprintf("Error retrieving godbstudents: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var student Student
		err := rows.Scan(&student.ID, &student.FirstName, &student.LastName, &student.Email, &student.Major, &student.TeacherID)
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

	query := `UPDATE godbstudents SET first_name = $1, last_name = $2, email = $3, major = $4, teacher_id = $5 WHERE id = $6`
	result, err := db.Exec(query, student.FirstName, student.LastName, student.Email, student.Major, student.TeacherID, student.ID)
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
