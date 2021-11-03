package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"golang.org/x/oauth2"
)

var (
	sfURL      = "Salesforce-url"
	sfUser     = "Salesforce-username"
	sfPassword = "Salesforce-password"
	sfKey      = "Salesforce-consumerKey"
	sfSecret   = "Salesforce-secret"
)

type Credentials struct {
	AccessToken string `json:"access_token"`
	InstanceURL string `json:"instance_url"`
	IssuedAt    int
	ID          string
	TokenType   string `json:"token_type"`
	Signature   string
}

type LogFiles struct {
	TotalSize int `json:"totalSize"`
	Records   []struct {
		EventType  string `json:"EventType"`
		Attributes struct {
			Type string `json:"type"`
			URL  string `json:"url"`
		} `json:"attributes"`
	} `json:"records"`
}

func newClient(sfURL string, sfUser string, sfPassword string, sfKey string, sfSecret string) (*oauth2.Token, error) {
	conf := &oauth2.Config{
		ClientID:     sfKey,
		ClientSecret: sfSecret,
		Endpoint: oauth2.Endpoint{
			TokenURL:  "https://login.salesforce.com/services/oauth2/token",
			AuthStyle: 1,
		},
	}
	token, err := conf.PasswordCredentialsToken(context.Background(), sfUser, sfPassword)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	return token, nil
}

func getLogFiles(token *oauth2.Token) ([]string, error) {
	baseUrl := token.Extra("instance_url").(string) + "/services/data/v52.0/query?q=SELECT+Id+,+Interval+,+EventType+,+LogFile+,+LogDate+,+LogFileLength+FROM+EventLogFile+WHERE+Interval+=+'Hourly'+AND+EventType+=+'Login'"
	var bearer = "Bearer " + token.Extra("access_token").(string)

	req, err := http.NewRequest("GET", baseUrl, nil)
	if err != nil {
		fmt.Print(err.Error())
		return nil, err
	}
	req.Header.Add("Authorization", bearer)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Println("Error on response.\n[ERROR] -", err)
		return nil, err
	}
	defer resp.Body.Close()

	var data LogFiles

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("Error while reading the response bytes:", err)
		return nil, err
	}
	json.Unmarshal(body, &data)

	var urls []string
	for _, record := range data.Records {
		urls = append(urls, record.Attributes.URL)
	}

	return urls, nil
}

func getLogData(urls []string, sfURL string, token *oauth2.Token) ([]string, error) {
	var data []string
	for _, url := range urls {
		baseUrl := "https://" + sfURL
		// "/services/data/v%s/sobjects/ContentVersion/%s/VersionData"
		baseUrl = baseUrl + url + "/LogFile"
		response, err := getData(baseUrl, token)
		if err != nil {
			fmt.Println(err)
			return nil, err
		}
		data = append(data, response)
	}

	return data, nil
}

func getData(baseUrl string, token *oauth2.Token) (string, error) {
	var bearer = "Bearer " + token.Extra("access_token").(string)

	req, err := http.NewRequest("GET", baseUrl, nil)
	if err != nil {
		fmt.Print(err.Error())
		return "", err
	}
	req.Header.Add("Authorization", bearer)
	req.Header.Add("Content-Type", "application/json; charset=UTF-8")
	req.Header.Add("Accept", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Println("Error on response.\n[ERROR] -", err)
		return "", err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("Error while reading the response bytes:", err)
		return "", err
	}

	return string(body), nil
}

func main() {
	token, err := newClient(sfURL, sfUser, sfPassword, sfKey, sfSecret)
	if err != nil {
		fmt.Println(err)
	}
	urls, err := getLogFiles(token)
	if err != nil {
		fmt.Println(err)
	}
	data, err := getLogData(urls, sfURL, token)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(data)
}
