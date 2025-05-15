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
	"github.com/krateoplatformops/finops-prometheus-scraper-generic/internal/helpers/kube/endpoints"
	"github.com/krateoplatformops/finops-prometheus-scraper-generic/internal/helpers/kube/httpcall"
	"github.com/krateoplatformops/finops-prometheus-scraper-generic/internal/helpers/kube/secrets"

	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"k8s.io/client-go/rest"

	"github.com/rs/zerolog/log"

	finopsdatatypes "github.com/krateoplatformops/finops-data-types/api/v1"
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

func WriteProm(api finopsdatatypes.API) (int64, error) {
	time.Sleep(2 * time.Second)
	out, err := os.Create(promFilePath)
	if err != nil {
		return -1, err
	}
	defer out.Close()

	rc, _ := rest.InClusterConfig()
	endpoint, err := endpoints.Resolve(context.TODO(), endpoints.ResolveOptions{
		RESTConfig: rc,
		API:        &api,
	})
	if err != nil {
		return -1, err
	}

	log.Logger.Info().Msgf("Request URL: %s", endpoint.ServerURL)

	res := &http.Response{StatusCode: 500}
	err_call := fmt.Errorf("")

	for ok := true; ok; ok = (err_call != nil || res.StatusCode != 200) {
		httpClient, err := httpcall.HTTPClientForEndpoint(endpoint)
		if err != nil {
			log.Logger.Error().Err(err)
			continue
		}

		res, err_call = httpcall.Do(context.TODO(), httpClient, httpcall.Options{
			API:      &api,
			Endpoint: endpoint,
		})

		if err_call == nil && res.StatusCode != 200 {
			log.Warn().Msgf("Received status code %d", res.StatusCode)
		} else {
			log.Error().Err(err).Msg("error during call to obtain prometheus metrics")
			continue
		}
		log.Logger.Warn().Msgf("Retrying connection in 5s...")
		time.Sleep(5 * time.Second)

		log.Logger.Info().Msgf("Parsing Endpoint again...")
		rc, _ := rest.InClusterConfig()
		endpoint, err = endpoints.Resolve(context.Background(), endpoints.ResolveOptions{
			RESTConfig: rc,
			API:        &api,
		})
		if err != nil {
			continue
		}
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return -1, fmt.Errorf("bad status: %s", res.Status)
	}

	written, err := io.Copy(out, res.Body)
	if err != nil {
		return -1, err
	}
	return written, nil
}

func main() {
	config, err := config.ParseConfigFile("/config/config.yaml")
	if err != nil {
		log.Error().Err(err).Msg("error occured while parsing scraper configuration, halting...")
		return
	}

	cfg, err := rest.InClusterConfig()
	if err != nil {
		log.Error().Err(err).Msg("error occured while retrieving InClusterConfig, halting...")
		return
	}

	uploadServiceURL := os.Getenv("URL_DB_WEBSERVICE")
	time.Sleep(5 * time.Second)
	for {
		log.Logger.Info().Msg("Starting loop...")

		passwordSecret, err := secrets.Get(context.Background(), cfg, &config.DatabaseConfig.PasswordSecretRef)
		if err != nil {
			log.Error().Err(err).Msg("error occured while retrieving password secret, continuing to next cycle...")
			continue
		}
		usernamePassword := &apis.UsernamePassword{
			Username: string(config.DatabaseConfig.Username),
			Password: string(passwordSecret.Data[config.DatabaseConfig.PasswordSecretRef.Key]),
		}

		// Get and verify metrics data
		first_file_size, err := WriteProm(config.Exporter.API)
		if err != nil {
			log.Error().Err(err).Msg("Error while writing prometheus file")
		}

		second_file_size := int64(-1)
		for first_file_size != second_file_size || first_file_size == 0 {
			second_file_size = first_file_size
			first_file_size, err = WriteProm(config.Exporter.API)
			if err != nil {
				log.Error().Err(err).Msg("error while writing prometheus file (loop)")
			}
			seconds := 5 * time.Second
			log.Logger.Info().Msgf("Exporter is still updating or has not published anything yet, waiting %s...", seconds)
			time.Sleep(seconds)
		}

		// Parse metrics
		mf, err := parseMF(promFilePath)
		if err != nil {
			log.Error().Err(err).Msg("Error while reading prometheus metrics from file")
		}

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
			log.Logger.Warn().Msgf("Error uploading metrics: %v, continuing...", err)
		}

		// Wait for next polling interval
		time.Sleep(config.Exporter.PollingInterval.Duration)
	}
}
