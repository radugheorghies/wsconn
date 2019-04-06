# wsconn

[![Go Report Card](https://goreportcard.com/badge/github.com/radugheorghies/wsconn)](https://goreportcard.com/report/github.com/radugheorghies/wsconn)
[![GitHub license](https://img.shields.io/github/license/Naereen/StrapDown.js.svg)](https://github.com/Naereen/StrapDown.js/blob/master/LICENSE)

wsconn is a websocket client based on [gorilla/websocket](https://github.com/gorilla/websocket) that automatically reconnects if the connection is dropped. It is thread safe, all write opperations are sent through a chanel, so you can have multiple goroutines that write to socket in the same time. If an error occured, you can wait until you receive a successful reconnect message (see the example)

If you have a list of write messages that you need to be executed right after a reconnect event, you can store them with AddToRecoverCommands. 

## Installation

    go get "github.com/radugheorghies/wsconn"

## Example

```go
package main

import (
	"log"

	"github.com/gorilla/websocket"
	"github.com/radugheorghies/wsconn"
)

func main() {
	wait := make(chan struct{})
	ws := wsconn.New("wss://api.hitbtc.com/api/2/ws")
	// use ws.SetHeaders if you need to set the headers

	ws.Run()
	defer ws.Close()

	go listenForMessages(ws)
	go listenForFatalErrors(ws)

	// subscribing to trades
	msg := "{\"method\": \"subscribeOrderbook\", \"params\": {\"symbol\": \"ETHBTC\"},\"id\": 123}"

	ws.WriteMessage(websocket.TextMessage, []byte(msg))
	ws.AddToRecoverCommands(msg)

	<-wait

}

func listenForMessages(ws *wsconn.WsConn) {
	for {
		_, v, err := ws.ReadMessage()
		if err != nil {
			log.Println(err)
			// we can wait until receiving the succesfull reconnect message
			<-ws.SuccessfulReconnect
		}
		log.Println(string(v))
	}
}

func listenForFatalErrors(ws *wsconn.WsConn) {
	for err := range ws.FatalErrorChan {
		log.Fatalln("Fatal error on socket:", err)
	}
}
```

## To do

* adding more tests

## Thanks

I've got inspired from the other packeges available:

* [mariuspass/recws](https://github.com/mariuspass/recws)

Thank you all for sharing your code!

## Licence

[![GitHub license](https://img.shields.io/github/license/Naereen/StrapDown.js.svg)](https://github.com/Naereen/StrapDown.js/blob/master/LICENSE)