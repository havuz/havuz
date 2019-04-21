package gateway

import (
	"crypto/rsa"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/fatih/structs"
	"github.com/gregjones/httpcache"
	"github.com/havuz/types"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// Server is a Proxy Gateway that tunnels the requests through
// the tunnels in the pool of Havuz.
type Server struct {
	Addr       string          // TCP address to listen on, ":8080" if empty.
	License    string          // License code of the client.
	BackendURL string          // BackendURL is the endpoint of upstream services.
	Logger     *logrus.Logger  // Custom logger for internal logging, logrus.New() if empty.
	PrivKey    *rsa.PrivateKey // Private key that will be used to authenticate with tunnels.
}

// Run prepares and fires the Server engine.
func (s *Server) Run() (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = errors.Errorf("%v", e)
		}
	}()

	if err = s.init(); err != nil {
		return
	}

	s.Logger.Info("server has been initialized...")

	resp, user, err := s.doAuth()
	if err != nil {
		return
	}
	defer resp.Body.Close()

	// making sure that the initial response is saved in cache
	io.Copy(ioutil.Discard, resp.Body)

	s.Logger.WithFields(map[string]interface{}{
		"Addr": s.Addr,
	}).Infoln("proxy gateway is now listening")
	return http.ListenAndServe(s.Addr, s.proxyHandler(user))
}

func (s *Server) init() error {
	if s.Addr == "" {
		s.Addr = ":8080"
	}

	if s.License == "" {
		return errors.New("license is required")
	}

	if s.BackendURL == "" {
		return errors.New("backendURL is required")
	}

	if s.PrivKey == nil {
		return errors.New("privKey is required")
	}

	if s.Logger == nil {
		s.Logger = logrus.New()
	}

	return nil
}

func (s *Server) doAuth() (resp *http.Response, user *types.User, err error) {
	log := s.Logger

	log.Info("auth flow has begun. just hang on...")

try:
	resp, err = s.doRoundtripToBackend()
	if err != nil {
		if err.(*url.Error).Timeout() {
			log.Warn(errors.WithMessage(err, "doAuth"))
			goto try
		}
		return
	}

	isCached, _ := strconv.ParseBool(resp.Header.Get(httpcache.XFromCache))
	log.WithFields(logrus.Fields{
		"Status": resp.Status,
		"Cached": isCached,
	}).Infoln("backend replied")

	user = new(types.User)

	// unmarshal X-User header
	{
		dec := json.NewDecoder(strings.NewReader(resp.Header.Get("X-User")))
		dec.UseNumber()

		if err := dec.Decode(user); err == nil {
			log.WithFields(structs.Map(user)).Infoln("a user was returned")
		}
	}

	// handle when the license is not authorized
	if resp.StatusCode == http.StatusUnauthorized {
		if structs.IsZero(*user) {
			err = errors.New("no such user was found by this license key")
		} else {
			err = errors.New("given license key was not granted access to backend. see user details above")
		}
		return
	}

	// TODO(0xbkt): perhaps improve this?
	if resp.StatusCode != http.StatusOK {
		resp.Write(os.Stdout)
		err = errors.New("backend did not reply with 200")
		return
	}

	log.WithField("User", user.ID).Infoln("user was successfully authenticated to backend")
	return
}

func (s *Server) doRoundtripToBackend() (resp *http.Response, err error) {
	req, err := http.NewRequest("GET", s.BackendURL, nil)
	if err != nil {
		return
	}
	req.SetBasicAuth("_", s.License)

	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		return
	}

	return
}

/*
func (s *Server) redactedLicense() string {
	licenseLen := len(s.License)

	first := math.Floor(float64(licenseLen / 2))
	second := int(math.Ceil(first / 2))

	if second > 10 {
		second = 10
	}

	if second == 2 {
		second = 0
	}

	return fmt.Sprintf("%v%v%v", s.License[:second], strings.Repeat("*", second+5), s.License[licenseLen-second:])
}
*/
