package web

import (
	"io/fs"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

func TestSPAHandlerServesIndexAndAssets(t *testing.T) {
	assets := fstest.MapFS{
		"index.html":     {Data: []byte("<html>ait</html>")},
		"assets/app.js":  {Data: []byte("console.log('ait')")},
		"assets/app.css": {Data: []byte("body{}")},
	}

	tests := []struct {
		name       string
		path       string
		wantStatus int
		wantBody   string
	}{
		{name: "root", path: "/", wantStatus: http.StatusOK, wantBody: "ait"},
		{name: "spa route", path: "/tasks/task-1", wantStatus: http.StatusOK, wantBody: "ait"},
		{name: "asset", path: "/assets/app.js", wantStatus: http.StatusOK, wantBody: "console.log"},
		{name: "missing asset", path: "/assets/missing.js", wantStatus: http.StatusNotFound, wantBody: "404"},
	}

	handler := spaHandler(assets)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if !strings.Contains(rec.Body.String(), tt.wantBody) {
				t.Fatalf("body = %q, want to contain %q", rec.Body.String(), tt.wantBody)
			}
		})
	}
}

func TestDistFSFindsBuiltDist(t *testing.T) {
	assets, err := distFS()
	if err != nil {
		t.Skipf("web dist is not built: %v", err)
	}
	if _, err := fs.Stat(assets, "index.html"); err != nil {
		t.Fatalf("embedded/dev dist missing index.html: %v", err)
	}
}
