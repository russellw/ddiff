package controllers

import "net/http"

func UserHandler(w http.ResponseWriter, r *http.Request) {
    // Handle user requests
    w.Write([]byte("User endpoint"))
}