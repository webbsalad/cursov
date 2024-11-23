package main

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"os/exec"
	"sync"
	"time"
)

var servers = []struct {
	name string
	url  string
	cmd  *exec.Cmd
}{
	{"python_flask", "http://localhost:8002", exec.Command("python3", "servers/python_flask/main.py")},
	{"python_fastapi", "http://localhost:8003", exec.Command("python3", "servers/python_fastapi/main.py")},
	{"go_http", "http://localhost:9001", exec.Command("go", "run", "servers/go_http/main.go")},
	{"go_gin", "http://localhost:9002", exec.Command("go", "run", "servers/go_gin/main.go")},
}

func checkServerAvailability(serverURL string) bool {
	resp, err := http.Get(serverURL + "/health")
	if err != nil {
		fmt.Printf("Server %s is not responding: %v\n", serverURL, err)
		return false
	}
	resp.Body.Close()
	return true
}

func uploadFiles(serverURL string, files []string) (time.Duration, error) {
	start := time.Now()
	for _, file := range files {
		fileData, err := os.ReadFile(file)
		if err != nil {
			fmt.Printf("Error reading file %s: %v\n", file, err)
			return 0, err
		}
		resp, err := http.Post(serverURL+"/upload/"+file, "application/octet-stream", bytes.NewReader(fileData))
		if err != nil {
			fmt.Printf("Error uploading file %s to %s: %v\n", file, serverURL, err)
			return 0, err
		}
		resp.Body.Close()
	}
	return time.Since(start), nil
}

