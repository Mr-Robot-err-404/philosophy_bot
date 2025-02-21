package main

import (
	"sync"
	"time"
)

type WiseReply struct {
	Resp PostedReplyResp
	Err  error
}

func dropWisdom(replies []ReplyPayload, credentials Credentials) ([]PostedReplyResp, time.Duration) {
	wisdom := []PostedReplyResp{}
	err_resp := []error{}

	var wg sync.WaitGroup
	ch := make(chan WiseReply)
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
			wisdom = append(wisdom, curr.Resp)
		}
	}()
	wg.Wait()
	close(ch)
	<-done

	logErrors(err_resp)
	return wisdom, time.Since(ts)
}
