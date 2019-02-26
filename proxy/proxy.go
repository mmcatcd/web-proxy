package proxy

// Props to this article: https://medium.com/@mlowicki/http-s-proxy-in-golang-in-less-than-100-lines-of-code-6a51c2f2c38c
// By: Michal Lowicki to use as a template for the project.

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"../cache"
)

type Proxy struct {
	blacklist []string
	requests  []Request
	cache     *cache.Cache
	Caching   bool
}

type Request struct {
	Timestamp     string `json:"timestamp"`
	Host          string `json:"host"`
	Method        string `json:"method"`
	ContentLength int64  `json:"contentLength"`
	Proto         string `json:"proto"`
}

func New() *Proxy {
	return &Proxy{
		blacklist: make([]string, 0),
		cache:     cache.New(1024),
		Caching:   true,
	}
}

func isFresh(dateString string) (bool, error) {
	date, err := http.ParseTime(dateString)

	if err != nil {
		return false, err
	}

	currentDate := time.Now()
	if currentDate.After(date) {
		return false, nil
	}

	return true, nil
}

func (proxy *Proxy) handleHTTPS(w http.ResponseWriter, r *http.Request) {
	// TCP handshake with server.
	dstServer := r.URL.Host

	// Check for url in the blacklist to block.
	for _, url := range proxy.blacklist {
		if strings.Contains(dstServer, url) {
			return
		}
	}

	dstConnection, err := net.Dial("tcp", dstServer)

	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	w.WriteHeader(http.StatusOK)

	// Hijack the TCP connection from the HTTP library so we can tunnel between server and client.
	hijacker, success := w.(http.Hijacker)

	if !success {
		http.Error(w, "Couldn't hijack.", http.StatusInternalServerError)
		return
	}

	srcConnection, _, err := hijacker.Hijack()

	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
	}

	go tunnel(dstConnection, srcConnection)
	go tunnel(srcConnection, dstConnection)
}

// Tunnels TCP connection between src and dst.
func tunnel(dst, src net.Conn) {
	defer dst.Close()
	defer src.Close()
	io.Copy(dst, src)
}

func isCacheable(r *http.Response) bool {
	header := r.Header

	if header.Get("Cache-Control") != "" {
		cacheControl := header.Get("Cache-Control")

		if !strings.Contains(cacheControl, "private") &&
			!strings.Contains(cacheControl, "no-cache") &&
			!strings.Contains(cacheControl, "no-store") &&
			!strings.Contains(cacheControl, "max-age=0") &&
			strings.Contains(cacheControl, "max-age") {
			return true
		}

		return false
	} else if header.Get("Expires") != "" {
		return true
	} else if header.Get("ETag") != "" {
		return true
	} else if header.Get("Last-Modified") != "" {
		return true
	}

	return false
}

func (proxy *Proxy) handleHTTP(w http.ResponseWriter, req *http.Request) {
	// Check for url in the blacklist to block.
	for _, url := range proxy.blacklist {
		if strings.Contains(req.URL.Host, url) {
			fmt.Fprintf(w, "Blocked by blacklist!")
			return
		}
	}

	// Check the cache for req.URL to see if there's an entry.
	targetURI := req.RequestURI

	fmt.Println("Request came in for: ", targetURI)

	if proxy.Caching {
		proxy.cache.Mutex.Lock()
		cachedResponse := proxy.cache.Get(targetURI)
		proxy.cache.Mutex.Unlock()
		if cachedResponse != nil {
			isFresh, err := proxy.cache.ItemIsFresh(cachedResponse)
			if err != nil {
				fmt.Fprintf(w, "Error in checking if the response is fresh!")
			}

			if isFresh {
				fmt.Println("Found valid cached version of: ", targetURI, ".")

				fmt.Println(len(cachedResponse.BodyBytes))

				respBody := ioutil.NopCloser(bytes.NewBuffer(cachedResponse.BodyBytes))

				resp := cachedResponse.Packet
				// First copy all of the headers from the response.
				copyHeader(w.Header(), resp.Header)

				// Copy the status code of the response.
				w.WriteHeader(resp.StatusCode)

				// Copy the body of the response.
				io.Copy(w, respBody)
				return
			}
		}
	}

	// Create a http client that forwards the request on to the destination server and waits for a response.
	client := &http.Client{}

	req.RequestURI = "" // Resolves this: Get http: Request.RequestURI can't be set in client requests.
	resp, err := client.Do(req)

	// Checks for any errors in getting a response.
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	// First copy all of the headers from the response.
	copyHeader(w.Header(), resp.Header)

	// Copy the status code of the response.
	w.WriteHeader(resp.StatusCode)

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error writing body bytes: ", err)
	}
	resp.Body.Close()

	resp.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))

	// Copy the body of the response.
	io.Copy(w, resp.Body)

	resp.Body.Close()

	// Check that the response is cacheable.
	if proxy.Caching {
		proxy.cache.Mutex.Lock()
		cachedItem := proxy.cache.Get(targetURI)
		proxy.cache.Mutex.Unlock()
		if isCacheable(resp) && cachedItem == nil {
			cacheItem := &cache.CacheItem{
				Key:       targetURI,
				Packet:    resp,
				BodyBytes: bodyBytes,
			}

			proxy.cache.Mutex.Lock()
			proxy.cache.Set(cacheItem)
			proxy.cache.Mutex.Unlock()
			fmt.Println("Inserted item at: ", targetURI, " into the cache!")

			proxy.cache.Mutex.Lock()
			resp = proxy.cache.Get(targetURI).Packet
			proxy.cache.Mutex.Unlock()
		}
	}

	/*
	 * When the handler exits, it will then forward the HTTP response
	 * on to the original sender of the HTTP request.
	 */
}

// Takes all src HTTP request headers and copies them to dst header.
func copyHeader(dst, src http.Header) {
	for name, values := range src {
		dst[name] = values
	}
}

/*
 * Decides whether to forward on the request or handle it internally.
 * Each one of these functions gets called as a goroutine which runs
 * the code in a new thread.
 */
func (proxy *Proxy) httpHandler(w http.ResponseWriter, r *http.Request) {
	t := time.Now()
	formatted := fmt.Sprintf("%d-%02d-%02d %02d:%02d:%02d",
		t.Year(), t.Month(), t.Day(),
		t.Hour(), t.Minute(), t.Second())

	request := Request{
		Timestamp:     formatted,
		Host:          r.Host,
		Method:        r.Method,
		ContentLength: r.ContentLength,
		Proto:         r.Proto,
	}
	proxy.requests = append(proxy.requests, request)

	switch r.URL.Path {
	case "/blacklist":
		proxy.blacklistHandler(w, r)
	case "/requests":
		proxy.getRequestsHandler(w, r)
	default:
		if strings.Contains(r.URL.Path, "/dashboard/") {
			http.ServeFile(w, r, r.URL.Path[1:])
		} else if r.Method == http.MethodConnect {
			proxy.handleHTTPS(w, r)
		} else {
			proxy.handleHTTP(w, r)
		}
	}
}

// Start the Proxy server.
func (proxy *Proxy) Start() {
	proxy.blacklist = append(proxy.blacklist, "amazon.com")

	server := &http.Server{
		Addr:         ":8888",
		Handler:      http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { proxy.httpHandler(w, r) }),
		TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler)),
	}

	http.Handle("/", http.FileServer(http.Dir("public/")))

	log.Fatal(server.ListenAndServeTLS("server.pem", "server.key"))
}
