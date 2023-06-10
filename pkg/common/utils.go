package common

import (
	"io"
	"log"
	"net/http"
	"sync"
)

func Transfer(dst io.Writer, src io.Reader, wg *sync.WaitGroup) {
	defer wg.Done()
	_, err := io.Copy(dst, src)
	if err != nil {
		log.Println(err)
	}
}

func TransferBidirectionally(dst io.ReadWriter, src io.ReadWriter) {
	wg := sync.WaitGroup{}
	wg.Add(2)
	go Transfer(dst, src, &wg)
	go Transfer(src, dst, &wg)
	wg.Wait()
}

func CopyHeader(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}
