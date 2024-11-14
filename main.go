package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/krateoplatformops/finops-prometheus-scraper-generic/apis"
	"github.com/krateoplatformops/finops-prometheus-scraper-generic/internal/config"
	"github.com/krateoplatformops/finops-prometheus-scraper-generic/internal/database"
	"github.com/krateoplatformops/finops-prometheus-scraper-generic/internal/helpers/kube/secrets"
	"github.com/krateoplatformops/finops-prometheus-scraper-generic/internal/utils"

	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"k8s.io/client-go/rest"
)

const (
	promFilePath = "/temp/temp.prom"
)

func parseMF(path string) (map[string]*dto.MetricFamily, error) {
	reader, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	var parser expfmt.TextParser
	mf, err := parser.TextToMetricFamilies(reader)
	if err != nil {
		return nil, err
	}
	return mf, nil
}

func WriteProm(url string) (int64, error) {
	time.Sleep(2 * time.Second)
	out, err := os.Create(promFilePath)
	if err != nil {
		return -1, err
	}
	defer out.Close()

	resp, err := http.Get(url)
	for err != nil {
		fmt.Println(err, "\n\t > Cannot reach exporter, waiting 1 second and retrying...")
		time.Sleep(1 * time.Second)
		resp, err = http.Get(url)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return -1, fmt.Errorf("bad status: %s", resp.Status)
	}

	written, err := io.Copy(out, resp.Body)
	if err != nil {
		return -1, err
	}
	return written, nil
}

func main() {
	config, err := config.ParseConfigFile("/config/config.yaml")
	utils.Fatal(err)

	cfg, err := rest.InClusterConfig()
	if err != nil {
		utils.Fatal(err)
		fmt.Println("error occured while retrieving InClusterConfig, continuing to next cycle...")
		return
	}

	uploadServiceURL := os.Getenv("URL_DB_WEBSERVICE")

	for {
		fmt.Println("Starting loop...")

		passwordSecret, err := secrets.Get(context.Background(), cfg, &config.DatabaseConfig.PasswordSecretRef)
		if err != nil {
			utils.Fatal(err)
			fmt.Println("error occured while retrieving password secret, continuing to next cycle...")
			continue
		}
		usernamePassword := &apis.UsernamePassword{
			Username: string(config.DatabaseConfig.Username),
			Password: string(passwordSecret.Data[config.DatabaseConfig.PasswordSecretRef.Key]),
		}

		// Get and verify metrics data
		first_file_size, err := WriteProm(config.Exporter.Url)
		utils.Fatal(err)

		second_file_size := int64(-1)
		for first_file_size != second_file_size || first_file_size == 0 {
			second_file_size = first_file_size
			first_file_size, err = WriteProm(config.Exporter.Url)
			utils.Fatal(err)
			fmt.Println("Exporter is still updating or has not published anything yet, waiting 60 seconds...")
			time.Sleep(60 * time.Second)
		}

		// Parse metrics
		mf, err := parseMF(promFilePath)
		utils.Fatal(err)

		// Convert metrics to records
		var metrics []apis.MetricRecord
		timestamp := time.Now().Unix()

		for _, value := range mf {
			for _, metric := range value.Metric {
				record := apis.MetricRecord{
					Labels:    make(map[string]string),
					Value:     metric.GetGauge().GetValue(),
					Timestamp: timestamp,
				}

				// Convert labels to map
				for _, label := range metric.Label {
					record.Labels[*label.Name] = *label.Value
				}

				metrics = append(metrics, record)
			}
		}

		// Upload metrics in batches
		err = database.UploadMetrics(metrics, uploadServiceURL, config, usernamePassword)
		if err != nil {
			fmt.Printf("Error uploading metrics: %v\n", err)
		}

		// Wait for next polling interval
		time.Sleep(time.Duration(config.Exporter.PollingIntervalHours) * time.Hour)
	}
}
