// Assessment Submission for IT Specialist (APPSW) (Software Developer) 21-CFPB-72-P/75-MP
// Applicant ID: 4712645
// Chosen exercise: https://homework.adhoc.team/noclist/
//
// Requirements:
//   - Install Go: https://golang.org/doc/install
//   - Tested locally on: go1.16.2 darwin/amd64
// To run:
//   - `go run 4712645.go`
// To run against an endpoint other than "http://0.0.0.0:8888", pass the endpoint URL as an argument:
//   - `go run 4712645.go <endpoint_url>`
// To test:
//   - `go test 4712645`
package main

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"os"
	"strings"
	"time"
)

// Default endpoint. To use a different endpoint, edit this value or pass as command-line-argument
var endpoint = "http://0.0.0.0:8888"

const (
	maxRetries     = 2
	backoffSeconds = 1
)

// RetryingClient adds retry/backoff request functionality to http.Client.
type RetryingClient struct {
	client     *http.Client
	MaxRetries int
	Backoff    time.Duration
}

func (RetryingClient) shouldRetry(res *http.Response, err error) bool {
	if err != nil || res.StatusCode != http.StatusOK {
		return true
	}
	return false
}

func (rc *RetryingClient) Do(req *http.Request) (*http.Response, error) {
	retries := 0
	res, err := rc.client.Do(req)
	for retries < rc.MaxRetries && rc.shouldRetry(res, err) {
		backoff := rc.Backoff * time.Duration(math.Pow(2, float64(retries)))
		time.Sleep(backoff)
		res, err = rc.client.Do(req)
		retries++
	}
	return res, err
}

func getToken(client *RetryingClient) (string, error) {

	// We only need header info and HEAD is supported
	req, err := http.NewRequest("HEAD", endpoint+"/auth", nil)
	if err != nil {
		return "", err
	}
	res, err := client.Do(req)
	if err != nil {
		return "", err
	}
	if res.StatusCode != http.StatusOK {
		return "", errors.New("bad response code")
	}
	return res.Header.Get("Badsec-Authentication-Token"), nil
}

func getUsers(client *RetryingClient, token string) (string, error) {
	path := "/users"
	req, err := http.NewRequest("GET", endpoint+path, nil)
	if err != nil {
		return "", err
	}

	// Required checksum is computed as `sha256(<auth_token> + <request path>)`
	sum := sha256.Sum256([]byte(token + path))
	req.Header.Add("X-Request-Checksum", fmt.Sprintf("%x", sum))
	res, err := client.Do(req)
	if err != nil {
		return "", err
	}
	if res.StatusCode != http.StatusOK {
		return "", errors.New("bad response code")
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

func main() {
	if len(os.Args) > 1 {
		endpoint = os.Args[1]
	}
	client := &RetryingClient{
		client: &http.Client{
			Timeout: 3 * time.Second,
		},
		MaxRetries: maxRetries,
		Backoff:    time.Duration(backoffSeconds) * time.Second,
	}

	token, err := getToken(client)
	if err != nil {
		log.Fatal(err)
	}
	bodyStr, err := getUsers(client, token)
	if err != nil {
		log.Fatal(err)
	}

	// Body
	jsonData, err := json.Marshal(strings.Split(strings.TrimSpace(bodyStr), "\n"))
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("%s\n", jsonData)
}
