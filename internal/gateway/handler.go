package gateway

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"reflect"
	"strings"
	"sync"

	"github.com/havuz/types"
	"github.com/pkg/errors"
	"github.com/vmihailenco/msgpack"
	"golang.org/x/crypto/ssh"
	"golang.org/x/sync/semaphore"
)

func (s *Server) proxyHandler(u *types.User) http.HandlerFunc {
	sem := semaphore.NewWeighted(int64(u.SimultaneityCap))

	signer, err := ssh.NewSignerFromKey(s.PrivKey)
	if err != nil {
		panic(err)
	}

	var dial = func(network, addr string) (conn net.Conn, err error) {
		var tunMap map[string]types.Tunnel

	pollStuff:
		resp, _, err := s.doAuth()
		if err != nil {
			s.Logger.Fatal(err)
		}
		defer resp.Body.Close()

		if err = msgpack.NewDecoder(resp.Body).UseJSONTag(true).Decode(&tunMap); err != nil {
			err = errors.Wrap(err, "msgpack")
			return
		}

		// picking random tunnels
		{
			var tunMapKeys = reflect.ValueOf(tunMap).MapKeys()

			tmpMap := make(map[string]types.Tunnel)
			for i := 0; i < 5; i++ {
			try:
				randKey := tunMapKeys[rand.Intn(len(tunMapKeys))]

				// check whether the key is already used
				if randKey == (reflect.Value{}) {
					goto try
				}

				tmpMap[randKey.String()] = tunMap[randKey.String()]

				// mark the key as used
				randKey = reflect.Value{}
			}
			tunMap = tmpMap
		}

		var (
			wg sync.WaitGroup

			clientCh    = make(chan *ssh.Client)
			completedCh = make(chan struct{})
		)

		for _, tun := range tunMap {
			tun := tun

			wg.Add(1)
			go func() {
				defer wg.Done()

				cfg := &ssh.ClientConfig{
					User:            tun.SSHUser,
					Auth:            []ssh.AuthMethod{ssh.PublicKeys(signer)},
					HostKeyCallback: ssh.InsecureIgnoreHostKey(),
				}

				client, err := ssh.Dial("tcp", net.JoinHostPort(tun.SSHHost, "80"), cfg)
				if err != nil {
					return
				}

				select {
				case clientCh <- client:
				default:
				}
			}()
		}

		go func() {
			wg.Wait()
			close(completedCh)
		}()

		var client *ssh.Client
		select {
		// we failed to receive the client and all goroutines
		// finished execution. that's why we should try again
		case <-completedCh:
			goto pollStuff
		case client = <-clientCh:
		}

		return client.Dial(network, addr)
	}

	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if e := recover(); e != nil {
				s.Logger.WithField("Err", e).Error("panic recovered in proxy handler")
				s.Logger.Error(fmt.Sprintf("%+v", e))
			}
		}()

		if !r.URL.IsAbs() && r.URL.Host == "" {
			http.Redirect(w, r, "https://github.com/havuz", http.StatusSeeOther)
			return
		}

		// todo(0xbkt): implement proxy authentication here
		// user, pass, ok := parseBasicAuth(r.Header.Get("Proxy-Authorization"))

		if err := sem.Acquire(context.TODO(), 1); err != nil {
			panic(err)
		}
		defer sem.Release(1)

		if r.Method == "CONNECT" {
			handleCONNECT(w, r, dial)
		} else {
			http.Error(w, "Only CONNECT requests are allowed at the moment.", http.StatusMethodNotAllowed)
		}
	}
}

type dialFunc func(string, string) (net.Conn, error)

func handleCONNECT(w http.ResponseWriter, r *http.Request, dial dialFunc) {
	hij, _ := w.(http.Hijacker)

	clientConn, _, err := hij.Hijack()
	if err != nil {
		panic(err)
	}
	defer clientConn.Close()

	proxyConn, err := dial("tcp", r.Host)
	if err != nil {
		panic(err)
	}
	defer proxyConn.Close()

	_, err = clientConn.Write([]byte("HTTP/1.1 200 OK\r\n\r\n"))
	if err != nil {
		panic(err)
	}

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		io.Copy(clientConn, proxyConn)
		proxyConn.Close()
		clientConn.Close()
	}()

	go func() {
		defer wg.Done()
		io.Copy(proxyConn, clientConn)
		clientConn.Close()
		proxyConn.Close()
	}()

	wg.Wait()
}

// parseBasicAuth parses an HTTP Basic Authentication string.
// "Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==" returns ("Aladdin", "open sesame", true).
func parseBasicAuth(auth string) (username, password string, ok bool) {
	const prefix = "Basic "
	// Case insensitive prefix match. See Issue 22736.
	if len(auth) < len(prefix) || !strings.EqualFold(auth[:len(prefix)], prefix) {
		return
	}
	c, err := base64.StdEncoding.DecodeString(auth[len(prefix):])
	if err != nil {
		return
	}
	cs := string(c)
	s := strings.IndexByte(cs, ':')
	if s < 0 {
		return
	}
	return cs[:s], cs[s+1:], true
}
