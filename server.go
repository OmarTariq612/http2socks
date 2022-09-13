package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"

	"golang.org/x/net/proxy"
)

type Relayer struct {
	bindAddr   string
	serverAddr string
}

func NewRelayer(bindAddr, serverAddr string) *Relayer {
	return &Relayer{bindAddr: bindAddr, serverAddr: serverAddr}
}

const (
	green = "\033[92m"
	// blue   = "\033[94m"
	// red = "\033[0;31m"
	orange = "\033[38;5;214m"
	end    = "\033[0m"
)

func (relayer *Relayer) ListenAndServe() error {
	socksDialer, err := proxy.SOCKS5("tcp", relayer.serverAddr, nil, proxy.Direct)
	if err != nil {
		return fmt.Errorf("could not create a socks dialer, %v", err)
	}

	log.Println("Serving on", relayer.bindAddr)

	return http.ListenAndServe(relayer.bindAddr, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		default:
			log.Printf("%v[%v]%v: %v - from %v\n", orange, r.Method, end, r.Host, r.RemoteAddr)

			_, _, err = net.SplitHostPort(r.Host)
			if err != nil {
				r.Host = fmt.Sprintf("%s:80", r.Host)
			}

			serverConn, err := socksDialer.Dial("tcp", r.Host)
			if err != nil {
				http.Error(w, err.Error(), http.StatusServiceUnavailable)
				return
			}
			defer serverConn.Close()

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

			//copy headers
			for key, values := range resp.Header {
				for _, value := range values {
					w.Header().Add(key, value)
				}
			}
			// copy body
			_, err = io.Copy(w, resp.Body)
			if err != nil {
				log.Println(err)
			}

		case http.MethodConnect:
			log.Printf("%v[%v]%v: %v - from %v\n", green, r.Method, end, r.Host, r.RemoteAddr)
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
			clientConn.Write([]byte("HTTP/1.1 200 OK\r\n\r\n"))

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
