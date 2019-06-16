package wsconn

// WriteMessage will write message in channel in order to be written on socket
func (wsc *WsConn) WriteMessage(messageType int, data []byte) error {
	if wsc.status.isConnected() {
		responseChan := make(chan error)
		wsc.writeChan <- message{
			data:         data,
			messageType:  messageType,
			responseChan: responseChan,
		}
		response := <-responseChan
		close(responseChan)
		return response
	}
	return ErrNotConnected
}

// listenForWrite will listen for any message and write it to the ws connection
func (wsc *WsConn) listenForWrite() {
	for msg := range wsc.writeChan {
		err := wsc.ws.WriteMessage(msg.messageType, msg.data)
		if err != nil {
			wsc.dropConnChan <- struct{}{}
		}
		msg.responseChan <- err
	}
}
