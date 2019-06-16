package wsconn

import (
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// TextMessage is text message type
	TextMessage = websocket.TextMessage
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

type recover struct {
	commands []string
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
	dropConnChan     chan struct{}    // we use a channel to grab all dorp connections request
	reqHeader        http.Header      // request header
	httpResp         *http.Response   // httpResponse used only for debuging
	recoverCommands  recover          // if a reconnection ocured, we will write to socket all comands, right after reconnect
	reconnectAtempts int              // current number of reconnect tryes
	status           connectionStatus // the connection status: true = connected

	URL                    string        // the URL to dial
	KeepAliveTimeout       time.Duration // the inerval for sending pings
	MaxReconnectionAtempts int           // max reconnection atempts
	TimeBetweenReconnects  time.Duration // how many seconds we wait between reonnect atempts
	ReadBufferSize         int           // read buffer size
	WriteBufferSize        int           // write buffer size
	SuccessfulReconnect    chan struct{} // this chanel send a signal when a reconnetion is succesfull
	FatalErrorChan         chan error    // this is a channel were we trow all fatal errors
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
		dropConnChan:        make(chan struct{}),
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

// SetReqHeader set the reques header
func (wsc *WsConn) SetReqHeader(reqHeader http.Header) {
	wsc.reqHeader = reqHeader
}

// Run is starting the magic :)
func (wsc *WsConn) Run() {
	wsc.setDialer()
	go wsc.listenForDropConnMsg()
	go wsc.reconnect()
	go wsc.listenForWrite()
	wsc.Connect()

	for !wsc.status.isConnected() {
		time.Sleep(500 * time.Microsecond)
	}

	wsc.keepAlive()

}

func (wsc *WsConn) setDialer() {
	wsc.dialer = &websocket.Dialer{
		ReadBufferSize:  wsc.ReadBufferSize,
		WriteBufferSize: wsc.WriteBufferSize,
	}
}

// AddToRecoverCommands generate a list of commands to be executed right after reconnect
func (wsc *WsConn) AddToRecoverCommands(command string) {
	wsc.recoverCommands.Lock()
	wsc.recoverCommands.commands = append(wsc.recoverCommands.commands, command)
	wsc.recoverCommands.Unlock()
}
