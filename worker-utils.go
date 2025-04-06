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

func unsubscribeChannels(callback string, bearer string) error {
	channels, err := queries.GetChannels(ctx)
	if err != nil {
		fmt.Println(err)
		return err
	}
	err_resp := []error{}

	for _, channel := range channels {
		err := server.PostPubSub(channel.ID, Subscribe, callback, bearer)
		if err != nil {
			err_resp = append(err_resp, err)
		}
	}
	logErrors(err_resp)
	fmt.Println("Unsubscribed from channels")
	return nil
}

func subscribeToChannels(channels []database.Channel, callback string, bearer string, ch chan<- Log) {
	for _, channel := range channels {
		err := server.PostPubSub(channel.ID, Subscribe, callback, bearer)
		if err != nil {
			ch <- Log{Err: err}
			continue
		}
		ch <- Log{Msg: fmt.Sprintf("Subscribed to %s", channel.ID)}
	}
}
