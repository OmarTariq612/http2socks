package main

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"golang.org/x/net/proxy"
)

type Relayer struct {
	bindAddr    string
	serverAddr  string
	credentials string
}

func NewRelayer(bindAddr, serverAddr, credentials string) *Relayer {
	return &Relayer{bindAddr: bindAddr, serverAddr: serverAddr, credentials: credentials}
}

const (
	green = "\033[92m"
	// blue   = "\033[94m"
	red    = "\033[0;31m"
	orange = "\033[38;5;214m"
	end    = "\033[0m"
)

func (relayer *Relayer) Authenticate(r *http.Request) bool {
	if relayer.credentials == "" {
		return true // no auth
	}
	auth := r.Header.Get("Proxy-Authorization")
	if auth == "" {
		return false // Proxy-Authorization is not set
	}
	params := strings.Split(auth, " ")
	if params[0] != "Basic" {
		return false // Proxy-Authorization scheme is not "Basic"
	}
	if providedCred, err := base64.StdEncoding.DecodeString(params[1]); err != nil || string(providedCred) != relayer.credentials {
		return false // invalid credentials
	}
	return true // valid credentials
}

func (relayer *Relayer) ListenAndServe() error {
	socksDialer, err := proxy.SOCKS5("tcp", relayer.serverAddr, nil, proxy.Direct)
	if err != nil {
		return fmt.Errorf("could not create a socks dialer, %v", err)
	}

	log.Println("Serving on", relayer.bindAddr)
	log.Println("the provided socks server address is", relayer.serverAddr)

	return http.ListenAndServe(relayer.bindAddr, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		if !relayer.Authenticate(r) {
			w.Header().Add("Proxy-Authenticate", "Basic")
			http.Error(w, "Proxy Authentication Required", http.StatusProxyAuthRequired)
			io.Copy(io.Discard, r.Body)
			r.Body.Close()
			return
		}

		switch r.Method {
		default:
			log.Printf("%s[%s]%s: %s - from %s\n", orange, r.Method, end, r.Host, r.RemoteAddr)
			if _, _, err = net.SplitHostPort(r.Host); err != nil {
				r.Host = fmt.Sprintf("%s:80", r.Host)
			}

			serverConn, err := socksDialer.Dial("tcp", r.Host)
			if err != nil {
				http.Error(w, err.Error(), http.StatusServiceUnavailable)
				io.Copy(io.Discard, r.Body)
				r.Body.Close()
				return
			}
			defer serverConn.Close()

			// r.Header.Del("Proxy-Connection")
			// r.Header.Set("Connection", "keep-alive")

			err = r.Write(serverConn)
			if err != nil {
				http.Error(w, err.Error(), http.StatusServiceUnavailable)
				return
			}

			resp, err := http.ReadResponse(bufio.NewReader(serverConn), r)
			if err != nil {
				http.Error(w, err.Error(), http.StatusServiceUnavailable)
				return
			}
			defer resp.Body.Close()

			// //copy headers
			// for key, values := range resp.Header {
			// 	for _, value := range values {
			// 		w.Header().Add(key, value)
			// 	}
			// }
			// w.WriteHeader(resp.StatusCode)

			// // copy body
			// _, err = io.Copy(w, resp.Body)
			// if err != nil {
			// 	log.Println(err)
			// }

			// w.Write(nil)

			hj, ok := w.(http.Hijacker)
			if !ok {
				log.Println("it is not hijacker")
				return
			}

			clientConn, _, err := hj.Hijack()
			if err != nil {
				log.Println("can not hijack")
				return
			}
			defer clientConn.Close()
			resp.Write(clientConn)

			for {
				req, err := http.ReadRequest(bufio.NewReader(clientConn))
				if err != nil {
					log.Printf("%s[%s]%s: %s", red, "read_request", end, err)
					return
				}

				log.Printf("%s[%s]%s: %s - from %s\n", orange, req.Method, end, req.Host, clientConn.RemoteAddr())

				// req.Header.Del("Proxy-Connection")
				// req.Header.Set("Connection", "keep-alive")

				err = req.Write(serverConn)
				if err != nil {
					log.Printf("%s[%s]%s: %s", red, "write_request", end, err)
					return
				}

				resp, err = http.ReadResponse(bufio.NewReader(serverConn), req)
				if err != nil {
					log.Printf("%s[%s]%s: %s", red, "read_response", end, err)
					return
				}
				defer resp.Body.Close()

				err = resp.Write(clientConn)
				if err != nil {
					log.Printf("%s[%s]%s: %s", red, "write_response", end, err)
					return
				}
			}

		case http.MethodConnect:
			log.Printf("%s[%s]%s: %s - from %s\n", green, r.Method, end, r.Host, r.RemoteAddr)
			io.Copy(io.Discard, r.Body)

			serverConn, err := socksDialer.Dial("tcp", r.Host)
			if err != nil {
				http.Error(w, err.Error(), http.StatusServiceUnavailable)
				return
			}
			defer serverConn.Close()

			hj, ok := w.(http.Hijacker)
			if !ok {
				http.Error(w, err.Error(), http.StatusServiceUnavailable)
				return
			}
			clientConn, _, err := hj.Hijack()
			if err != nil {
				http.Error(w, err.Error(), http.StatusServiceUnavailable)
			}
			defer clientConn.Close()

			// w.WriteHeader(http.StatusOK) puts "Transfer-Encoding: chunked" header
			// and this behaviour can't be avoided
			// for this reason I'm writing the status code response directly to the socket in this way:
			// clientConn.Write([]byte("HTTP/1.1 200 OK\r\n"))
			clientConn.Write([]byte("HTTP/1.1 200 Connection established\r\n"))
			clientConn.Write([]byte(fmt.Sprintf("Date: %s\r\n\r\n", time.Now().Format(http.TimeFormat))))

			errc := make(chan error, 2)
			go func() {
				_, err := io.Copy(serverConn, clientConn)
				if err != nil {
					err = fmt.Errorf("could not copy from client to server, %v", err)
				}
				errc <- err
			}()
			go func() {
				_, err := io.Copy(clientConn, serverConn)
				if err != nil {
					err = fmt.Errorf("could not copy from server to client, %v", err)
				}
				errc <- err
			}()
			err = <-errc
			if err != nil {
				log.Println(err)
			}
		}
	}))
}
