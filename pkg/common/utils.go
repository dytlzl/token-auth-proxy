package common

import (
	"io"
	"log"
	"net/http"
	"sync"
)

func transfer(dst io.Writer, src io.Reader, wg *sync.WaitGroup) {
	defer wg.Done()
	_, err := io.Copy(dst, src)
	if err != nil {
		log.Println(err)
	}
}

func TransferBidirectionally(dst io.ReadWriter, src io.ReadWriter) {
	wg := sync.WaitGroup{}
	wg.Add(2)
	go transfer(dst, src, &wg)
	go transfer(src, dst, &wg)
	wg.Wait()
}

func copyHeader(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}

func TransferHTTPRequest(transport http.RoundTripper, w http.ResponseWriter, r *http.Request) {
	resp, err := transport.RoundTrip(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	defer resp.Body.Close()
	copyHeader(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)
	_, err = io.Copy(w, resp.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
}
