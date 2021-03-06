// Copyright 2013 The Gorilla WebSocket Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package utils

import (
	"encoding/json"

	l "github.com/crosstyan/mqtt-to-ws/logger"
	"github.com/crosstyan/mqtt-to-ws/model"
	"github.com/davecgh/go-spew/spew"
)

var logger = l.Lsugar

// Hub maintains the set of active clients and broadcasts messages to the
// clients.
type Hub struct {
	// Registered clients.
	clients map[*Client]bool

	// Inbound messages from the clients.
	broadcast chan []byte

	// Register requests from the clients.
	register chan *Client

	// Unregister requests from clients.
	unregister chan *Client

	mqttToWs chan model.MQTTMsg
}

func NewWsHub(mqttToWs chan model.MQTTMsg) *Hub {
	return &Hub{
		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
		mqttToWs:   mqttToWs,
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true
		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
		// Just do nothing when there is a broadcast message
		// case message := <-h.broadcast:
		//  lsugar.Infof("Message from other clients: %s", message)
		// 	for client := range h.clients {
		// 		select {
		// 		case client.send <- message:
		// 		default:
		// 			close(client.send)
		// 			delete(h.clients, client)
		// 		}
		// 	}
		case message := <-h.mqttToWs:
			logger.Infof("Message from MQTT:\n%s", spew.Sdump(message))
			marshaled, err := json.Marshal(message)
			if err != nil {
				logger.Errorf("Error marshaling message: %v", err)
			}
			for client := range h.clients {
				select {
				case client.send <- marshaled:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
		}
	}
}
