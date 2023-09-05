package common

import (
	"crypto/tls"
	"io"
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/WangYihang/Subdomain-Crawler/pkg/model"
)

var (
	// HTTPClient is the http client
	client *http.Client
	once   sync.Once
)

func GetHTTPClient() *http.Client {
	once.Do(func() {
		timeout := model.Opts.Timeout
		transport := http.Transport{
			Dial: (&net.Dialer{
				// Modify the time to wait for a connection to establish
				Timeout:   time.Duration(timeout) * time.Second,
				KeepAlive: time.Duration(timeout) * time.Second,
			}).Dial,
			TLSHandshakeTimeout:   time.Duration(timeout) * time.Second,
			IdleConnTimeout:       time.Duration(timeout) * time.Second,
			ResponseHeaderTimeout: time.Duration(timeout) * time.Second,
			ExpectContinueTimeout: time.Duration(timeout) * time.Second,
			DisableKeepAlives:     true,
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		}
		client = &http.Client{
			Transport: &transport,
			Timeout:   time.Duration(timeout) * time.Second,
		}
		log.SetOutput(io.Discard)
	})
	return client
}
