package main

import (
	"flag"
	"log"
	"net/http"
)

var addr = flag.String("addr", ":9200", "http service address")

func main() {
	//parse commadn line arguments
	flag.Parse()
	//create run hub
	hub := NewHub()
	s := &Server{
		Hub: hub,
	}
	hub.OnReceiveMessage = s.OnNewWsMessage
	go hub.Run()

	//server all static files
	http.Handle("/", http.FileServer(http.Dir("./public")))

	//inbound gate MT endpoint
	http.HandleFunc("/mt", s.serveMT)

	//websocket handler
	http.HandleFunc("/ws", s.serveWs)

	log.Printf("\nMock SMS Inbound server up and running @ [%s]\n", *addr)

	err := http.ListenAndServe(*addr, nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
