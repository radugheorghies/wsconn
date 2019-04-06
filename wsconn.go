package wsconn

import (
	"errors"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// defaultMaxReconnectionAtempts how many times we will try the reconnect
	defaultMaxReconnectionAtempts = 10
	// defaultTimeBetweenReconnects how many seconds we wait until will try the reconnect
	defaultTimeBetweenReconnects = 10 * time.Second
	// defaultKeepAliveTimeout is the number of secoons between ping messages
	defaultKeepAliveTimeout = 5 * time.Second
	// defaultReadBufferSize is the buffer size for read
	defaultReadBufferSize = 1024
	// defaultWriteBufferSize is the buffer size for write
	defaultWriteBufferSize = 1024
)

var (
	// ErrNotConnected is the generic error when the socket is disconnected
	ErrNotConnected = errors.New("websocket: not connected. reconnecting")
)

type connectionStatus struct {
	connected bool
	sync.Mutex
}

func (cs *connectionStatus) isConnected() bool {
	cs.Lock()
	defer cs.Unlock()
	return cs.connected
}

func (cs *connectionStatus) setStatus(v bool) {
	cs.Lock()
	cs.connected = v
	cs.Unlock()
}

// WsConn is the type of custom socket connection
// base on gorila websocket package
type WsConn struct {
	ws     *websocket.Conn   // websocket connection
	dialer *websocket.Dialer // websocket dialer

	writeChan        chan message     // the chan for writing to socket
	reconnectChan    chan struct{}    // the chan where we send recconect signals
	reqHeader        http.Header      // request header
	httpResp         *http.Response   // httpResponse used only for debuging
	recoverCommands  []string         // if a reconnection ocured, we will write to socket all comands, right after reconnect
	reconnectAtempts int              // current number of reconnect tryes
	status           connectionStatus // the connection status: true = connected

	URL                    string        // the URL to dial
	KeepAliveTimeout       time.Duration // the inerval for sending pings
	MaxReconnectionAtempts int           // max reconnection atempts
	TimeBetweenReconnects  time.Duration // how many seconds we wait between reonnect atempts
	ReadBufferSize         int           // read buffer size
	WriteBufferSize        int           // write buffer size

	SuccessfulReconnect chan struct{} // this chanel send a signal when a reconnetion is succesfull
	FatalErrorChan      chan error    // this is a channel were we trow all fatal errors
}

// for gorilla wesockets, we must specify
type message struct {
	data         []byte
	messageType  int
	responseChan chan error
}

// New is the factory function for this package
func New(url string) *WsConn {
	wsc := WsConn{
		writeChan:           make(chan message, 1),
		reconnectChan:       make(chan struct{}),
		SuccessfulReconnect: make(chan struct{}, 1),
		FatalErrorChan:      make(chan error, 1),
	}

	wsc.URL = url
	wsc.ReadBufferSize = defaultReadBufferSize
	wsc.WriteBufferSize = defaultWriteBufferSize
	wsc.MaxReconnectionAtempts = defaultMaxReconnectionAtempts
	wsc.TimeBetweenReconnects = defaultTimeBetweenReconnects
	wsc.KeepAliveTimeout = defaultKeepAliveTimeout

	return &wsc
}

// // SetReqHeader set the reques header
// func (wsc *WsConn) SetReqHeader(reqHeader http.Header) {
// 	wsc.reqHeader = reqHeader
// }

func (wsc *WsConn) setDialer() {
	wsc.dialer = &websocket.Dialer{
		ReadBufferSize:  wsc.ReadBufferSize,
		WriteBufferSize: wsc.WriteBufferSize,
	}
}

// Run is starting the magic :)
func (wsc *WsConn) Run() {
	wsc.setDialer()
	go wsc.reconnect()
	go wsc.listenForWrite()
	wsc.Connect()
}

// WriteMessage will write message in channel in order to be writen on socket
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
		select {
		case <-wsc.reconnectChan:
			msg.responseChan <- ErrNotConnected
			wsc.dropConnection()
			return
		default:
			err := wsc.ws.WriteMessage(msg.messageType, msg.data)
			if err != nil {
				wsc.dropConnection()
			}
			msg.responseChan <- err
		}
	}
}

// ReadMessage will read message from the ws
// If the connection is closed ErrNotConnected is returned
func (wsc *WsConn) ReadMessage() (messageType int, message []byte, err error) {
	err = ErrNotConnected

	if wsc.status.isConnected() {
		select {
		case <-wsc.reconnectChan:
			wsc.dropConnection()
			return
		default:
			messageType, message, err = wsc.ws.ReadMessage()
			if err != nil {
				wsc.dropConnection()
			}
		}
	}
	return
}

func (wsc *WsConn) dropConnection() {
	wsc.status.setStatus(false)
	wsc.reconnectChan <- struct{}{}
}

func (wsc *WsConn) reconnect() {
	// listen for reconect message
	// if other reconnect message is comming and the
	// reconnect faze is not finished, we will apply the drop pattern
	for range wsc.reconnectChan {
		finishReconnect := make(chan struct{})

		go func() {
			for {
				select {
				case <-wsc.reconnectChan:
					log.Println("Consuming useless reconnect messages")
					// do nothing
				case <-finishReconnect:
					return
				default:
					//do nothing
				}
			}
		}()

		// reconnect code
		wsc.ws = nil

		for err := wsc.connect(); err != nil && wsc.reconnectAtempts < wsc.MaxReconnectionAtempts; wsc.reconnectAtempts++ {
			log.Println("Error reconnecting to socket, we try again: ", err)
			time.Sleep(wsc.TimeBetweenReconnects)
			err = wsc.connect()
		}

		if wsc.reconnectAtempts == wsc.MaxReconnectionAtempts {
			// we finish reconnection atempts
			// we must kill the process
			fatalError := errors.New("We could't reconnect to the web socket")
			wsc.FatalErrorChan <- fatalError
			return
		}

		wsc.reconnectAtempts = 0
		wsc.SuccessfulReconnect <- struct{}{}

		// finishReconnect <- struct{}{}
		close(finishReconnect)

		wsc.keepAlive()

		// running recover comands
		for _, v := range wsc.recoverCommands {
			if err := wsc.WriteMessage(1, []byte(v)); err != nil {
				wsc.dropConnection()
			}
		}
	}
}

// Connect to ws
func (wsc *WsConn) Connect() {
	wsc.dropConnection()
}

func (wsc *WsConn) connect() error {
	log.Println("Connecting")

	wsConn, httpResp, err := wsc.dialer.Dial(wsc.URL, wsc.reqHeader)

	if err != nil {
		wsc.httpResp = httpResp
		return err
	}

	wsc.ws = wsConn
	wsc.status.setStatus(true)

	return nil
}

// Close closes the underlying network connection without
// sending or waiting for a close frame.
func (wsc *WsConn) Close() {
	if wsc.ws != nil {
		wsc.ws.Close()
	}

	wsc.status.setStatus(false)
}

// AddToRecoverCommands generate a list of commands to be executed right after reconnect
func (wsc *WsConn) AddToRecoverCommands(command string) {
	wsc.recoverCommands = append(wsc.recoverCommands, command)
}
