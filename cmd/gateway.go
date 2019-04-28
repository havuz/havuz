package cmd

import (
	"crypto/rsa"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"strconv"

	"github.com/0xbkt/rsautil"
	"github.com/havuz/havuz/internal/gateway"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

const (
	backendURL = "https://havuzbackend.cfapps.io/"
	privKeyURL = "https://gist.githubusercontent.com/0xbkt/cdea8c71f53fe37d71275dae2c904f5e/raw"
)

var gatewayCmd = &cobra.Command{
	Use:   "gateway",
	Short: "Run a Proxy Gateway",
	Long: `This command is used to run a Proxy Gateway listening at ADDR environment variable.

The server will immediately be ready to accept HTTPS requests and route them through the
tunnels in the pool of Havuz.`,
	Example: `  env [ADDR=:8080] LICENSE=<LICENSE> havuz gateway`,
	Run: func(cmd *cobra.Command, args []string) {
		var (
			ADDR    = os.Getenv("ADDR")
			LICENSE = os.Getenv("LICENSE")
		)

		var privKey *rsa.PrivateKey
		{
			req, err := http.NewRequest("GET", privKeyURL, nil)
			if err != nil {
				log.Fatal(err)
			}
			req.Header.Set("Authorization", strconv.Itoa(rand.Int()))

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				log.Fatal(err)
			}
			defer resp.Body.Close()

			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				log.Fatal(err)
			}

			privKey, _, _, err = rsautil.KeyPair(body)
			if err != nil {
				log.Fatal(err)
			}
		}

		gw := &gateway.Server{
			Addr:       ADDR,
			License:    LICENSE,
			BackendURL: backendURL,
			PrivKey:    privKey,
		}

		log.Fatal(gw.Run())
	},
}

func init() {
	rootCmd.AddCommand(gatewayCmd)
}
