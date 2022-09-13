
# HTTP to SOCKSv5 Proxy

`http2socks` is an http proxy that relays traffic to a SOCKSv5 proxy in the right format.

It can be used for programs/devices that don't support socks 5 protocol.

## Build

```
go build .
```

## Usage
```
usage: http2socks [bind_addr:bind_port] server_addr:server_port
```
`bind_addr:bind_port` is `:5555` by default.

```
./http2socks 127.0.0.1:5050 192.168.1.10:1080
```
```
2022/09/13 18:10:19 Serving on 127.0.0.1:5050



```
Now you can use `127.0.0.1:5050` as an http proxy and the traffic will be forwarded in the right format to the socks server at `192.168.1.10:1080`

## Notes
* Binding to ports below 1024 requires root priviliges.

## REF
* socks 5 (rfc 1928) : https://datatracker.ietf.org/doc/html/rfc1928
* Hypertext Transfer Protocol (HTTP/1.1) - Semantics and Content (rfc 7231): https://www.rfc-editor.org/rfc/rfc7231