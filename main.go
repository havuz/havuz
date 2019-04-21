package main

import (
	"crypto/tls"
	"math/rand"
	"net/http"
	"time"

	"github.com/certifi/gocertifi"
	"github.com/gregjones/httpcache"
	"github.com/havuz/havuz/cmd"
)

func init() {
	pool, _ := gocertifi.CACerts()

	rand.Seed(time.Now().UnixNano())

	http.DefaultClient.Timeout = 10 * time.Second
	http.DefaultClient.Transport = httpcache.NewMemoryCacheTransport()

	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{
		RootCAs: pool,
	}
}

func main() {
	cmd.Execute()
}
