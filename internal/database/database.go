package database

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"

	"github.com/krateoplatformops/finops-prometheus-scraper-generic/apis"
)

const maxBatchSize = 100 // Number of records to send in each batch

func sanitizeColumnValue(value string) string {
	// Remove or replace any unsupported characters
	unsupportedCharsRegex := regexp.MustCompile(`[^\w\s-.]`)
	sanitizedValue := unsupportedCharsRegex.ReplaceAllString(value, "")

	// Trim leading/trailing whitespace
	sanitizedValue = strings.TrimSpace(sanitizedValue)

	// Replace multiple consecutive spaces with a single space
	sanitizedValue = strings.ReplaceAll(sanitizedValue, "  ", " ")

	return sanitizedValue
}

func UploadMetrics(metrics []apis.MetricRecord, uploadServiceURL string, config apis.Config, usernamePassword *apis.UsernamePassword) error {
	for i := range metrics {
		for label, value := range metrics[i].Labels {
			metrics[i].Labels[label] = sanitizeColumnValue(value)
		}
	}

	// Split metrics into batches
	for i := 0; i < len(metrics); i += maxBatchSize {
		end := i + maxBatchSize
		if end > len(metrics) {
			end = len(metrics)
		}
		batch := metrics[i:end]

		// Convert batch to JSON
		jsonData, err := json.Marshal(batch)
		if err != nil {
			return fmt.Errorf("error marshaling data: %v", err)
		}

		// Create request
		url := fmt.Sprintf("%s/upload?table=%s&type=%s", uploadServiceURL, config.Exporter.TableName, config.Exporter.MetricType)
		req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
		if err != nil {
			return fmt.Errorf("error creating request: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.SetBasicAuth(usernamePassword.Username, usernamePassword.Password)

		// Send request
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return fmt.Errorf("error sending request: %v", err)
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		// Check response
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("upload failed with status %d: %s", resp.StatusCode, string(body))
		}

		fmt.Println(string(body))

		fmt.Printf("Successfully uploaded batch of %d records\n", len(batch))
	}
	return nil
}
