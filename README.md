# go-foward-proxy

## Generate a certificate for local testing purposes
```
openssl genrsa 2048 > server.key
openssl req -new -key server.key > server.csr
openssl x509 -req \
 -days 3650 \
 -signkey server.key \
 -extfile san.txt \
 < server.csr > server.crt

```
