package main

import (
	"bot/philosophy/internal/database"
	"bot/philosophy/internal/server"
	"fmt"
)

func readServerState(rd chan ReadReq) ServerState {
	ch := make(chan ServerState)
	defer close(ch)

	req := ReadReq{resp: ch}
	rd <- req

	state := <-ch
	return state
}

func subscribeToChannels(channels []database.Channel, callback string, bearer string) error {
	for _, channel := range channels {
		err := server.PostPubSub(channel.ID, Subscribe, callback, bearer)
		if err != nil {
			return err
		}
		fmt.Printf("Subscribed: %s\n", channel.ID)
	}
	return nil
}
