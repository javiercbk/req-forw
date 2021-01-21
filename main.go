package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
)

func main() {
	var port int
	var targetHost string
	var scheme string
	flag.IntVar(&port, "port", 8080, "the port to run the http server")
	flag.StringVar(&targetHost, "host", "", "the host to forward the request to")
	flag.StringVar(&scheme, "scheme", "http", "the default scheme to use when no scheme is provided")
	flag.Parse()
	if targetHost == "" {
		log.Fatalf("the host param is mandatory\n")
	}
	if port <= 0 {
		log.Fatalf("invalid port %d\n", port)
	}
	_, err := url.Parse(targetHost)
	if err != nil {
		log.Fatalf("error parsing host: %v\n", err)
	}
	httpClient := http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}
	forwardRequestHandler := newRequestForwarder(targetHost, scheme, httpClient)
	http.ListenAndServe(":8080", forwardRequestHandler)
}

func newRequestForwarder(target, defaultScheme string, httpClient http.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		scheme := r.URL.Scheme
		if scheme == "" {
			scheme = defaultScheme
		}
		r.URL.Host = target
		r.URL.Scheme = scheme
		newURL := r.URL.String()
		forwardRequest, err := http.NewRequest(r.Method, newURL, r.Body)
		for headerName, headerValueSlice := range r.Header {
			for _, headerValue := range headerValueSlice {
				forwardRequest.Header.Add(headerName, headerValue)
			}
		}
		if err != nil {
			fmt.Printf("error creating forward request: %v\n", err)
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		res, err := httpClient.Do(forwardRequest)
		if err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		for headerName, headerValueSlice := range res.Header {
			for _, headerValue := range headerValueSlice {
				w.Header().Add(headerName, headerValue)
			}
		}
		if res.Body != nil {
			defer res.Body.Close()
			_, err := io.Copy(w, res.Body)
			if err != nil {
				fmt.Printf("error sending response to client: %v\n", err)
			}
		}
	}
}
