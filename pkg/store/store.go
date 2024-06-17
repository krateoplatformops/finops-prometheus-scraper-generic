package store

import (
	"context"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/krateoplatformops/finops-prometheus-scraper-generic/pkg/config"
	"github.com/krateoplatformops/finops-prometheus-scraper-generic/pkg/utils"

	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/databricks/databricks-sdk-go/service/files"
	"github.com/databricks/databricks-sdk-go/service/jobs"
)

const maximumChunkLength = 1000000

type WSclient struct {
	Client       *databricks.WorkspaceClient
	clusterId    string
	notebookPath string
	clusterName  string
}

func (db *WSclient) Init(config config.Config) {
	db.Client = databricks.Must(databricks.NewWorkspaceClient(&databricks.Config{
		Host:               config.DatabaseConfig.Host,
		Token:              config.DatabaseConfig.Token,
		InsecureSkipVerify: true,
	}))
	db.notebookPath = config.DatabaseConfig.NotebookPath
	db.clusterName = config.DatabaseConfig.ClusterName
}

func (db *WSclient) CheckCluster() {
	cluster, err := db.Client.Clusters.GetByClusterName(context.Background(), db.clusterName)
	utils.Fatal(err)

	if cluster.State.String() != "RUNNING" {
		fmt.Println("Cluster is not running, starting cluster")
		db.Client.Clusters.Start(context.Background(), compute.StartCluster{ClusterId: cluster.ClusterId})
	}
	db.Client.Clusters.WaitGetClusterRunning(context.Background(), cluster.ClusterId, 10*time.Minute, clusterRunningCallback)
	db.clusterId = cluster.ClusterId
}

func (db *WSclient) UploadFile(filePathRemote string, fileData string) {
	handle, err := db.Client.Dbfs.Create(context.Background(), files.Create{Path: filePathRemote, Overwrite: true})
	utils.Fatal(err)
	// If the file is larger than 1MB, split it into multiple data chunks
	fileDataBytes := []byte(fileData)
	numberOfChunks := int(len(fileDataBytes) / maximumChunkLength) // If the file is not larger than 1MB, then the number of chunks is zero
	fmt.Println("File size:", len(fileDataBytes), "Number of chunks to upload", numberOfChunks+1)
	// And only the final chunk length is used
	finalChunkLength := len(fileDataBytes) % maximumChunkLength
	for i := 0; i < numberOfChunks; i++ {
		chunk := fileDataBytes[i*maximumChunkLength : (i+1)*maximumChunkLength]
		fmt.Println("Uploading chunk on DBFS:", i)
		err = db.Client.Dbfs.AddBlock(context.Background(), files.AddBlock{Handle: handle.Handle, Data: base64.StdEncoding.EncodeToString(chunk)})
		utils.Fatal(err)
	}
	chunk := fileDataBytes[numberOfChunks*maximumChunkLength : (numberOfChunks)*maximumChunkLength+finalChunkLength]
	fmt.Println("Uploading final chunk on DBFS")
	err = db.Client.Dbfs.AddBlock(context.Background(), files.AddBlock{Handle: handle.Handle, Data: base64.StdEncoding.EncodeToString(chunk)})
	utils.Fatal(err)
	err = db.Client.Dbfs.Close(context.Background(), files.Close{Handle: handle.Handle})
	utils.Fatal(err)
}

func (db *WSclient) DeleteFile(filePathRemote string) {
	err := db.Client.Dbfs.Delete(context.Background(), files.Delete{Path: filePathRemote})
	utils.Fatal(err)
}

/*
* Creates the job with the given parameters. Returns the job id.
 */
func (db *WSclient) CreateJob(jobName string, jobDescription string, jobKey string) (bool, int64) {
	jobList, err := db.Client.Jobs.ListAll(context.Background(), jobs.ListJobsRequest{})
	utils.Fatal(err)

	for _, job := range jobList {
		if job.Settings.Name == jobName {
			return true, job.JobId
		}
	}

	_, err = db.Client.Jobs.Create(context.Background(), jobs.CreateJob{
		Name:        jobName,
		Description: jobDescription,
		Queue: &jobs.QueueSettings{
			Enabled: true,
		},
		Tasks: []jobs.Task{
			{
				TaskKey:           "genericKey",
				ExistingClusterId: db.clusterId,
				NotebookTask: &jobs.NotebookTask{
					NotebookPath: db.notebookPath,
				},
			},
		},
	})
	utils.Fatal(err)
	if err != nil {
		return false, -1
	} else {
		jobList, err := db.Client.Jobs.ListAll(context.Background(), jobs.ListJobsRequest{})
		utils.Fatal(err)
		for _, job := range jobList {
			if job.Settings.Name == jobName {
				return true, job.JobId
			}
		}
	}
	return false, -1
}

func (db *WSclient) RunJob(jobId int64, tableName string, promFilePathRemote string) {
	runner, err := db.Client.Jobs.RunNow(context.Background(), jobs.RunNow{
		JobId:          jobId,
		NotebookParams: map[string]string{"table_name": tableName, "file_name": promFilePathRemote}})
	utils.Fatal(err)
	runner.Poll(10*time.Minute, responseCallback)
	db.DeleteFile(promFilePathRemote)
}

func responseCallback(response *jobs.Run) {
	if response.State.LifeCycleState.String() == "TERMINATED" {
		fmt.Println()
		fmt.Println(response.State.LifeCycleState)
		return
	}
	fmt.Print(response.State.LifeCycleState + "\r")
}

func clusterRunningCallback(clusterState *compute.ClusterDetails) {
	fmt.Println("Cluster state: " + clusterState.State)
}
