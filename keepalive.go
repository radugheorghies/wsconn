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
	keepAliveR.setLastResponse() // initiate keepalive with the current timestamp

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
					wsc.dropConnChan <- struct{}{}
					return
				}

				// now we wait for the timeout moment
				<-ticker.C

				// test timeout condition
				if time.Now().Sub(keepAliveR.getLastResponse()) > wsc.KeepAliveTimeout {
					log.Println("Ping timeout! Reconnecting.")
					log.Println("Diference in time:", time.Now().Sub(keepAliveR.getLastResponse()))
					wsc.dropConnChan <- struct{}{}
					return
				}

			} else {
				log.Println("Socket is no longer connected, we didn't send ping msg")
				return
			}
		}
	}()
}
