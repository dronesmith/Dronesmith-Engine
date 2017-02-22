/**
 * Dronesmith API
 *
 * Authors
 *  Geoff Gardner <geoff@dronesmith.io>
 *
 * Copyright (C) 2017 Dronesmith Technologies Inc, all rights reserved.
 * Unauthorized copying of any source code or assets within this project, via
 * any medium is strictly prohibited.
 *
 * Proprietary and confidential.
 */

package apiservice

import (
	"fmt"
	"net/http"
	"time"

  "config"
)

const patience time.Duration = time.Second * 1

type StreamBroker struct {
	Notifier       chan []byte
	newClients     chan chan []byte
	closingClients chan chan []byte
	clients        map[chan []byte]bool
}

func NewStreamListener() (broker *StreamBroker) {
	broker = &StreamBroker{
		Notifier:       make(chan []byte, 1),
		newClients:     make(chan chan []byte),
		closingClients: make(chan chan []byte),
		clients:        make(map[chan []byte]bool),
	}

	go broker.listen()
  return
}

func (broker *StreamBroker) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	flusher, ok := rw.(http.Flusher)

	if !ok {
		http.Error(rw, "Streaming unsupported!", http.StatusInternalServerError)
		return
	}

	rw.Header().Set("Content-Type", "text/event-stream")
	rw.Header().Set("Cache-Control", "no-cache")
	rw.Header().Set("Connection", "keep-alive")
	rw.Header().Set("Access-Control-Allow-Origin", "*")

	messageChan := make(chan []byte)

	broker.newClients <- messageChan

	defer func() {
		broker.closingClients <- messageChan
	}()

	notify := rw.(http.CloseNotifier).CloseNotify()

	for {
		select {
		case <-notify:
			return
		default:
			fmt.Fprintf(rw, "%s\n\n", <-messageChan)
			flusher.Flush()
		}
	}

}

func (broker *StreamBroker) listen() {
	for {
		select {
		case s := <-broker.newClients:
			broker.clients[s] = true
			config.Log(config.LOG_INFO, "Stream | Client added.", len(broker.clients), "registered clients")

		case s := <-broker.closingClients:
			delete(broker.clients, s)
			config.Log(config.LOG_INFO, "Stream | Removed client.", len(broker.clients), "registered clients")

		case event := <-broker.Notifier:

			for clientMessageChan, _ := range broker.clients {
				select {
				case clientMessageChan <- event:
				case <-time.After(patience):
					config.Log(config.LOG_INFO, "Stream | Skipping client.")
				}
			}
		}
	}
}
