package main

import (
	"flag"
	"log"
	"strings"
)

func main() {
	bindAddr := flag.String("bind", ":5555", "bind address")
	serverAddr := flag.String("socks", "", "socks server address")
	cred := flag.String("cred", "", "username:password that will be used to authenticate http clients")
	flag.Parse()

	if *serverAddr == "" {
		log.Println("socks flag is required")
		return
	}

	if *cred != "" && !strings.Contains(*cred, ":") {
		log.Println("cred must take the username:password form")
		return
	}

	r := NewRelayer(*bindAddr, *serverAddr, *cred)
	err := r.ListenAndServe()
	if err != nil {
		log.Println(err)
	}
}
