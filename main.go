package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"syscall"
	"time"
)

var (
	// Define command-line flags
	versionFlag = flag.Bool("v", false, "Print the program name and version")
	sourceFlag  = flag.String("s", "", "Set the source path")
	jobsFlag    = flag.Int("j", 0, "Set the quantity of jobs")
	workersFlag = flag.Int("c", 0, "Set the quantity of concurrent connections")
)

var (
	programName    = "Bearer"
	programVersion = "1.0.0"
)

type Engine struct {
	inbox *chan Job
	jobs  chan Job
}

func (engine Engine) ProcessJobs(waitGroup *sync.WaitGroup) {
	defer waitGroup.Done()
	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:        10,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     30 * time.Second,
	}
	httpClient := &http.Client{Transport: transport}
	for job := range engine.jobs {
		job.Process(httpClient)
	}
}

func (engine Engine) ScheduleNextJob(ctx context.Context) {
	for {
		select {
		case job := <-*engine.inbox:
			engine.jobs <- job
		case <-ctx.Done():
			close(engine.jobs)
			return
		}
	}
}

type Job struct {
	delay   float64
	url     string
	method  string
	headers map[string]interface{}
	payload map[string]interface{}
}

func (job Job) Process(httpClient *http.Client) {
	if job.delay != 0 {
		time.Sleep(time.Duration(job.delay) * time.Millisecond)
	}
	var req *http.Request
	switch job.method {
	case "POST":
		payload, err := json.Marshal(job.payload)
		if err != nil {
			log.Println(err)
			return
		}
		r, _ := http.NewRequest(job.method, job.url, bytes.NewBuffer(payload))
		req = r
	case "PUT":
		payload, err := json.Marshal(job.payload)
		if err != nil {
			log.Println(err)
			return
		}
		r, _ := http.NewRequest(job.method, job.url, bytes.NewBuffer(payload))
		req = r
	default:
		req, _ = http.NewRequest(job.method, job.url, nil)
	}

	for key, value := range job.headers {
		req.Header.Add(key, value.(string))
	}

	log.Printf("Sending: %s %s \n", job.method, job.url)
	resp, err := httpClient.Do(req)

	if err != nil {
		log.Printf("Error during HTTP request: %v", err)
		httpClient.CloseIdleConnections()
		return
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error reading response body: %v", err)
	}

	log.Printf("Status: %s Body: %s", resp.Status, string(body))
	httpClient.CloseIdleConnections()
}

type Producer struct {
	inbox *chan Job
}

type Headers struct {
	Key   string
	Value string
}

type LoadSource struct {
	Url     string  `json:"url"`
	Method  string  `json:"method"`
	Headers Headers `json:"headers"`
	Payload string  `json:"payload"`
	Delay   float64 `json:"delay"`
}

func loadJobs(path string, inbox chan Job) {
	files, err := ioutil.ReadDir(path)
	if err != nil {
		log.Fatal(err)
	}

	for _, file := range files {
		loadJob(file, path, inbox)
	}
}

func makeJob(jobDefinition map[string]interface{}) Job {
	job := Job{}
	for key, value := range jobDefinition {
		switch key {
		case "delay":
			job.delay = value.(float64)
		case "url":
			job.url = value.(string)
		case "method":
			job.method = value.(string)
		case "headers":
			job.headers = value.(map[string]interface{})
		case "payload":
			// job.payload = value.(string)
			payloadStr := value.(string)
			payload := make(map[string]interface{})
			if err := json.Unmarshal([]byte(payloadStr), &payload); err != nil {
				log.Fatalf("error parsing payload: %v", err)
			}
			job.payload = payload
		}
	}
	return job
}

func loadJob(file fs.FileInfo, path string, inbox chan Job) {
	filename := file.Name()
	jsonFile, err := os.Open(path + "/" + filename)
	if err != nil {
		log.Println(err)
	}

	defer jsonFile.Close()
	readBytes, err := ioutil.ReadAll(jsonFile)
	if err != nil {
		log.Println(err)
	}

	if !json.Valid(readBytes) {
		log.Println("Invalid JSON")
		return
	}

	var jobDefinition map[string]interface{}
	_ = json.Unmarshal(readBytes, &jobDefinition)
	if err != nil {
		log.Println(err)
		return
	}
	job := makeJob(jobDefinition)
	inbox <- job
}

func (producer Producer) ProduceJobs(quantity *int, source_path string) {
	count := *quantity
	for i := 0; i < count; i++ {
		loadJobs(source_path, *producer.inbox)
	}
}

func checkArgs() int {
	if *versionFlag {
		fmt.Printf("%s version %s\n", programName, programVersion)
		return 1
	}
	if *workersFlag == 0 {
		fmt.Printf("The concurrency value must be set")
		return 1
	}

	if *jobsFlag == 0 {
		fmt.Printf("The concurrency value must be set")
		return 1
	}
	if *sourceFlag == "" {
		fmt.Printf("The source path value must be set")
		return 1
	}

	return 0
}

func main() {
	flag.Parse()
	fmt.Printf("workersFlag %d\n", *workersFlag)
	fmt.Printf("jobsFlag %d\n", *jobsFlag)
	fmt.Printf("sourceFlag %s\n", *sourceFlag)
	if checkArgs() != 0 {
		return
	}

	inbox := make(chan Job, *jobsFlag)
	quantityOfWorkers := workersFlag
	runtime.GOMAXPROCS(runtime.NumCPU())
	producer := Producer{&inbox}
	engine := Engine{&inbox, make(chan Job, *quantityOfWorkers)}
	go producer.ProduceJobs(jobsFlag, *sourceFlag)
	ctx, cancelFunc := context.WithCancel(context.Background())
	go engine.ScheduleNextJob(ctx)
	waitGroup := &sync.WaitGroup{}
	waitGroup.Add(*quantityOfWorkers)
	for i := 0; i < *quantityOfWorkers; i++ {
		go engine.ProcessJobs(waitGroup)
	}
	terminalChannel := make(chan os.Signal)
	signal.Notify(terminalChannel, syscall.SIGINT, syscall.SIGTERM)
	<-terminalChannel
	cancelFunc()
	waitGroup.Wait()
}
