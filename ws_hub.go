package main

import (
	"encoding/json"
	"log"

	"github.com/gorilla/websocket"
)

//ws message type
const (
	WSMsgTypeMO    = "mo"
	WSMsgTypeMORep = "mo_reply"
	WSMsgTypeMT    = "mt"
)

//WSHub maintains the set of active clients and broadcasts messages to theclients.
type WSHub struct {
	// Registered clients.
	clients map[*WSClient]bool

	// Inbound messages from the clients.
	broadcast chan []byte

	// Register requests from the clients.
	register chan *WSClient

	// Unregister requests from clients.
	unregister chan *WSClient

	OnReceiveMessage func(message *WSMessage)
}

//WSMessage is WebSocker message struct
type WSMessage struct {
	MsgType string            `json:"type"`
	Data    map[string]string `json:"data"`
}

//NewHub is hub constructor
func NewHub() *WSHub {
	return &WSHub{
		broadcast:  make(chan []byte),
		register:   make(chan *WSClient),
		unregister: make(chan *WSClient),
		clients:    make(map[*WSClient]bool),
	}
}

//NewConnection creates and registeres new WS client
func (h *WSHub) NewConnection(conn *websocket.Conn) {
	cl := &WSClient{
		hub:  h,
		conn: conn,
		send: make(chan []byte, 256),
	}

	h.register <- cl
	go cl.writePump()
	cl.readPump()
}

//BroadcastMessage broadcasts a message to all registered ws clients
func (h *WSHub) BroadcastMessage(message *WSMessage) {
	messageBytes, err := json.Marshal(message)
	if err != nil {
		log.Fatal(err)
	}

	h.broadcast <- messageBytes
}

//ReceiveMessage is method to receive messages, calls hook
func (h *WSHub) ReceiveMessage(message []byte) {
	if h.OnReceiveMessage == nil {
		return
	}
	m := WSMessage{}
	err := json.Unmarshal(message, &m)
	if err != nil {
		log.Printf("Error parsing ws message %s", message)
		return
	}
	h.OnReceiveMessage(&m)
}

//Run the hub channels
func (h *WSHub) Run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true
		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
		case message := <-h.broadcast:
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
		}
	}
}
