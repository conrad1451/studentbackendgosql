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
	"github.com/gorilla/mux"

	"github.com/descope/go-sdk/descope/client"
	_ "github.com/lib/pq"
)

// Utilizing the context package allows for the transmission of context capabilities like cancellation
//      signals during the function call. In cases where context is absent, the context.Background()
//      function serves as a viable alternative.
//      Utilizing context within the Descope GO SDK is supported within versions 1.6.0 and higher.

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

func isPrefixAllowed(anOrigin string, anAllowedPrefix string) bool {
	return strings.HasPrefix(anOrigin, "https://"+anAllowedPrefix) || strings.HasPrefix(anOrigin, "http://"+anAllowedPrefix);
}


func isAnyPrefixAllowed(origin string, prefix1 string, prefix2 string, prefix3 string) bool {
	return isPrefixAllowed(origin, prefix1) || isPrefixAllowed(origin, prefix2)|| isPrefixAllowed(origin, prefix3)
}

// // CHQ: Gemini AI debugged function
// // corsMiddleware dynamically sets the Access-Control-Allow-Origin header
// // for any origin that starts with a specific pattern.
// func corsMiddleware(next http.Handler) http.Handler {
//     return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//         origin := r.Header.Get("Origin")
//         sitePrefix1 := os.Getenv("FRONT_END_SITE_PREFIX_1")
//         sitePrefix2 := os.Getenv("FRONT_END_SITE_PREFIX_2")
//         sitePrefix3 := os.Getenv("TESTER_1")

//         if isAnyPrefixAllowed(origin, sitePrefix1, sitePrefix2, sitePrefix3) {
//             w.Header().Set("Access-Control-Allow-Origin", origin)
//             w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
//             w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
//             w.Header().Set("Access-Control-Allow-Credentials", "true") // This is important for session tokens
//         }

//         // Handle preflight requests
//         if r.Method == http.MethodOptions {
//             if isAnyPrefixAllowed(origin, sitePrefix1, sitePrefix2, sitePrefix3) {
//                 w.WriteHeader(http.StatusOK)
//                 return
//             } else {
//                 http.Error(w, "CORS: Not Allowed", http.StatusForbidden)
//                 return
//             }
//         }

//         // Pass the request to the next handler.
//         next.ServeHTTP(w, r)
//     })
// }
var listOfDBConnections = []string{"NEON_STUDENT_RECORDS_DB", "PROJECT2_DB", "GOOGLE_CLOUD_SQL", "GOOGLE_VM_HOSTED_SQL"}

// CHQ: Gemini AI generated this function
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

	// CHQ: Gemini AI added descope verification
    // Initialize Descope client once at the start
    // var err error
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

	// All routes now go through the mux router
	router.HandleFunc("/", helloHandler)
	router.HandleFunc("/favicon.ico", faviconHandler)

   // Protected routes (require session validation)
    protectedRoutes := router.PathPrefix("/api").Subrouter()
    protectedRoutes.Use(sessionValidationMiddleware) // Apply middleware to all routes in this subrouter
    protectedRoutes.HandleFunc("/godbstudents", createStudent).Methods("POST")
    protectedRoutes.HandleFunc("/godbstudents/{id}", getStudent).Methods("GET")
    protectedRoutes.HandleFunc("/godbstudents", getAllgodbstudents).Methods("GET")
    protectedRoutes.HandleFunc("/godbstudents/{id}", updateStudent).Methods("PUT")
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
	// allowedOrigins := handlers.AllowedOrigins([]string{"*"})
	// allowedMethods := handlers.AllowedMethods([]string{"GET", "POST", "PUT", "DELETE", "OPTIONS"})
	// allowedHeaders := handlers.AllowedHeaders([]string{"Content-Type", "Authorization"})
	// corsRouter := handlers.CORS(allowedOrigins, allowedMethods, allowedHeaders)(router)
	// --- End of CORS Setup ---

	// CHQ: Gemini AI replaced corsRouter with this
	router.Use(corsMiddleware)
	
	// Start the HTTP server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" // Default port
	}
	
	fmt.Printf("Server listening on port %s...\n", port)
	// log.Fatal(http.ListenAndServe(":"+port, corsRouter))
	// CHQ: Gemini AI replaced the above with the below
    log.Fatal(http.ListenAndServe(":"+port, router)) // Note: Use the original router
}

// CHQ: Gemini AI generated function
// helloHandler is the function that will be executed for requests to the "/" route.
func helloHandler(w http.ResponseWriter, r *http.Request) {
	// Set the Content-Type header to inform the browser that the response is HTML.
	w.Header().Set("Content-Type", "text/html")
	
	// Write the "Hello, World!" message as the response body.
	// This will be a simple, unstyled page with the text.
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
        // Validate the session token and get the Descope token
        authorized, token, err := descopeClient.Auth.ValidateSessionWithToken(ctx, sessionToken)
        if err != nil || !authorized {
            log.Printf("Session validation failed: %v", err)
            http.Error(w, "Unauthorized: Invalid session token", http.StatusUnauthorized)
            return
        }

        // Extract the userID from the validated token
        userID := token.ID
        if userID == "" {
            http.Error(w, "Unauthorized: User ID not found in token", http.StatusUnauthorized)
            return
        }
        
        // Extract the teacher ID from the token's custom claims or other fields
        // This is a placeholder, you'll need to know the specific claim key.
        // For this example, we assume the teacher ID is an integer stored in a custom claim called 'teacher_id'.
        // var teacherID int
        // // Let's assume you have a custom claim for teacher_id
        // if val, ok := token.CustomClaims["teacher_id"]; ok {
        //     if floatVal, ok := val.(float64); ok {
        //         teacherID = int(floatVal)
        //     }
        // }
        
        // Store the user ID and teacher ID in the request's context
        ctxWithUserID := context.WithValue(ctx, contextKeyUserID, userID)
        // ctxWithIDs := context.WithValue(ctxWithUserID, contextKeyTeacherID, teacherID)
        
        // Pass the request with the updated context to the next handler
        // next.ServeHTTP(w, r.WithContext(ctxWithIDs))
		next.ServeHTTP(w, r.WithContext(ctxWithUserID))
    })
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

// CHQ: Gemini AI refactored to filter SQL query by teacherID
// getAllgodbstudents handles GET requests to retrieve all student records.
// It now supports an optional `teacherID` query parameter to filter results.
func getAllgodbstudents(w http.ResponseWriter, r *http.Request) {
    // 1. Get the teacherID from the request context (added by your middleware).
    teacherID, ok := r.Context().Value(contextKeyTeacherID).(int)
    if !ok || teacherID == 0 {
        http.Error(w, "Forbidden: Teacher ID not found in session", http.StatusForbidden)
        return
    }

    var students []Student
    // 2. Modify the SQL query to filter by the authenticated teacherID.
    // The WHERE clause is crucial here.
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
