package httppanic

import (
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/elastic/beats/v7/libbeat/logp"
)

var ch = make(chan struct{}, 1)

func StartServer() {
	port := fmt.Sprintf(":424%d", rand.Intn(10))
	go func() {
		s := &http.Server{
			Addr: port,
			Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.Method {
				case http.MethodGet:
					fmt.Print("logging to stdout")
					fmt.Fprintf(os.Stderr, "logging to os.Stderr")
					fmt.Fprintln(w, "logged to stdout and os.StdErr")
				case http.MethodPost:
				case "PANIC":
					fmt.Fprintln(w, "HTTP PANIC server called: seeding signal to panic")
					fmt.Println("logging to stdout before PANIC panic!")
					time.Sleep(50 * time.Millisecond)
					ch <- struct{}{}
				}
			}),
		}
		fmt.Printf("starting HTTP panic server on port %s\n", port)
		logp.L().Infof("starting HTTP panic server on port %s\n", port)
		if err := s.ListenAndServe(); err != nil {
			logp.L().Error(fmt.Errorf("panic http server error: %w", err))
		}
	}()
}

func PanicCh() chan struct{} {
	return ch
}
