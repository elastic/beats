// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package mock

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"math/rand"
	"time"
)

const (
	TotalRandomDataSets  = 10000
	ConcurrencyContainer = "concurrency_container"
)

// Generates random Azure blob storage container metadata in XML format
type EnumerationResults struct {
	XMLName         xml.Name `xml:"EnumerationResults"`
	ServiceEndpoint string   `xml:"ServiceEndpoint,attr"`
	ContainerName   string   `xml:"ContainerName,attr"`
	Blobs           []Blob   `xml:"Blobs>Blob"`
	NextMarker      string   `xml:"NextMarker"`
}

type Blob struct {
	Name       string     `xml:"Name"`
	Properties Properties `xml:"Properties"`
	Metadata   string     `xml:"Metadata"`
}

type Properties struct {
	LastModified  string `xml:"Last-Modified"`
	Etag          string `xml:"Etag"`
	ContentLength int    `xml:"Content-Length"`
	ContentType   string `xml:"Content-Type"`
}

func generateMetadata() []byte {
	// Generate random data for x data sets defined by TotalRandomDataSets
	const numDataSets = TotalRandomDataSets
	dataSets := make([]Blob, numDataSets)

	for i := 0; i < numDataSets; i++ {
		dataSets[i] = createRandomBlob(i)
	}

	// Fill in the root XML structure
	xmlData := EnumerationResults{
		ServiceEndpoint: "https://127.0.0.1/",
		ContainerName:   "concurrency_container",
		Blobs:           dataSets,
		NextMarker:      "",
	}

	// Marshal the data into XML format
	xmlBytes, err := xml.MarshalIndent(xmlData, "", "\t")
	if err != nil {
		panic(fmt.Sprintf("Error marshaling data: %v", err))
	}
	return []byte(xml.Header + string(xmlBytes))
}

// Helper function to create a random Blob
func createRandomBlob(i int) Blob {
	rand.New(rand.NewSource(12345))

	return Blob{
		Name: fmt.Sprintf("data_%d.json", i),
		Properties: Properties{
			LastModified: time.Now().Format(time.RFC1123),
			Etag:         fmt.Sprintf("0x%X", rand.Int63()),
			ContentType:  "application/json",
		},
		Metadata: "",
	}
}

// Generate Random Blob data in JSON format
type MyData struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Age         int    `json:"age"`
	Email       string `json:"email"`
	Description string `json:"description"`
}

func generateRandomBlob() []byte {
	const numObjects = 10
	dataObjects := make([]MyData, numObjects)

	for i := 0; i < numObjects; i++ {
		dataObjects[i] = createRandomData()
	}

	jsonBytes, err := json.MarshalIndent(dataObjects, "", "\t")
	if err != nil {
		panic(fmt.Sprintf("Error marshaling data: %v", err))
	}
	return jsonBytes
}

func createRandomData() MyData {
	rand.New(rand.NewSource(12345))

	return MyData{
		ID:          rand.Intn(1000) + 1,
		Name:        getRandomString([]string{"John", "Alice", "Bob", "Eve"}),
		Age:         rand.Intn(80) + 18,
		Email:       getRandomString([]string{"john@example.com", "alice@example.com", "bob@example.com"}),
		Description: getRandomString([]string{"Student", "Engineer", "Artist", "Doctor"}),
	}
}

func getRandomString(options []string) string {
	if len(options) == 0 {
		return ""
	}

	rand.New(rand.NewSource(time.Now().UnixNano()))
	return options[rand.Intn(len(options))]
}
