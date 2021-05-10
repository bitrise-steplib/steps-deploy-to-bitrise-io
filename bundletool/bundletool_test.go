package bundletool

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_fetchAny(t *testing.T) {
	tests := []struct {
		name    string
		fn1     func(w http.ResponseWriter, r *http.Request)
		fn2     func(w http.ResponseWriter, r *http.Request)
		wantErr bool
	}{
		{
			name:    "found - 1 url",
			wantErr: false,
			fn1: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			},
			fn2: nil,
		},
		{
			name:    "not found - 1 url",
			wantErr: true,
			fn1: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusBadRequest)
			},
			fn2: nil,
		},
		{
			name:    "found - 2 url",
			wantErr: false, fn1: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusBadRequest)
			},
			fn2: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			},
		},
		{
			name:    "not found - 2 url",
			wantErr: true,
			fn1: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusBadRequest)
			},
			fn2: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusBadRequest)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts1 := httptest.NewServer(http.HandlerFunc(tt.fn1))

			fallbackURLs := []string{}
			if tt.fn2 != nil {
				t.Log("add seccondary url")
				ts2 := httptest.NewServer(http.HandlerFunc(tt.fn2))
				fallbackURLs = append(fallbackURLs, ts2.URL)
			}
			t.Logf("source: %s, fallbackSources: %s", ts1.URL, fallbackURLs)

			got, err := fetchAny(ts1.URL, fallbackURLs...)
			if (err != nil) != tt.wantErr {
				t.Errorf("fetchAny() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				assert.NotNil(t, got)
			}
		})
	}
}
