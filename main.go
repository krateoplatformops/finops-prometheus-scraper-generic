package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/krateoplatformops/finops-prometheus-scraper-generic/pkg/config"
	"github.com/krateoplatformops/finops-prometheus-scraper-generic/pkg/store"
	"github.com/krateoplatformops/finops-prometheus-scraper-generic/pkg/utils"

	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
)

const (
	promFilePath = "/temp/temp.prom"
)

var (
	promFilePathRemote = "/filestore/temp" + os.Getenv("HOSTNAME")
)

/*
* This function utilizes the official Prometheus parses to obtain the metrics. The metrics need to be stored in a file.
* @param path: the path to the metrics file.
 */
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

/*
* Given an URL, the content is written to a temporary file.
* @param url: the URL to the metrics
 */
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
	//time.Sleep(1 * time.Minute)
	config, err := config.ParseConfigFile("/config/config.yaml")
	utils.Fatal(err)

	wsClient := store.WSclient{}
	err = wsClient.Init(config)
	utils.Fatal(err)
	wsClient.CheckCluster()

	for {
		fmt.Println("Starting loop...")

		// If the scraper and exporter are started together, it may take some time for the exporter to render all the prometheus metrics
		// This means that it might answer, and it answers 200 OK, however, the body is either empty or incomplete
		// The multiple requests allow the verify that the output file size has stabilized before proceeding
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

		mf, err := parseMF(promFilePath)
		utils.Fatal(err)

		keyValuePairs := []string{}
		for _, value := range mf {
			row := ""
			for _, label := range value.Metric[0].Label {
				row += *label.Name + "=" + *label.Value + ","
			}
			row += "Value=" + strconv.FormatFloat(value.Metric[0].GetGauge().GetValue(), 'f', -1, 64)
			keyValuePairs = append(keyValuePairs, row)
		}

		wsClient.UploadFile(promFilePathRemote, strings.Join(keyValuePairs, "\n"))

		result, jobId := wsClient.CreateJob("upload_table_row_job", "upload of row table with prometheus scraper", "")
		fmt.Printf("Creating job: %t\n", result)
		if result {
			wsClient.RunJob(jobId, config.Exporter.TableName, promFilePathRemote)
		}

		time.Sleep(time.Duration(config.Exporter.PollingIntervalHours) * time.Hour)
	}
}