func downloadFiles(serverURL string, files []string) (time.Duration, error) {
	start := time.Now()
	for _, file := range files {
		resp, err := http.Get(serverURL + "/download/" + file)
		if err != nil {
			fmt.Printf("Error downloading file %s from %s: %v\n", file, serverURL, err)
			return 0, err
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}
	return time.Since(start), nil
}

func uploadFilesParallel(serverURL string, files []string) (time.Duration, error) {
	start := time.Now()
	var wg sync.WaitGroup

	for _, file := range files {
		wg.Add(1)
		go func(file string) {
			defer wg.Done()
			fileData, err := os.ReadFile(file)
			if err != nil {
				fmt.Printf("Error reading file %s: %v\n", file, err)
				return
			}
			resp, err := http.Post(serverURL+"/upload/"+file, "application/octet-stream", bytes.NewReader(fileData))
			if err != nil {
				fmt.Printf("Error uploading file %s to %s: %v\n", file, serverURL, err)
				return
			}
			resp.Body.Close()
		}(file)
	}

	wg.Wait()
	return time.Since(start), nil
}

func downloadFilesParallel(serverURL string, files []string) (time.Duration, error) {
	start := time.Now()
	var wg sync.WaitGroup

	for _, file := range files {
		wg.Add(1)
		go func(file string) {
			defer wg.Done()
			resp, err := http.Get(serverURL + "/download/" + file)
			if err != nil {
				fmt.Printf("Error downloading file %s from %s: %v\n", file, serverURL, err)
				return
			}
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}(file)
	}

	wg.Wait()
	return time.Since(start), nil
}

func clearUploads() error {
	uploadsDir := "uploads"
	err := os.RemoveAll(uploadsDir)
	if err != nil {
		return err
	}
	return os.Mkdir(uploadsDir, os.ModePerm)
}

func runTestRound(round int, files []string, uploadResults, downloadResults, uploadResultsParallel, downloadResultsParallel map[string][]int64) error {
	fileCount := 1000 * round
	if fileCount > len(files) {
		fileCount = len(files)
	}
	currentFiles := files[:fileCount]

	for _, server := range servers {
		if err := clearUploads(); err != nil {
			fmt.Printf("Error clearing uploads directory: %v\n", err)
			return err
		}

		if !checkServerAvailability(server.url) {
			fmt.Printf("Skipping server %s as it is not available.\n", server.name)
			continue
		}

		uploadTime, err := uploadFiles(server.url, currentFiles)
		if err != nil {
			fmt.Printf("Error during upload to server %s: %v\n", server.name, err)
			return err
		}
		downloadTime, err := downloadFiles(server.url, currentFiles)
		if err != nil {
			fmt.Printf("Error during download from server %s: %v\n", server.name, err)
			return err
		}
		uploadResults[server.name] = append(uploadResults[server.name], uploadTime.Milliseconds())
		downloadResults[server.name] = append(downloadResults[server.name], downloadTime.Milliseconds())

		uploadTimeParallel, err := uploadFilesParallel(server.url, currentFiles)
		if err != nil {
			fmt.Printf("Error during parallel upload to server %s: %v\n", server.name, err)
			return err
		}
		downloadTimeParallel, err := downloadFilesParallel(server.url, currentFiles)
		if err != nil {
			fmt.Printf("Error during parallel download from server %s: %v\n", server.name, err)
			return err
		}
		uploadResultsParallel[server.name] = append(uploadResultsParallel[server.name], uploadTimeParallel.Milliseconds())
		downloadResultsParallel[server.name] = append(downloadResultsParallel[server.name], downloadTimeParallel.Milliseconds())
	}

	fmt.Printf("Test round %d completed successfully.\n", round)
	return nil
}

func startServers() error {
	for _, server := range servers {
		if err := server.cmd.Start(); err != nil {
			fmt.Printf("Failed to start server %s: %v\n", server.name, err)
			return err
		}
		time.Sleep(5000 * time.Millisecond)
		if !checkServerAvailability(server.url) {
			fmt.Printf("Server %s is not available after start.\n", server.name)
		}
	}
	return nil
}

func stopServers() {
	for _, server := range servers {
		if err := server.cmd.Process.Kill(); err != nil {
			fmt.Printf("Error stopping server %s: %v\n", server.name, err)
		} else {
			server.cmd.Wait()
		}
	}
}

func writeResultsToCSV(writer *csv.Writer, label string, results map[string][]int64) {
	writer.Write(append([]string{label}, createRoundHeaders(len(results[servers[0].name]))...))
	for server, times := range results {
		writer.Write(append([]string{server}, convertToStringSlice(times)...))
	}
	writer.Flush()
}

func createRoundHeaders(rounds int) []string {
	headers := make([]string, rounds)
	for i := 0; i < rounds; i++ {
		headers[i] = fmt.Sprintf("%d", i+1)
	}
	return headers
}

func convertToStringSlice(times []int64) []string {
	result := make([]string, len(times))
	for i, t := range times {
		result[i] = fmt.Sprintf("%d", t)
	}
	return result
}

func main() {
	fmt.Println("Initializing test files...")
	files := []string{}
	for i := 1; i <= 1000; i++ {
		files = append(files, fmt.Sprintf("data/file_%d.json", i))
	}
	files = append(files, "data/large_text_file.txt")

	fmt.Println("Starting servers...")
	if err := startServers(); err != nil {
		fmt.Printf("Error starting servers: %v\n", err)
		return
	}
	defer stopServers()

	fmt.Println("Creating results.csv and results_mn.csv for storing test results...")
	f1, err := os.Create("results/results.csv")
	if err != nil {
		fmt.Printf("Error creating CSV file: %v\n", err)
		return
	}
	defer f1.Close()
	writer1 := csv.NewWriter(f1)

	f2, err := os.Create("results/results_mn.csv")
	if err != nil {
		fmt.Printf("Error creating CSV file: %v\n", err)
		return
	}
	defer f2.Close()
	writer2 := csv.NewWriter(f2)

	uploadResults := make(map[string][]int64)
	downloadResults := make(map[string][]int64)
	uploadResultsParallel := make(map[string][]int64)
	downloadResultsParallel := make(map[string][]int64)

	const totalRounds = 100
	fmt.Printf("\033[32m%.2d%%\033[0m\n", 0)
	for round := 1; round <= totalRounds; round++ {
		if err := runTestRound(round, files, uploadResults, downloadResults, uploadResultsParallel, downloadResultsParallel); err != nil {
			fmt.Printf("Error during test round %d: %v\n", round, err)
			return
		}
		fmt.Printf("\033[32m%.2f%%\033[0m\n", math.Round((float64(round)/float64(totalRounds))*10000)/100)
	}

	writeResultsToCSV(writer1, "Server up", uploadResults)
	writer1.Write([]string{})
	writeResultsToCSV(writer1, "Server dw", downloadResults)

	writeResultsToCSV(writer2, "Server up (parallel)", uploadResultsParallel)
	writer2.Write([]string{})
	writeResultsToCSV(writer2, "Server dw (parallel)", downloadResultsParallel)

	fmt.Println("Test rounds completed successfully. Results saved to CSV files.")
}
