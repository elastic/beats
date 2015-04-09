// Copyright 2013 Matthew Baird
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//     http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package api

import (
        "bytes"
        "encoding/json"
        "fmt"
        hostpool "github.com/bitly/go-hostpool"
        "io"
        "io/ioutil"
        "log"
        "net/http"
        "net/url"
        "runtime"
        "strconv"
        "strings"
        "sync"
        "time"
)

type Request struct {
        *http.Request
        hostResponse hostpool.HostPoolResponse
}

const (
        Version         = "0.0.2"
        DefaultProtocol = "http"
        DefaultDomain   = "localhost"
        DefaultPort     = "9200"
        // A decay duration of zero results in the default behaviour
        DefaultDecayDuration = 0
)

var (
        _ = log.Ldate
        // Maintain these for backwards compatibility
        Protocol       string    = DefaultProtocol
        Domain         string    = DefaultDomain
        ClusterDomains [1]string = [1]string{DefaultDomain}
        Port           string    = DefaultPort
        Username       string
        Password       string
        BasePath       string
        // Store a slice of hosts in a hostpool
        Hosts []string
        hp    hostpool.HostPool
        once  sync.Once

        // To compute the weighting scores, we perform a weighted average of recent response times,
        // over the course of `DecayDuration`. DecayDuration may be set to 0 to use the default
        // value of 5 minutes. The EpsilonValueCalculator uses this to calculate a score
        // from the weighted average response time.
        DecayDuration time.Duration = time.Duration(DefaultDecayDuration * time.Second)
)

func ElasticSearchRequest(method, path, query string) (*Request, error) {
        // Setup the hostpool on our first run
        once.Do(initializeHostPool)

        // Get a host from the host pool
        hr := hp.Get()

        // Get the final host and port
        host, portNum := splitHostnamePartsFromHost(hr.Host(), Port)

        // Prepend base path if set
        if BasePath != "" {
                path = fmt.Sprintf("%s%s", BasePath, path)
        }

        // Build request
        var uri string
        // If query parameters are provided, the add them to the URL,
        // otherwise, leave them out
        if len(query) > 0 {
                uri = fmt.Sprintf("%s://%s:%s%s?%s", Protocol, host, portNum, path, query)
        } else {
                uri = fmt.Sprintf("%s://%s:%s%s", Protocol, host, portNum, path)
        }
        req, err := http.NewRequest(method, uri, nil)
        if err != nil {
                return nil, err
        }
        req.Header.Add("Accept", "application/json")
        req.Header.Add("User-Agent", "elasticSearch/"+Version+" ("+runtime.GOOS+"-"+runtime.GOARCH+")")

        if Username != "" || Password != "" {
                req.SetBasicAuth(Username, Password)
        }

        newRequest := &Request{
                Request:      req,
                hostResponse: hr,
        }
        return newRequest, nil
}

func SetHosts(newhosts []string) {

        // Store the new host list
        Hosts = newhosts

        // Reinitialise the host pool
        // Pretty naive as this will nuke the current hostpool, and therefore reset any scoring
        initializeHostPool()

}

func (r *Request) SetBodyJson(data interface{}) error {
        body, err := json.Marshal(data)
        if err != nil {
                return err
        }
        r.SetBodyBytes(body)
        r.Header.Set("Content-Type", "application/json")
        return nil
}

func (r *Request) SetBodyString(body string) {
        r.SetBody(strings.NewReader(body))
}

func (r *Request) SetBodyBytes(body []byte) {
        r.SetBody(bytes.NewReader(body))
}

func (r *Request) SetBody(body io.Reader) {
        rc, ok := body.(io.ReadCloser)
        if !ok && body != nil {
                rc = ioutil.NopCloser(body)
        }
        r.Body = rc
        if body != nil {
                switch v := body.(type) {
                case *strings.Reader:
                        r.ContentLength = int64(v.Len())
                case *bytes.Reader:
                        r.ContentLength = int64(v.Len())
                }
        }
}

func (r *Request) Do(v interface{}) (int, []byte, error) {
        response, bodyBytes, err := r.DoResponse(v)
        if err != nil {
                return -1, nil, err
        }
        return response.StatusCode, bodyBytes, err
}

func (r *Request) DoResponse(v interface{}) (*http.Response, []byte, error) {
        res, err := http.DefaultClient.Do(r.Request)
        // Inform the HostPool of what happened to the request and allow it to update
        r.hostResponse.Mark(err)
        if err != nil {
                return nil, nil, err
        }

        defer res.Body.Close()
        bodyBytes, err := ioutil.ReadAll(res.Body)

        if err != nil {
                return nil, nil, err
        }

        if res.StatusCode > 304 && v != nil {
                jsonErr := json.Unmarshal(bodyBytes, v)
                if jsonErr != nil {
                        return nil, nil, jsonErr
                }
        }
        return res, bodyBytes, err
}

func QueryString(args map[string]interface{}) (s string, err error) {
        vals := url.Values{}
        for key, val := range args {
                switch v := val.(type) {
                case string:
                        vals.Add(key, v)
                case bool:
                        vals.Add(key, strconv.FormatBool(v))
                case int, int32, int64:
                        vals.Add(key, strconv.Itoa(v.(int)))
                case float32, float64:
                        vals.Add(key, strconv.FormatFloat(v.(float64), 'f', -1, 64))
                case []string:
                        vals.Add(key, strings.Join(v, ","))
                default:
                        err = fmt.Errorf("Could not format URL argument: %s", key)
                        return
                }
        }
        s = vals.Encode()
        return
}

// Set up the host pool to be used
func initializeHostPool() {

        // If no hosts are set, fallback to defaults
        if len(Hosts) == 0 {
                Hosts = append(Hosts, fmt.Sprintf("%s:%s", Domain, Port))
        }

        // Epsilon Greedy is an algorithm that allows HostPool not only to track failure state,
        // but also to learn about "better" options in terms of speed, and to pick from available hosts
        // based on how well they perform. This gives a weighted request rate to better
        // performing hosts, while still distributing requests to all hosts (proportionate to their performance).
        // The interface is the same as the standard HostPool, but be sure to mark the HostResponse immediately
        // after executing the request to the host, as that will stop the implicitly running request timer.
        //
        // A good overview of Epsilon Greedy is here http://stevehanov.ca/blog/index.php?id=132

        hp = hostpool.NewEpsilonGreedy(Hosts, DecayDuration, &hostpool.LinearEpsilonValueCalculator{})

}

// Split apart the hostname on colon
// Return the host and a default port if there is no separator
func splitHostnamePartsFromHost(fullHost string, defaultPortNum string) (string, string) {

        h := strings.Split(fullHost, ":")

        if len(h) == 2 {
                return h[0], h[1]
        }

        return h[0], defaultPortNum
}
