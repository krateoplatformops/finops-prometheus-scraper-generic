package config

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"prometheus-scraper-generic/pkg/utils"

	"gopkg.in/yaml.v3"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

/*
* This struct is used to parse the YAML config file for databricks.
 */
type ConfigFromFile struct {
	DatabaseConfigRef DatabaseConfigRef `yaml:"databaseConfigRef"`
	Exporter          Exporter          `yaml:"exporter"`
}

type DatabaseConfigRef struct {
	Name      string `yaml:"name"`
	Namespace string `yaml:"namespace"`
}

type Config struct {
	DatabaseConfig DatabaseConfig `yaml:"databaseConfig"`
	Exporter       Exporter       `yaml:"exporter"`
}

type DatabaseConfig struct {
	Host         string `yaml:"host" json:"host"`
	Token        string `yaml:"token" json:"token"`
	NotebookPath string `yaml:"notebookPath" json:"notebookPath"`
	ClusterName  string `yaml:"clusterName" json:"clusterName"`
}

type Exporter struct {
	Url                  string `yaml:"url"`
	PollingIntervalHours int    `yaml:"pollingIntervalHours"`
	TableName            string `yaml:"tableName"`
}

func ParseConfigFile(file string) (Config, error) {
	fileReader, err := os.OpenFile(file, os.O_RDONLY, 0600)
	if err != nil {
		return Config{}, err
	}
	defer fileReader.Close()
	data, err := io.ReadAll(fileReader)
	if err != nil {
		return Config{}, err
	}

	parse := ConfigFromFile{}

	err = yaml.Unmarshal(data, &parse)
	if err != nil {
		return Config{}, err
	}

	configuration := Config{}
	configuration.Exporter = parse.Exporter

	// Get CR from Kubernetes
	inClusterConfig, err := rest.InClusterConfig()
	utils.Fatal(err)
	clientset, err := kubernetes.NewForConfig(inClusterConfig)
	utils.Fatal(err)
	jsonData, err := clientset.RESTClient().
		Get().
		AbsPath("/apis/finops.krateo.io/v1").
		Namespace(parse.DatabaseConfigRef.Namespace).
		Resource("databaseconfigs").
		Name(parse.DatabaseConfigRef.Name).
		DoRaw(context.TODO())
	utils.Fatal(err)

	var crdResponse CRDResponse
	err = json.Unmarshal(jsonData, &crdResponse)
	utils.Fatal(err)

	configuration.DatabaseConfig = crdResponse.Spec

	return configuration, nil
}

type CRDResponse struct {
	Spec DatabaseConfig `json:"spec"`
}
