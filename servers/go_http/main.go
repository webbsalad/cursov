package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
)

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "File upload failed", http.StatusInternalServerError)
		return
	}
	defer file.Close()

	out, err := os.Create("uploads/" + r.URL.Path[1:])
	if err != nil {
		http.Error(w, "Failed to save file", http.StatusInternalServerError)
		return
	}
	defer out.Close()

	io.Copy(out, file)
	fmt.Fprintf(w, "File uploaded successfully")
}

func downloadHandler(w http.ResponseWriter, r *http.Request) {
	filepath := "uploads/" + r.URL.Path[1:]
	file, err := os.Open(filepath)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer file.Close()

	w.Header().Set("Content-Disposition", "attachment; filename="+r.URL.Path[1:])
	w.Header().Set("Content-Type", "application/octet-stream")
	io.Copy(w, file)
}

func main() {
	http.HandleFunc("/upload/", uploadHandler)
	http.HandleFunc("/download/", downloadHandler)

	fmt.Println("Starting Go HTTP server on port 9001...")
	http.ListenAndServe(":9001", nil)
}
