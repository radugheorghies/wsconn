package wsconn

import (
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type keepAliveResponse struct {
	lastResponse time.Time
	sync.RWMutex
}

func (k *keepAliveResponse) setLastResponse() {
	k.Lock()
	defer k.Unlock()

	k.lastResponse = time.Now()
}

func (k *keepAliveResponse) getLastResponse() time.Time {
	k.RLock()
	defer k.RUnlock()

	return k.lastResponse
}

func (wsc *WsConn) keepAlive() {
	ticker := time.NewTicker(wsc.KeepAliveTimeout)
	keepAliveR := &keepAliveResponse{}

	wsc.ws.SetPongHandler(func(msg string) error {
		keepAliveR.setLastResponse()
		return nil
	})

	go func() {
		defer ticker.Stop()

		for {
			if wsc.status.isConnected() {
				if err := wsc.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
					log.Println(err)
					wsc.dropConnection()
					return
				}
			} else {
				log.Println("Socket is no longer connected, we didn't send ping msg")
				return
			}
			// now we wait for the timeout moment
			<-ticker.C
			// test timeout condition
			if wsc.status.isConnected() {
				if time.Now().Sub(keepAliveR.getLastResponse()) > wsc.KeepAliveTimeout {
					log.Println("Timeout on ping!!!")
					wsc.dropConnection()
					return
				}
			} else {
				log.Println("Socket is no longer connected, we didn't test the ping response")
				return
			}
		}
	}()
}
