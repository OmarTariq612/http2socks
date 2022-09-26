
# HTTP to SOCKSv5 Proxy

`http2socks` is an http proxy that relays traffic to a SOCKSv5 proxy in the right format.

It can be used for programs/devices that don't support socks 5 protocol.

## Build

```
go build .
```

## Usage
```
Usage of http2socks:
  -bind string
        bind address (default ":5555")
  -cred string
        username:password that will be used to authenticate http clients
  -socks string
        socks server address
```
`cred` is an option that can be used to provide `Basic` authentication for http clients.

```
./http2socks --bind 127.0.0.1:5050 --socks 192.168.1.10:1080
```
```
2022/09/13 18:10:19 Serving on 127.0.0.1:5050
2022/09/13 18:10:19 the provided socks server address is 192.168.1.10:1080


```
Now you can use `127.0.0.1:5050` as an http proxy and the traffic will be forwarded in the right format to the socks server at `192.168.1.10:1080`

## Notes
* Binding to ports below 1024 requires root priviliges.
* `Basic` authentication is not a secure way to protect the server as the username:password base64-encoded parameter is sent in clear text.

## REF
* socks 5 (rfc 1928) : https://datatracker.ietf.org/doc/html/rfc1928
* Hypertext Transfer Protocol (HTTP/1.1) - Semantics and Content (rfc 9110): https://www.rfc-editor.org/rfc/rfc9110.html