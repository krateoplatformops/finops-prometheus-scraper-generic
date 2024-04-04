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
func WriteProm(url string) error {
	resp, err := http.Get(url)
	for err != nil {
		resp, err = http.Get(url)
		fmt.Println("Cannot reach exporter, waiting 1 second and retrying...")
		time.Sleep(1 * time.Second)
	}
	defer resp.Body.Close()

	out, err := os.Create(promFilePath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	return nil
}

func main() {
	config, err := config.ParseConfigFile("/config/config.yaml")
	utils.Fatal(err)

	wsClient := store.WSclient{}
	wsClient.Init(config)
	wsClient.CheckCluster()

	for {
		exporter := config.Exporter
		err = WriteProm(exporter.Url)
		utils.Fatal(err)

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
			wsClient.RunJob(jobId, exporter.TableName, promFilePathRemote)
		}

		time.Sleep(time.Duration(exporter.PollingIntervalHours) * time.Hour)
	}
}
