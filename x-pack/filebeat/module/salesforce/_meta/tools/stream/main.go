package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

var (
	sfURL      = "Salesforce-url"
	sfUser     = "Salesforce-username"
	sfPassword = "Salesforce-password"
	sfKey      = "Salesforce-consumerKey"
	sfSecret   = "Salesforce-secret"
)

func main() {
	b := Bayeux{}
	creds := GetSalesforceCredentials()
	c := b.TopicToChannel(creds, "/event/LoginEventStream")
	for {
		select {
		case e := <-c:
			fmt.Printf("TriggerEvent Received: %+v\n", e)
		}
	}
}

// TriggerEvent describes an event received from Bayeaux Endpoint
type TriggerEvent struct {
	ClientID string `json:"clientId"`
	Data     struct {
		Event struct {
			CreatedDate time.Time `json:"createdDate"`
			ReplayID    int       `json:"replayId"`
			Type        string    `json:"type"`
		} `json:"event"`
		Object json.RawMessage `json:"sobject"`
	} `json:"data,omitempty"`
	Channel    string `json:"channel"`
	Successful bool   `json:"successful,omitempty"`
}

func (t TriggerEvent) topic() string {
	s := strings.Replace(t.Channel, "/topic/", "", 1)
	return s
}

// Status is the state of success and subscribed channels
type Status struct {
	connected bool
	clientID  string
	channels  []string
}

type BayeuxHandshake []struct {
	Ext struct {
		Replay bool `json:"replay"`
	} `json:"ext"`
	MinimumVersion           string   `json:"minimumVersion"`
	ClientID                 string   `json:"clientId"`
	SupportedConnectionTypes []string `json:"supportedConnectionTypes"`
	Channel                  string   `json:"channel"`
	Version                  string   `json:"version"`
	Successful               bool     `json:"successful"`
}

type Subscription struct {
	ClientID     string `json:"clientId"`
	Channel      string `json:"channel"`
	Subscription string `json:"subscription"`
	Successful   bool   `json:"successful"`
}

type Credentials struct {
	AccessToken string `json:"access_token"`
	InstanceURL string `json:"instance_url"`
	IssuedAt    int
	ID          string
	TokenType   string `json:"token_type"`
	Signature   string
}

func (c Credentials) bayeuxUrl() string {
	return c.InstanceURL + "/cometd/38.0"
}

type clientIDAndCookies struct {
	clientID string
	cookies  []*http.Cookie
}

// Bayeux struct allow for centralized storage of creds, ids, and cookies
type Bayeux struct {
	creds Credentials
	id    clientIDAndCookies
}

var wg sync.WaitGroup
var logger = log.New(os.Stdout, "", log.Ldate|log.Ltime|log.Lmicroseconds|log.Lshortfile)
var status = Status{false, "", []string{}}

