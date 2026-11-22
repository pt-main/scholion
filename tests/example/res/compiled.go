// server.go.schol
package main

import "fmt"
import "net/http"

// Generated var 1
// Generated var 2
// Generated var 3
var Port1 int = 8001
var Port2 int = 8002
var Port3 int = 8003

// DSL will be: ROUTE: <method> <path> => <handler>

var DebugMode bool = true
// Debug mode is ON!
// Auto-generated handler: GET /users
func HandleGetUsers(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusOK)
    fmt.Fprintf(w, "Handling %s %s", r.Method, r.URL.Path)
}

// Auto-generated handler: POST /users
func HandleCreateUser(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusOK)
    fmt.Fprintf(w, "Handling %s %s", r.Method, r.URL.Path)
}



func main() {
	fmt.Println("Server starting...")
}
