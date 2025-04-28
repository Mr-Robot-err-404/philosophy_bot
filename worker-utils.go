package main

import (
	"bot/philosophy/internal/database"
	"bot/philosophy/internal/server"
	"fmt"
)

func readServerState(rd chan<- ReadReq) ServerState {
	ch := make(chan ServerState)
	defer close(ch)

	rd <- ReadReq{resp: ch}
	state := <-ch
	return state
}

func readChannelByTag(tag string, find chan<- FindTag) rdChannelResp {
	ch := make(chan rdChannelResp)
	defer close(ch)

	find <- FindTag{resp: ch, value: tag}
	return <-ch
}
func updateChannelFreq(params database.UpdateChannelFreqParams, update chan<- Freq) error {
	ch := make(chan error)
	defer close(ch)

	update <- Freq{resp: ch, params: params}
	return <-ch
}

func createChannel(params database.CreateChannelParams, create chan<- CreateChannel) CreateResp {
	ch := make(chan CreateResp)
	defer close(ch)

	create <- CreateChannel{params: params, resp: ch}
	return <-ch
}
func saveReply(params database.StoreReplyParams, reply chan<- SaveReply) ReplyResp {
	ch := make(chan ReplyResp)
	defer close(ch)

	reply <- SaveReply{params: params, resp: ch}
	return <-ch
}
func getAllChannels(getAll chan<- GetAll) GetAllResp {
	ch := make(chan GetAllResp)
	defer close(ch)

	getAll <- GetAll{resp: ch}
	return <-ch
}
func getUnusedQuotes(channelId string, unused chan<- GetUnused) UnusedResp {
	ch := make(chan UnusedResp)
	defer close(ch)

	unused <- GetUnused{id: channelId, resp: ch}
	return <-ch
}

func simpleMan(id string, simple chan<- SimpleMan) error {
	ch := make(chan error)
	defer close(ch)

	simple <- SimpleMan{id: id, resp: ch}
	return <-ch
}
func suddenEpiphany(epiphany database.CreateQuoteParams, wisdom chan<- Wisdom) WisdomResp {
	ch := make(chan WisdomResp)
	defer close(ch)

	wisdom <- Wisdom{epiphany: epiphany, resp: ch}
	return <-ch
}
func findChannel(id string, get chan<- GetChannel) GetResp {
	ch := make(chan GetResp)
	defer close(ch)

	get <- GetChannel{id: id, resp: ch}
	return <-ch
}
func updateSeen(params database.UpdateVideosSincePostParams, seen chan<- SeenVid) error {
	ch := make(chan error)
	defer close(ch)

	seen <- SeenVid{params: params, resp: ch}
	return <-ch
}
func getPopularComments(popular chan<- PopularComments) PopularCommentsResp {
	ch := make(chan PopularCommentsResp)
	defer close(ch)

	popular <- PopularComments{resp: ch}
	return <-ch
}
func getPopularReplies(popular chan<- PopularReplies) PopularRepliesResp {
	ch := make(chan PopularRepliesResp)
	defer close(ch)

	popular <- PopularReplies{resp: ch}
	return <-ch
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
