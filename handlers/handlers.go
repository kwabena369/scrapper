// handlers/handlers.go
package handlers

import (
    "encoding/json"
    "log"
    "net/http"
)


//  this is the specific handler for taking care of the error that happen 
func RespondWithError(w http.ResponseWriter,code int,msg string){
	 if code >499 {
		// then we are going to give it 
      log.Printf("this is the error , ")
	 }

	 type errorMessage struct { 
		 Error string `json:"error"` 
	 }

	 RespondWithJSON(w,code,errorMessage{
		Error: "there is something wrong",
	 })
}

func RespondWithJSON(w http.ResponseWriter, code int, payload interface{}) { 
	// Capitalized to export
   dat, err := json.Marshal(payload)
    if err != nil {
        log.Printf("Failed to marshal JSON response: %v", payload)
        w.WriteHeader(500)
        return
    }
    w.Header().Add("Content-Type", "application/json")
    w.WriteHeader(code)
    w.Write(dat)
}

func TheLoggingMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        log.Printf("%s %s", r.Method, r.URL.Path)
        next.ServeHTTP(w, r)
    })
}