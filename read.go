package wsconn

// ReadMessage will read message from the ws
// If the connection is closed ErrNotConnected is returned
func (wsc *WsConn) ReadMessage() (messageType int, message []byte, err error) {
	err = ErrNotConnected

	if wsc.status.isConnected() {

		messageType, message, err = wsc.ws.ReadMessage()
		if err != nil {
			wsc.dropConnChan <- struct{}{}
		}

	}
	return
}
