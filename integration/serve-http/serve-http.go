// Serve an HTTP request read on stdin.  The requested resource is read from a
// static file in a directory.  The response is written to stdout.  This input
// and output is tailored to work with QEMU's guestfwd option, see ../qemu.sh.
package main

import (
	"bufio"
	"bytes"
	"flag"

	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
)

func main() {
	dir := flag.String("d", ".", "directory with static files to serve")
	flag.Parse()

	req, err := http.ReadRequest(bufio.NewReader(os.Stdin))
	if err != nil {
		log.Fatalf("Failed reading request: %v", err)
	}

	rsp := response(req, *dir)
	if err := rsp.Write(os.Stdout); err != nil {
		log.Fatalf("Failed writing response: %v", err)
	}
}

func response(req *http.Request, dir string) *http.Response {
	// Eliminate evil use of ".." from the url path
	file := path.Join(dir, path.Clean(req.URL.Path))

	responseBuffer := bytes.Buffer{}
	w := httptest.ResponseRecorder{Body: &responseBuffer}
	http.ServeFile(&w, req, file)
	response := w.Result()
	response.Request = req

	return response
}
