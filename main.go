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

// Teacher represents a teacher record in the database.
type Teacher struct {
	TeacherID       string `json:"teacher_id"`
	FirstName       string `json:"first_name"`
	LastName        string `json:"last_name"`
	TeacherUsername string `json:"teacher_username"`
}

type TeacherAlt struct {
	TeacherID       string `json:"teacher_id"` 
}

var db *sql.DB
var descopeClient *client.DescopeClient

// var isAnAdmin bool
// Define a custom key type to avoid collisions
type contextKey string

// CHQ: Gemini AI changed global flag variable into context key
// var isAnAdmin bool
const contextKeyIsAdmin contextKey = "isAdmin"

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

// CHQ: Gemini AI created endpoint
// registerTeacher handles POST requests to register a new user from a Descope token
// into the 'real_teachers' table.
func registerTeacher(w http.ResponseWriter, r *http.Request) {
	// The Descope middleware should ensure the teacherID is in the context
	teacherID, ok := r.Context().Value(contextKeyTeacherID).(string)
	if !ok || teacherID == "" {
		http.Error(w, "Forbidden: TeacherAlt ID not found in session", http.StatusForbidden)
		return
	}

	// We expect the request body to contain the teacher's profile info
	var teacher TeacherAlt
	err := json.NewDecoder(r.Body).Decode(&teacher)
	if err != nil {
		// Log the error but continue, as we'll use token data if body is empty/malformed
		log.Printf("Warning: Failed to decode teacher profile from body: %v. Relying on token data.", err)
	}

	// For a more robust solution, we can fetch the token's claims here to ensure we have the data
	// Note: We'll use the r.Context().Value(contextKeyUserID) if the Descope token struct was stored there.
	// Since your middleware only stores the ID, we'll try to use the body data or placeholders.

	// In a real application, you would pass the full token object or the necessary claims
	// through the context, or expect the client to send a minimal payload with the required fields.
	// For this example, we assume the front end sends the teacher data (Name/Username).
	// if teacher.FirstName == "" || teacher.LastName == "" || teacher.TeacherUsername == "" {
	// 	// This is a simplified fallback/error handling. A better solution would rely on
	// 	// custom claims in the token or a more complete client-side payload.
	// 	http.Error(w, "Bad Request: Missing first_name, last_name, or teacher_username in body.", http.StatusBadRequest)
	// 	return
	// }

	// Set the ID from the authenticated token, overriding any ID in the request body
	teacher.TeacherID = teacherID

	// This PostgreSQL query uses ON CONFLICT DO NOTHING to ensure idempotent operations.
	// If the teacher_id already exists, the row is not inserted, avoiding a primary key error.
	// query := `
	// 	INSERT INTO the_real_teachers (teacher_id)
	// 	VALUES ($1)
	// 	ON CONFLICT (teacher_id) DO NOTHING
	// `

	query := `
	INSERT INTO the_real_teachers (teacher_id)
	VALUES ($1)	
	`
	// _, err = db.Exec(query, teacher.TeacherID, teacher.FirstName, teacher.LastName, teacher.TeacherUsername)
	_, err = db.Exec(query, teacher.TeacherID )

	if err != nil {
		// Log the error for the server but return a generic message to the client
		log.Printf("Error inserting teacher %s into DB: %v", teacher.TeacherID, err)
		http.Error(w, "Internal Server Error: Could not register teacher.", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "TeacherAlt registration processed. Existing users ignored."})
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

	projectID := os.Getenv("DESCOPE_PERSONALPROJECT1")
	// projectID := os.Getenv("DESCOPE_PROJECT_PERSONAL_MELLOW_NAVY")
	// projectID := os.Getenv("DESCOPE_PROJECT_ID")
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

	protectedRoutes.HandleFunc("/registerteacher", registerTeacher).Methods("POST")

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

	originList := []string{
		"https://studentfrontendreact.vercel.app", 
		"http://studentfrontendreact.vercel.app",
		"https://localhost:5173",
		"http://localhost:5173",
		"https://localhost:5174",
		"http://localhost:5174",
	}

	allowedOrigins := handlers.AllowedOrigins(originList)
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


		// CHQ: GEmini AI removed global flag check
		// if descopeClient.Auth.ValidateRoles(context.Background(), token, []string{"School Administrator"}) {
		// 	isAnAdmin = true
		// } else {
		// 	isAnAdmin = false
		// }
        isAdmin := descopeClient.Auth.ValidateRoles(context.Background(), token, []string{"School Administrator"})


		userID := token.ID
		// userRole := token.GetTenants()
		// userRole := token.GetTenantValue()
		// userRole := token.GetTenants()
		if userID == "" {
			http.Error(w, "Unauthorized: User ID not found in token", http.StatusUnauthorized)
			return
		}
		
		// For this example, we assume the teacher ID is the same as the user ID.
		// In a real-world app, you would extract this from custom claims in the token.
		teacherID := userID

				
        // Store the user ID, teacher ID, and admin status in the request's context
        ctxWithUserID := context.WithValue(ctx, contextKeyUserID, userID)
        ctxWithIDs := context.WithValue(ctxWithUserID, contextKeyTeacherID, teacherID)
        
		// CHQ: Gemini AI added admin status to context
        ctxWithAdminStatus := context.WithValue(ctxWithIDs, contextKeyIsAdmin, isAdmin)
        
        // Use the final context
        next.ServeHTTP(w, r.WithContext(ctxWithAdminStatus))

		// // Store the user ID and teacher ID in the request's context
		// ctxWithUserID := context.WithValue(ctx, contextKeyUserID, userID)
		// ctxWithIDs := context.WithValue(ctxWithUserID, contextKeyTeacherID, teacherID)
		
		// next.ServeHTTP(w, r.WithContext(ctxWithIDs))
	})
}

