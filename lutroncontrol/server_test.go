package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAssetCacheRevalidation(t *testing.T) {
	for _, prefix := range []string{"", "/secret"} {
		t.Run(prefix+"/", func(t *testing.T) {
			s := &Server{assetDir: "assets", basePath: prefix + "/"}
			// Match the normalized base path used by NewServer.
			if prefix != "" {
				s.basePath = prefix
			}
			handler := s.addRoutes()
			for _, asset := range []string{"/", "/style.css?v=20261023-1", "/js/api.js?v=20261023-1", "/js/app.js?v=20261023-1"} {
				t.Run(asset, func(t *testing.T) {
					url := prefix + asset
					recorder := httptest.NewRecorder()
					handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, url, nil))
					if recorder.Code != http.StatusOK {
						t.Fatalf("GET %s: status %d", url, recorder.Code)
					}
					if got := recorder.Header().Get("Cache-Control"); got != "no-cache" {
						t.Fatalf("Cache-Control = %q", got)
					}
					modified := recorder.Header().Get("Last-Modified")
					if modified == "" {
						t.Fatal("missing Last-Modified validator")
					}
					request := httptest.NewRequest(http.MethodGet, url, nil)
					request.Header.Set("If-Modified-Since", modified)
					recorder = httptest.NewRecorder()
					handler.ServeHTTP(recorder, request)
					if recorder.Code != http.StatusNotModified {
						t.Fatalf("conditional GET %s: status %d", url, recorder.Code)
					}
					if got := recorder.Header().Get("Cache-Control"); got != "no-cache" {
						t.Fatalf("304 Cache-Control = %q", got)
					}
				})
			}
		})
	}
}
