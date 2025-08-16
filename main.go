// main.go

package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	// PostgreSQL driver

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"

	"github.com/descope/go-sdk/descope/client"
	_ "github.com/lib/pq"
)

// Utilizing the context package allows for the transmission of context capabilities like cancellation
// signals during the function call. In cases where context is absent, the context.Background()
// function serves as a viable alternative.
// Utilizing context within the Descope GO SDK is supported within versions 1.6.0 and higher.

// Student represents a student record in the database.
type Student struct {
	ID        int    `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Email     string `json:"email"`
	Major     string `json:"major"`
	TeacherID string `json:"teacher_id"`
}

var db *sql.DB
var descopeClient *client.DescopeClient

// Define a custom key type to avoid collisions
type contextKey string

const contextKeyUserID contextKey = "userID"
const contextKeyTeacherID contextKey = "teacherID" // A key for the teacher ID

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

// CHQ: Gemini AI generated this function
// faviconHandler serves the favicon.ico file.
func faviconHandler(w http.ResponseWriter, r *http.Request) {
    // Open the favicon file
    favicon, err := os.ReadFile("./static/calculator.ico")
    if err != nil {
        http.NotFound(w, r)
        return
    }

    // Set the Content-Type header
    w.Header().Set("Content-Type", "image/x-icon")
    
    // Write the file content to the response
    w.Write(favicon)
}

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

	projectID := os.Getenv("DESCOPE_PROJECT_ID")
	if projectID == "" {
		log.Fatal("DESCOPE_PROJECT_ID environment variable not set.")
	}
	descopeClient, err = client.NewWithConfig(&client.Config{ProjectID: projectID})
	if err != nil {
		log.Fatalf("failed to initialize Descope client: %v", err)
	}

	// Initialize the router
	router := mux.NewRouter()

	// All routes now go through the mux router, including static files
	router.HandleFunc("/", helloHandler)
	router.HandleFunc("/favicon.ico", faviconHandler)

	// Protected routes (require session validation)

    protectedRoutes := router.PathPrefix("/api").Subrouter()
    protectedRoutes.Use(sessionValidationMiddleware) // Apply middleware to all routes in this subrouter
    protectedRoutes.HandleFunc("/godbstudents", createStudent).Methods("POST")
    protectedRoutes.HandleFunc("/godbstudents/{id}", getStudent).Methods("GET")
    protectedRoutes.HandleFunc("/godbstudents", getAllgodbstudents).Methods("GET")
    protectedRoutes.HandleFunc("/godbstudents/{id}", updateStudent).Methods("PUT")
    protectedRoutes.HandleFunc("/godbstudents/{id}", updateStudentAlt).Methods("PATCH")
	protectedRoutes.HandleFunc("/godbstudents/{id}", deleteStudent).Methods("DELETE")

	// router.HandleFunc("/restfox/godbstudents", createStudent).Methods("POST")
    // router.HandleFunc("/restfox/godbstudents/{id}", getStudent).Methods("GET")
    // router.HandleFunc("/restfox/godbstudents", getAllgodbstudents).Methods("GET")
    // router.HandleFunc("/restfox/godbstudents/{id}", getAllgodbstudents).Methods("PUT")
    // router.HandleFunc("/restfox/godbstudents/{id}", getAllgodbstudents).Methods("DELETE")

	// // Define API routes
	// router.HandleFunc("/godbstudents", createStudent).Methods("POST")
	// router.HandleFunc("/godbstudents/{id}", getStudent).Methods("GET")
	// router.HandleFunc("/godbstudents", getAllgodbstudents).Methods("GET")
	// router.HandleFunc("/godbstudents/{id}", updateStudent).Methods("PUT")
	// router.HandleFunc("/godbstudents/{id}", deleteStudent).Methods("DELETE")

	// --- CORS Setup ---
	allowedOrigins := handlers.AllowedOrigins([]string{"*"})
	allowedMethods := handlers.AllowedMethods([]string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"})
	allowedHeaders := handlers.AllowedHeaders([]string{"Content-Type", "Authorization"})
	corsRouter := handlers.CORS(allowedOrigins, allowedMethods, allowedHeaders)(router)
	// --- End of CORS Setup ---
	
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" // Default port
	}
	
	fmt.Printf("Server listening on port %s...\n", port)
	log.Fatal(http.ListenAndServe(":"+port, corsRouter))
}

// CHQ: Gemini AI generated function
// helloHandler is the function that will be executed for requests to the "/" route.
func helloHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprint(w, "This is the server for the student records app. It's written in Go (aka GoLang).")
}

// CHQ: Gemini AI created function
// sessionValidationMiddleware is a middleware to validate the Descope session token.
func sessionValidationMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sessionToken := r.Header.Get("Authorization")
		if sessionToken == "" {
			http.Error(w, "Unauthorized: No session token provided", http.StatusUnauthorized)
			return
		}

		sessionToken = strings.TrimPrefix(sessionToken, "Bearer ")

		ctx := r.Context()
		authorized, token, err := descopeClient.Auth.ValidateSessionWithToken(ctx, sessionToken)
		if err != nil || !authorized {
			log.Printf("Session validation failed: %v", err)
			http.Error(w, "Unauthorized: Invalid session token", http.StatusUnauthorized)
			return
		}

		userID := token.ID
		if userID == "" {
			http.Error(w, "Unauthorized: User ID not found in token", http.StatusUnauthorized)
			return
		}
		
		// For this example, we assume the teacher ID is the same as the user ID.
		// In a real-world app, you would extract this from custom claims in the token.
		teacherID := userID

		// Store the user ID and teacher ID in the request's context
		ctxWithUserID := context.WithValue(ctx, contextKeyUserID, userID)
		ctxWithIDs := context.WithValue(ctxWithUserID, contextKeyTeacherID, teacherID)
		
		next.ServeHTTP(w, r.WithContext(ctxWithIDs))
	})
}

// createStudent handles POST requests to create a new student record.
func createStudent(w http.ResponseWriter, r *http.Request) {
	teacherID, ok := r.Context().Value(contextKeyTeacherID).(string)
	if !ok || teacherID == "" {
		http.Error(w, "Forbidden: Teacher ID not found in session", http.StatusForbidden)
		return
	}

	var student Student
	err := json.NewDecoder(r.Body).Decode(&student)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	// Enforce that the student being created is associated with the authenticated teacher.
	student.TeacherID = teacherID

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

// getStudent handles GET requests to retrieve a single student by ID, but also checks for ownership.
func getStudent(w http.ResponseWriter, r *http.Request) {
	teacherID, ok := r.Context().Value(contextKeyTeacherID).(string)
	if !ok || teacherID == "" {
		http.Error(w, "Forbidden: Teacher ID not found in session", http.StatusForbidden)
		return
	}
	
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid student ID", http.StatusBadRequest)
		return
	}

	var student Student
	// Ensure the student belongs to the authenticated teacher.
	query := `SELECT id, first_name, last_name, email, major, teacher_id FROM godbstudents WHERE id = $1 AND teacher_id = $2`
	row := db.QueryRow(query, id, teacherID)

	err = row.Scan(&student.ID, &student.FirstName, &student.LastName, &student.Email, &student.Major, &student.TeacherID)
	if err == sql.ErrNoRows {
		http.Error(w, "Student not found or not owned by this teacher", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, fmt.Sprintf("Error retrieving student: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(student)
}

// getAllgodbstudents handles GET requests to retrieve all student records for the authenticated teacher.
func getAllgodbstudents(w http.ResponseWriter, r *http.Request) {
	teacherID, ok := r.Context().Value(contextKeyTeacherID).(string)
	if !ok || teacherID == "" {
		http.Error(w, "Forbidden: Teacher ID not found in session", http.StatusForbidden)
		return
	}

	var students []Student
	query := `SELECT id, first_name, last_name, email, major, teacher_id FROM godbstudents WHERE teacher_id = $1 ORDER BY id`
	rows, err := db.Query(query, teacherID)

	if err != nil {
		http.Error(w, fmt.Sprintf("Error retrieving students: %v", err), http.StatusInternalServerError)
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
		students = append(students, student)
	}

	if err = rows.Err(); err != nil {
		http.Error(w, fmt.Sprintf("Error iterating over student rows: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(students)
}
// updateStudent handles PUT requests to update an existing student record, with an ownership check.
func updateStudent(w http.ResponseWriter, r *http.Request) {
	teacherID, ok := r.Context().Value(contextKeyTeacherID).(string)
	if !ok || teacherID == "" {
		http.Error(w, "Forbidden: Teacher ID not found in session", http.StatusForbidden)
		return
	}

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

	// The teacherID from the request body is ignored and replaced with the authenticated teacher's ID
	student.TeacherID = teacherID
	
	query := `UPDATE godbstudents SET first_name = $1, last_name = $2, email = $3, major = $4 WHERE id = $5 AND teacher_id = $6`
	result, err := db.Exec(query, student.FirstName, student.LastName, student.Email, student.Major, id, student.TeacherID)
	
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
		http.Error(w, "Student not found or not owned by this teacher", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Student updated successfully"})
}


// updateStudentAlt handles PATCH requests to update an existing student record, with an ownership check.
func updateStudentAlt(w http.ResponseWriter, r *http.Request) {
	teacherID, ok := r.Context().Value(contextKeyTeacherID).(string)
	if !ok || teacherID == "" {
		http.Error(w, "Forbidden: Teacher ID not found in session", http.StatusForbidden)
		return
	}

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

	// The teacherID from the request body is ignored and replaced with the authenticated teacher's ID
	student.TeacherID = teacherID
	
	query := `UPDATE godbstudents SET first_name = $1, last_name = $2, email = $3, major = $4 WHERE id = $5 AND teacher_id = $6`
	result, err := db.Exec(query, student.FirstName, student.LastName, student.Email, student.Major, id, student.TeacherID)
	
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
		http.Error(w, "Student not found or not owned by this teacher", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Student updated successfully"})
}

// deleteStudent handles DELETE requests to delete a student record by ID, with an ownership check.
func deleteStudent(w http.ResponseWriter, r *http.Request) {
	teacherID, ok := r.Context().Value(contextKeyTeacherID).(string)
	if !ok || teacherID == "" {
		http.Error(w, "Forbidden: Teacher ID not found in session", http.StatusForbidden)
		return
	}
	
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid student ID", http.StatusBadRequest)
		return
	}
	
	// Ensure the student belongs to the authenticated teacher.
	query := `DELETE FROM godbstudents WHERE id = $1 AND teacher_id = $2`
	result, err := db.Exec(query, id, teacherID)
	
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
		http.Error(w, "Student not found or not owned by this teacher", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Student deleted successfully"})
}
