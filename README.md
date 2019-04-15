# Web Proxy

## Overview

This is an implementation of a web proxy compatible with HTTP/HTTPS and WebSockets with a dashboard that 
logs requests and allows to blacklist URL's. Caching is also implemented for HTTP.

## Installation Instructions

### Generate an SSL Cert

```
$ ./cert.sh
```

Then you will have to add it to keychain for browsers to trust the cert. Add the `.pem` 
cert into the certificates section of the keychain and right click, selecting `Get Info`.
Then under trust select `Always Trust`. Sin é.

### Start the Proxy Server

```
$ go run proxy.go
```

### Open a browser redirecting to the proxy

```$ open -a /Applications/Chromium.app --args --proxy-server="https://localhost:8888"```
