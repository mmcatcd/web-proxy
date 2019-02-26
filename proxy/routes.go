package proxy

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

type BlackList struct {
	Blacklist []string `json:"blacklist"`
}

type Requests []Request

func (proxy *Proxy) blacklistHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		data := &BlackList{
			Blacklist: proxy.blacklist,
		}

		out, err := json.Marshal(data)
		if err != nil {
			fmt.Fprintf(w, "Couldn't marshal!")
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(out)

	case "POST":
		blacklist := BlackList{}

		// Decode the stuff
		data, err := ioutil.ReadAll(r.Body)
		if err != nil {
			fmt.Fprintf(w, "Couldn't read the body.")
		}

		err = json.Unmarshal(data, &blacklist)
		if err != nil {
			fmt.Fprintf(w, "Decoding error!")
		}

		proxy.blacklist = blacklist.Blacklist

		// Send response
		out, err := json.Marshal(blacklist)
		if err != nil {
			fmt.Fprintf(w, "Couldn't marshal!")
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(out)
	}
}

func (proxy *Proxy) getRequestsHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		data := proxy.requests

		out, err := json.Marshal(data)
		if err != nil {
			fmt.Fprintf(w, "Couldn't marshal!")
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(out)
	}
}
