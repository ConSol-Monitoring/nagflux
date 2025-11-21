package helper

import (
	"errors"
	"io"
	"net/http"
	"strings"
)

// RequestedReturnCodeIsOK makes an HEAD or GET request. If the returncode is 2XX it will return true.
func RequestedReturnCodeIsOK(client http.Client, url, function string) bool {
	var resp *http.Response
	var err error
	switch function {
	case "HEAD":
		resp, err = client.Head(url)
		if err == nil {
			resp.Body.Close()
		}
	case "GET":
		resp, err = client.Get(url)
		if err == nil {
			resp.Body.Close()
		}
	default:
		err = errors.New("unknown function")
	}
	if err == nil && isReturnCodeOK(resp) {
		return true
	}
	return false
}

// SentReturnCodeIsOK makes the given request. If the returncode is 2XX it will return true and the body else the error message.
func SentReturnCodeIsOK(client http.Client, url, function string, data string) (bool, string) {
	var req *http.Request
	var resp *http.Response
	var err error

	req, err = http.NewRequest(function, url, strings.NewReader(data))
	req.Header.Set("User-Agent", "Nagflux")
	if err != nil {
		return false, err.Error()
	}

	resp, err = client.Do(req)
	if err != nil {
		return false, err.Error()
	}
	defer resp.Body.Close()

	if isReturnCodeOK(resp) {
		return true, getBody(resp)
	}
	return false, resp.Status
}

func isReturnCodeOK(resp *http.Response) bool {
	return resp != nil && resp.StatusCode >= 200 && resp.StatusCode < 300
}

func getBody(resp *http.Response) string {
	if resp != nil {
		body, err := io.ReadAll(resp.Body)
		if err == nil {
			return string(body)
		}
	}
	return ""
}

// GetHeaders makes an HEAD or GET request. If no errors, it will return all HTTP Response Headers
func GetHeaders(client http.Client, url, function string) map[string]string {
	var resp *http.Response
	var err error
	r := make(map[string]string)
	switch function {
	case "HEAD":
		resp, err = client.Head(url)
		if err != nil {
			defer resp.Body.Close()
		}
	case "GET":
		resp, err = client.Get(url)
		if err != nil {
			defer resp.Body.Close()
		}
	default:
		err = errors.New("unknown function")
	}
	if err != nil {
		return r
	}

	for name, values := range resp.Header {
		// Loop over all values for the name.
		for _, value := range values {
			r[name] = value
		}
	}
	return r
}
