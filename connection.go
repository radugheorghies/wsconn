package wsconn

import (
	"errors"
	"log"
	"time"
)

// Connect to ws
func (wsc *WsConn) Connect() {
	wsc.dropConnection()
}

// Close closes the underlying network connection without
// sending or waiting for a close frame.
func (wsc *WsConn) Close() {
	wsc.status.setStatus(false)
	if wsc.ws != nil {
		wsc.ws.Close()
	}
}

// CloseForTests closes the underlying network connection without
// sending or waiting for a close frame, used only for test scenario.
func (wsc *WsConn) CloseForTests() {
	if wsc.ws != nil {
		wsc.ws.Close()
	}
}

func (wsc *WsConn) dropConnection() {
	wsc.status.setStatus(false)
	wsc.reconnectChan <- struct{}{}
}

func (wsc *WsConn) listenForDropConnMsg() {
	for range wsc.dropConnChan {
		log.Println("Drop connection chan signal received.")
		if wsc.status.isConnected() {
			wsc.dropConnection()
		}
	}
}

func (wsc *WsConn) reconnect() {
	// listen for reconect message
	// if other reconnect message is coming and the
	// reconnect faze is not finished, we will apply the drop pattern
	for range wsc.reconnectChan {

		// reconnect code
		wsc.ws = nil

		for err := wsc.connect(); err != nil && wsc.reconnectAtempts < wsc.MaxReconnectionAtempts; wsc.reconnectAtempts++ {
			log.Println("Error reconnecting to socket: ", err)
			time.Sleep(wsc.TimeBetweenReconnects)
			err = wsc.connect()
		}

		if wsc.reconnectAtempts == wsc.MaxReconnectionAtempts {
			// we achive max reconnection atempts
			// we must kill the process
			fatalError := errors.New("We could't reconnect to the web socket. We tryed too many times")
			wsc.FatalErrorChan <- fatalError
			return
		}

		// running recover comands
		wsc.recoverCommands.Lock()
		for _, v := range wsc.recoverCommands.commands {
			if err := wsc.WriteMessage(1, []byte(v)); err != nil {
				wsc.dropConnection()
			}
		}
		wsc.recoverCommands.Unlock()

	}
}

func (wsc *WsConn) connect() error {
	log.Println("Connecting ....")

	wsConn, httpResp, err := wsc.dialer.Dial(wsc.URL, wsc.reqHeader)

	if err != nil {
		wsc.httpResp = httpResp
		return err
	}

	wsc.ws = wsConn
	wsc.status.setStatus(true)
	wsc.SuccessfulReconnect <- struct{}{}
	wsc.reconnectAtempts = 0

	log.Println("Connected.")

	wsc.keepAlive()

	return nil
}
