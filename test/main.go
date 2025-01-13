package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"time"

	"github.com/defskela/logger"
	"github.com/joho/godotenv"
)

const apiURL = "https://api.cloudconvert.com/v2"

var apiKey string

type JobRequest struct {
	Tasks map[string]interface{} `json:"tasks"`
}

type JobResponse struct {
	Data struct {
		ID    string `json:"id"`
		Tasks []struct {
			Operation string `json:"operation"`
			Status    string `json:"status"`
			Result    struct {
				Files []struct {
					URL string `json:"url"`
				} `json:"files,omitempty"`
			} `json:"result,omitempty"`
		} `json:"tasks"`
	} `json:"data"`
}

func initApiKey() error {
	if err := godotenv.Load(); err != nil {
		return err
	}
	apiKey = os.Getenv("CONVERT_TOKEN")
	if apiKey == "" {
		return fmt.Errorf("CONVERT_TOKEN not set in environment")
	}
	return nil
}

func main() {
	fileName := "Отчет.pdf"
	if err := initApiKey(); err != nil {
		logger.Warn("Failed to initialize API key", err)
		// return err
	}

	filePath := path.Join("files", fileName)
	logger.Info("Starting conversion of PDF to Word", "file", filePath)

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		logger.Warn("File not found", "file", filePath)
		// return fmt.Errorf("file %s not found", filePath)
	}

	fileContent, err := os.ReadFile(filePath)
	if err != nil {
		logger.Warn("Error reading file", err)
		// return err
	}
	fileBase64 := base64.StdEncoding.EncodeToString(fileContent)

	job := JobRequest{
		Tasks: map[string]interface{}{
			"import-my-file": map[string]interface{}{
				"operation": "import/base64",
				"file":      fileBase64,
				"filename":  fileName,
			},
			"convert-my-file": map[string]interface{}{
				"operation":     "convert",
				"input":         "import-my-file",
				"input_format":  "pdf",
				"output_format": "docx",
			},
			"export-my-file": map[string]interface{}{
				"operation": "export/url",
				"input":     "convert-my-file",
			},
		},
	}

	jsonData, err := json.Marshal(job)
	if err != nil {
		logger.Warn("Error marshaling job request", err)
		// return err
	}

	req, err := http.NewRequest("POST", apiURL+"/jobs", bytes.NewBuffer(jsonData))
	if err != nil {
		logger.Warn("Error creating HTTP request", err)
		// return err
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		logger.Warn("Error sending HTTP request", err)
		// return err
	}
	defer resp.Body.Close()

	var jobResp JobResponse
	if err := json.NewDecoder(resp.Body).Decode(&jobResp); err != nil {
		logger.Warn("Error decoding job response", err)
		// return err
	}

	jobID := jobResp.Data.ID
	logger.Debug("Job created", "jobID", jobID)

	for {
		status, url := checkJobStatus(jobID)
		if status == "error" {
			// return fmt.Errorf("conversion failed for job %s", jobID)
		}
		if status == "finished" && url != "" {
			if err := downloadFile(url, "files/output.docx"); err != nil {
				logger.Warn("Error downloading file", err)
				// return err
			}
			break
		}
		logger.Debug("Job processing, retrying in 5 seconds")
		time.Sleep(5 * time.Second)
	}

	logger.Info("File successfully converted", "file", "output.docx")
	// return nil
}

func checkJobStatus(jobID string) (string, string) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/jobs/%s", apiURL, jobID), nil)
	if err != nil {
		logger.Warn("Error creating status request", err)
		return "error", ""
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		logger.Warn("Error checking job status", err)
		return "error", ""
	}
	defer resp.Body.Close()

	var jobResp JobResponse
	if err := json.NewDecoder(resp.Body).Decode(&jobResp); err != nil {
		logger.Warn("Error decoding job status response", err)
		return "error", ""
	}

	for _, task := range jobResp.Data.Tasks {
		if task.Operation == "export/url" && task.Status == "finished" {
			if len(task.Result.Files) > 0 {
				return "finished", task.Result.Files[0].URL
			}
		}
	}

	return "processing", ""
}

func downloadFile(url, filepath string) error {
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("error downloading file: %w", err)
	}
	defer resp.Body.Close()

	out, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("error creating file: %w", err)
	}
	defer out.Close()

	if _, err := io.Copy(out, resp.Body); err != nil {
		return fmt.Errorf("error saving file: %w", err)
	}

	logger.Debug("File downloaded", "file", filepath)
	return nil
}