// createStudent handles POST requests to create a new student record.
func createStudentAsAdmin(w http.ResponseWriter, r *http.Request) {
 
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

// createStudent handles POST requests to create a new student record.
func createStudentAsTeacher(w http.ResponseWriter, r *http.Request) {
	teacherID, ok := r.Context().Value(contextKeyTeacherID).(string)
	if !ok || teacherID == "" {
		http.Error(w, "Forbidden: TeacherAlt ID not found in session", http.StatusForbidden)
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

func createStudent(w http.ResponseWriter, r *http.Request){

	// CHQ: Gemini AI changed fetching global vatiable to retrieving variable from context
	// Retrieve isAdmin from context
    isAdmin, ok := r.Context().Value(contextKeyIsAdmin).(bool)
    if !ok {
        // Fallback for safety, though middleware should ensure it's set
        http.Error(w, "Forbidden: Role not determined", http.StatusForbidden)
        return
    }

	if (isAdmin) {
		createStudentAsAdmin(w, r)
	} else {
		createStudentAsTeacher(w, r)
	}
}

func getStudentAsAdmin(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid student ID", http.StatusBadRequest)
		return
	}

	var student Student
	// Ensure the student belongs to the authenticated teacher.
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

// getStudent handles GET requests to retrieve a single student by ID, but also checks for ownership.
func getStudentAsTeacher(w http.ResponseWriter, r *http.Request) {
	teacherID, ok := r.Context().Value(contextKeyTeacherID).(string)
	if !ok || teacherID == "" {
		http.Error(w, "Forbidden: TeacherAlt ID not found in session", http.StatusForbidden)
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

func getStudent(w http.ResponseWriter, r *http.Request){
	
	// CHQ: Gemini AI changed fetching global vatiable to retrieving variable from context
	// Retrieve isAdmin from context
    isAdmin, ok := r.Context().Value(contextKeyIsAdmin).(bool)
    if !ok {
        // Fallback for safety, though middleware should ensure it's set
        http.Error(w, "Forbidden: Role not determined", http.StatusForbidden)
        return
    }

	if (isAdmin) {
		getStudentAsAdmin(w, r)
	} else {
		getStudentAsTeacher(w, r)
	}
}


func getAllgodbstudentsAsAdmin(w http.ResponseWriter) {
	var students []Student
	query := `SELECT id, first_name, last_name, email, major FROM godbstudents ORDER BY id`
	rows, err := db.Query(query)

	if err != nil {
		http.Error(w, fmt.Sprintf("Error retrieving students: %v", err), http.StatusInternalServerError)
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
		students = append(students, student)
	}

	if err = rows.Err(); err != nil {
		http.Error(w, fmt.Sprintf("Error iterating over student rows: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(students)
}

// getAllgodbstudents handles GET requests to retrieve all student records for the authenticated teacher.
func getAllgodbstudentsAsTeacher(w http.ResponseWriter, r *http.Request) {
	teacherID, ok := r.Context().Value(contextKeyTeacherID).(string)
	if !ok || teacherID == "" {
		http.Error(w, "Forbidden: TeacherAlt ID not found in session", http.StatusForbidden)
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

func getAllgodbstudents(w http.ResponseWriter, r *http.Request){
	// CHQ: Gemini AI changed fetching global vatiable to retrieving variable from context

	// Retrieve isAdmin from context
    isAdmin, ok := r.Context().Value(contextKeyIsAdmin).(bool)
    if !ok {
        // Fallback for safety, though middleware should ensure it's set
        http.Error(w, "Forbidden: Role not determined", http.StatusForbidden)
        return
    }

	if (isAdmin) {
		getAllgodbstudentsAsAdmin(w)
	} else {
		getAllgodbstudentsAsTeacher(w, r)
	}
}


// updateStudent handles PUT requests to update an existing student record, with an ownership check.
func updateStudentAsAdmin(w http.ResponseWriter, r *http.Request) {
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
	
	query := `UPDATE godbstudents SET first_name = $1, last_name = $2, email = $3, major = $4 WHERE id = $5`
	result, err := db.Exec(query, student.FirstName, student.LastName, student.Email, student.Major, id)
	
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
		http.Error(w, "Student not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Student updated successfully"})
}

// updateStudent handles PUT requests to update an existing student record, with an ownership check.
func updateStudentAsTeacher(w http.ResponseWriter, r *http.Request) {
	teacherID, ok := r.Context().Value(contextKeyTeacherID).(string)
	if !ok || teacherID == "" {
		http.Error(w, "Forbidden: TeacherAlt ID not found in session", http.StatusForbidden)
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

func updateStudent(w http.ResponseWriter, r *http.Request){
	// CHQ: Gemini AI changed fetching global vatiable to retrieving variable from context

	// Retrieve isAdmin from context
    isAdmin, ok := r.Context().Value(contextKeyIsAdmin).(bool)
    if !ok {
        // Fallback for safety, though middleware should ensure it's set
        http.Error(w, "Forbidden: Role not determined", http.StatusForbidden)
        return
    }

	if (isAdmin) {
		updateStudentAsAdmin(w, r)
	} else {
		updateStudentAsTeacher(w, r)
	}
}


// updateStudentAlt handles PATCH requests to update an existing student record, with an ownership check.
func updateStudentAltAsAdmin(w http.ResponseWriter, r *http.Request) {
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
 
	query := `UPDATE godbstudents SET first_name = $1, last_name = $2, email = $3, major = $4 WHERE id = $5`
	result, err := db.Exec(query, student.FirstName, student.LastName, student.Email, student.Major, id)
	
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
		http.Error(w, "Student not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Student updated successfully"})
}

func updateStudentAltAsTeacher(w http.ResponseWriter, r *http.Request) {
	teacherID, ok := r.Context().Value(contextKeyTeacherID).(string)
	if !ok || teacherID == "" {
		http.Error(w, "Forbidden: TeacherAlt ID not found in session", http.StatusForbidden)
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

func updateStudentAlt(w http.ResponseWriter, r *http.Request){
	// CHQ: Gemini AI changed fetching global vatiable to retrieving variable from context

	// Retrieve isAdmin from context
    isAdmin, ok := r.Context().Value(contextKeyIsAdmin).(bool)
    if !ok {
        // Fallback for safety, though middleware should ensure it's set
        http.Error(w, "Forbidden: Role not determined", http.StatusForbidden)
        return
    }

	if (isAdmin) {
		updateStudentAltAsAdmin(w, r)
	} else {
		updateStudentAltAsTeacher(w, r)
	}
}


// deleteStudent handles DELETE requests to delete a student record by ID, with an ownership check.
func deleteStudentAltAsAdmin(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid student ID", http.StatusBadRequest)
		return
	}
	
	// Ensure the student belongs to the authenticated teacher.
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

func deleteStudentAltAsTeacher(w http.ResponseWriter, r *http.Request) {
	teacherID, ok := r.Context().Value(contextKeyTeacherID).(string)
	if !ok || teacherID == "" {
		http.Error(w, "Forbidden: TeacherAlt ID not found in session", http.StatusForbidden)
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

func deleteStudent(w http.ResponseWriter, r *http.Request){
	// CHQ: Gemini AI changed fetching global vatiable to retrieving variable from context

	// Retrieve isAdmin from context
    isAdmin, ok := r.Context().Value(contextKeyIsAdmin).(bool)
    if !ok {
        // Fallback for safety, though middleware should ensure it's set
        http.Error(w, "Forbidden: Role not determined", http.StatusForbidden)
        return
    }

	if (isAdmin) {
		deleteStudentAltAsAdmin(w, r)
	} else {
		deleteStudentAltAsTeacher(w, r)
	}
}