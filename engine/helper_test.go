package engine

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func serveRemoteModules(t *testing.T, folder string, urlContext string) (string, func()) {
	handler := http.StripPrefix(urlContext, http.FileServer(http.Dir(folder)))
	server := httptest.NewServer(handler)
	done := func() {
		server.CloseClientConnections()
		server.Close()
	}
	return server.URL, done
}
