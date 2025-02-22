package main

import (
	"sync"
	"time"
)

type ReplyStatus struct {
	Resp     PostedReplyResp
	Video_id string
	Err      error
}

type WiseReply struct {
	Reply    PostedReplyResp
	Video_id string
	Quote_id int64
}

func dropWisdom(replies []ReplyInfo, credentials Credentials) ([]WiseReply, time.Duration) {
	wisdom := []WiseReply{}
	err_resp := []error{}

	var wg sync.WaitGroup
	ch := make(chan ReplyStatus)
	done := make(chan bool)

	ts := time.Now()

	for _, item := range replies {
		wg.Add(1)
		go postReply(item, credentials, ch, &wg)
	}
	go func() {
		for {
			curr, next := <-ch
			if !next {
				done <- true
				continue
			}
			if curr.Err != nil {
				err_resp = append(err_resp, curr.Err)
				continue
			}
			wisdom = append(wisdom, WiseReply{Reply: curr.Resp, Video_id: curr.Video_id})
		}
	}()
	wg.Wait()
	close(ch)
	<-done

	logErrors(err_resp)
	return wisdom, time.Since(ts)
}
