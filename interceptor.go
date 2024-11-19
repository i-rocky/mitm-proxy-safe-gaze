package main

import (
	"bytes"
	"github.com/AdguardTeam/gomitmproxy"
	"github.com/AdguardTeam/gomitmproxy/proxyutil"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
)

func onResponse(session *gomitmproxy.Session) *http.Response {
	res := session.Response()
	req := session.Request()

	log.Printf("Content-Type: %s %s", res.Header.Get("Content-Type"), req.URL.String())

	if strings.HasPrefix(res.Header.Get("Content-Type"), "image/") && !strings.Contains(req.URL.String(), "://safe-gaze.clapbox.net/") {
		encodedUrl := url.QueryEscape(req.URL.String())

		imgRes, err := http.Get("https://safe-gaze.clapbox.net/?q=" + encodedUrl)
		if err != nil {
			res.StatusCode = http.StatusInternalServerError
			return res
		}

		res.Header.Set("Content-Type", imgRes.Header.Get("Content-Type"))
		res.Body = imgRes.Body
		res.ContentLength = imgRes.ContentLength
		res.StatusCode = imgRes.StatusCode

		return res
	}

	if strings.HasPrefix(res.Header.Get("Content-Type"), "text/html") {
		b, err := proxyutil.ReadDecompressedBody(res)
		_ = res.Body.Close()
		if err != nil {
			log.Printf("failed to read body: %v", err)
			return proxyutil.NewErrorResponse(req, err)
		}

		decoded, err := proxyutil.DecodeLatin1(bytes.NewReader(b))
		if err != nil {
			log.Printf("failed to decode body: %v", err)
			return proxyutil.NewErrorResponse(req, err)
		}

		nContent := replaceBase64Images(decoded)

		encoded, err := proxyutil.EncodeLatin1(nContent)
		if err != nil {
			log.Printf("failed to encode body: %v", err)
			return proxyutil.NewErrorResponse(req, err)
		}

		res.Body = io.NopCloser(bytes.NewReader(encoded))
		res.Header.Del("Content-Encoding")
		res.ContentLength = int64(len(encoded))

		return res
	}

	return res
}
