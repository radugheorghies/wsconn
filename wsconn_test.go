package wsconn_test

import (
	"testing"

	"bitbucket.org/ircrypto/tradepipe/pkg/core/wsconn"
)

func TestNew(t *testing.T) {
	url := "ws://example.com"
	wsc := wsconn.New(url)

	if wsc.URL != url {
		t.Errorf("New(%s) = %s; want %s", url, wsc.URL, url)
	}

	if wsc.MaxReconnectionAtempts != wsconn.DefaultMaxReconnectionAtempts {
		t.Errorf("wsc.MaxReconnectionAtempts = %v; want %v", wsc.MaxReconnectionAtempts, wsconn.DefaultMaxReconnectionAtempts)
	}

	if wsc.TimeBetweenReconnects != wsconn.DefaultTimeBetweenReconnects {
		t.Errorf("wsc.TimeBetweenReconnects = %v; want %v", wsc.TimeBetweenReconnects, wsconn.DefaultTimeBetweenReconnects)
	}

	if wsc.KeepAliveTimeout != wsconn.DefaultKeepAliveTimeout {
		t.Errorf("wsc.KeepAliveTimeout = %v; want %v", wsc.KeepAliveTimeout, wsconn.DefaultKeepAliveTimeout)
	}

	if wsc.ReadBufferSize != wsconn.DefaultReadBufferSize {
		t.Errorf("wsc.ReadBufferSize = %v; want %v", wsc.ReadBufferSize, wsconn.DefaultReadBufferSize)
	}

	if wsc.WriteBufferSize != wsconn.DefaultWriteBufferSize {
		t.Errorf("wsc.WriteBufferSize = %v; want %v", wsc.WriteBufferSize, wsconn.DefaultWriteBufferSize)
	}
}
