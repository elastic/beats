package stackdriver

import (
    monitoring "cloud.google.com/go/monitoring/apiv3"
    monitoringpb "google.golang.org/genproto/googleapis/monitoring/v3"
    "context"
    "encoding/json"
    "fmt"
    "google.golang.org/api/iterator"
    "sync"
)

func requestMetrics(client *monitoring.MetricClient, req monitoringpb.ListTimeSeriesRequest, metric string, errs chan<- error, wg *sync.WaitGroup) {
    defer wg.Done()

    req.Filter = fmt.Sprintf(`metric.type="%s"`, metric)

    it := client.ListTimeSeries(context.Background(), &req)
    for {
        resp, err := it.Next()
        if err == iterator.Done {
            break
        }
        if err != nil {
            err = fmt.Errorf("could not read time series value: %v", err)
            errs <- err
        }

        byt, err := json.MarshalIndent(resp, "", "  ")
        if err != nil {
            errs <- err
            return
        }

        //TODO
        fmt.Println(string(byt))
    }
}
