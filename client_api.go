package cloudflare

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"golang.org/x/net/context"
)

var baseURL = "https://api.cloudflare.com/client/v4"

func apiURL(format string, a ...interface{}) string {
	return fmt.Sprintf("%s%s", baseURL, fmt.Sprintf(format, a...))
}

func readResponse(r io.Reader) (result *Response, err error) {
	result = new(Response)
	err = json.NewDecoder(r).Decode(result)
	if err != nil {
		return
	}
	if result.Success {
		return
	}
	if len(result.Errors) > 0 {
		return nil, errors.New(strings.Join(result.Errors, ", "))
	}
	return
}

type httpResponse struct {
	resp *http.Response
	err  error
}

func httpDo(ctx context.Context, opts *Options, method, url string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Auth-Email", opts.Email)
	req.Header.Set("X-Auth-Key", opts.Key)

	transport := &http.Transport{}
	client := &http.Client{Transport: transport}

	respchan := make(chan *httpResponse, 1)
	go func() {
		resp, err := client.Do(req)
		respchan <- &httpResponse{resp: resp, err: err}
	}()

	select {
	case <-ctx.Done():
		transport.CancelRequest(req)
		<-respchan
		return nil, ctx.Err()
	case r := <-respchan:
		return r.resp, r.err
	}
}