// Call is the base function for making bayeux requests
func (b *Bayeux) call(body string, route string) (resp *http.Response, e error) {
	var jsonStr = []byte(body)
	req, err := http.NewRequest("POST", route, bytes.NewBuffer(jsonStr))
	if err != nil {
		logger.Fatalf("Bad Call request: %s", err)
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", b.creds.AccessToken))
	// Per Stackexchange comment, passing back cookies is required though undocumented in Salesforce API
	// We were unable to get process working without passing cookies back to SF server.
	// SF Reference: https://developer.salesforce.com/docs/atlas.en-us.api_streaming.meta/api_streaming/intro_client_specs.htm
	for _, cookie := range b.id.cookies {
		req.AddCookie(cookie)
	}

	//logger.Printf("REQUEST: %#v", req)
	client := &http.Client{}
	resp, err = client.Do(req)
	if err == io.EOF {
		// Right way to handle EOF?
		logger.Printf("Bad bayeuxCall io.EOF: %s\n", err)
		logger.Printf("Bad bayeuxCall Response: %+v\n", resp)
	} else if err != nil {
		e = errors.New(fmt.Sprintf("Unknown error: %s", err))
		logger.Printf("Bad unrecoverable Call: %s", err)
	}
	return resp, e
}

func (b *Bayeux) getClientID() error {
	handshake := `{"channel": "/meta/handshake", "supportedConnectionTypes": ["long-polling"], "version": "1.0"}`
	//var id clientIDAndCookies
	// Stub out clientIDAndCookies for first bayeuxCall
	resp, err := b.call(handshake, b.creds.bayeuxUrl())
	if err != nil {
		logger.Fatalf("Cannot get client id %s", err)
	}
	defer resp.Body.Close()

	decoder := json.NewDecoder(resp.Body)
	var h BayeuxHandshake
	if err := decoder.Decode(&h); err == io.EOF {
		logger.Fatal(err)
	} else if err != nil {
		logger.Fatal(err)
	}
	creds := clientIDAndCookies{h[0].ClientID, resp.Cookies()}
	b.id = creds
	return nil
}

// ReplayAll replay for past 24 hrs
const ReplayAll = -2

// ReplayNone start playing events at current moment
const ReplayNone = -1

// Replay accepts the following values
// Value
// -2: replay all events from past 24 hrs
// -1: start at current
// >= 0: start from this event number
type Replay struct {
	Value int
}

func (b *Bayeux) subscribe(topic string, replay Replay) Subscription {
	handshake := fmt.Sprintf(`{
								"channel": "/meta/subscribe",
								"subscription": "%s",
								"clientId": "%s",
								"ext": {
									"replay": {"%s": "-2"}
									}
								}`, topic, b.id.clientID, topic)
	resp, err := b.call(handshake, b.creds.bayeuxUrl())
	if err != nil {
		logger.Fatalf("Cannot subscribe %s", err)
	}
	logger.Println(resp)

	defer resp.Body.Close()
	if os.Getenv("DEBUG") != "" {
		logger.Printf("Response: %+v", resp)
		// // Read the content
		var b []byte
		if resp.Body != nil {
			b, _ = ioutil.ReadAll(resp.Body)
		}
		// Restore the io.ReadCloser to its original state
		resp.Body = ioutil.NopCloser(bytes.NewBuffer(b))
		// Use the content
		s := string(b)
		logger.Printf("Response Body: %s", s)
	}

	if resp.StatusCode > 299 {
		logger.Fatalf("Received non 2XX response: HTTP_CODE %d", resp.StatusCode)
	}
	decoder := json.NewDecoder(resp.Body)
	var h []Subscription
	if err := decoder.Decode(&h); err == io.EOF {
		logger.Fatal(err)
	} else if err != nil {
		logger.Fatal(err)
	}
	sub := h[0]
	status.connected = sub.Successful
	status.clientID = sub.ClientID
	status.channels = append(status.channels, topic)
	logger.Printf("Established connection(s): %+v", status)
	return sub
}

func (b *Bayeux) connect() chan TriggerEvent {
	out := make(chan TriggerEvent)
	go func() {
		// TODO: add stop chan to bring this thing to halt
		for {
			postBody := fmt.Sprintf(`{"channel": "/meta/connect", "connectionType": "long-polling", "clientId": "%s"} `, b.id.clientID)
			resp, err := b.call(postBody, b.creds.bayeuxUrl())
			if err != nil {
				logger.Printf("Cannot connect to bayeux %s", err)
				logger.Println("Trying again...")
			} else {
				// defer resp.Body.Close()
				if os.Getenv("DEBUG") != "" {
					fmt.Println(resp.Body)
					// // Read the content
					var b []byte
					if resp.Body != nil {
						b, _ = ioutil.ReadAll(resp.Body)
					}
					// Restore the io.ReadCloser to its original state
					resp.Body = ioutil.NopCloser(bytes.NewBuffer(b))
					// Use the content
					s := string(b)
					logger.Printf("Response Body: %s", s)
				}
				var x []TriggerEvent
				decoder := json.NewDecoder(resp.Body)
				if err := decoder.Decode(&x); err != nil && err != io.EOF {
					logger.Fatal(err)
				}
				for _, e := range x {
					out <- e
				}
			}
		}
	}()
	return out
}

func GetSalesforceCredentials() Credentials {
	route := "https://login.salesforce.com/services/oauth2/token"
	params := url.Values{"grant_type": {"password"},
		"client_id":     {sfKey},
		"client_secret": {sfSecret},
		"username":      {sfUser},
		"password":      {sfPassword}}
	res, err := http.PostForm(route, params)
	if err != nil {
		logger.Fatal(err)
	}
	decoder := json.NewDecoder(res.Body)
	var creds Credentials
	if err := decoder.Decode(&creds); err == io.EOF {
		logger.Fatal(err)
	} else if err != nil {
		logger.Fatal(err)
	} else if creds.AccessToken == "" {
		logger.Fatalf("Unable to fetch access token. Check credentials in environmental variables")
	}
	return creds
}

func mustGetEnv(s string) string {
	r := os.Getenv(s)
	if r == "" {
		panic(fmt.Sprintf("Could not fetch key %s", s))
	}
	return r
}

func (b *Bayeux) TopicToChannel(creds Credentials, topic string) chan TriggerEvent {
	b.creds = creds
	err := b.getClientID()
	if err != nil {
		log.Fatal("Unable to get bayeux ClientId")
	}
	r := Replay{ReplayAll}
	b.subscribe(topic, r)
	c := b.connect()
	wg.Add(1)
	return c
}
