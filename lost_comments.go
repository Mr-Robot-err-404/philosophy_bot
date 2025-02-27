package main

import (
	"os"
	"sync"
)

type LostSheep struct {
	reply_id string
	video_id string
}

func generateCSV(lost_sheep []LostSheep) []byte {
	csv := "reply,video\n"
	for _, sheep := range lost_sheep {
		csv += sheep.reply_id + "," + sheep.video_id
		csv += "\n"
	}
	return []byte(csv)
}

func writeCSV(csv []byte) error {
	file, err := os.Create("padawan.csv")
	if err != nil {
		return err
	}
	_, err = file.Write(csv)
	if err != nil {
		return err
	}
	return nil
}

func locateLostSheep(results []ThreadItem, pair_map map[string]string, video_id string) []LostSheep {
	lost_sheep := []LostSheep{}

	for _, curr := range results {
		parent_id := curr.Id
		reply_id, exists := pair_map[parent_id]
		if !exists {
			continue
		}
		lost_sheep = append(lost_sheep, LostSheep{reply_id: reply_id, video_id: video_id})
	}
	return lost_sheep
}

func getPairMap() (map[string]string, error) {
	pair_map := make(map[string]string)

	replies, err := queries.GetReplies(ctx)
	if err != nil {
		return pair_map, err
	}
	for _, item := range replies {
		id := item.ID
		idx := idxOf(id, '.')

		parent_id := id[0:idx]
		pair_map[parent_id] = item.ID
	}
	return pair_map, nil
}

func findLostComments(videos []string, key string, pair_map map[string]string) []LostSheep {
	lost_sheep := []LostSheep{}
	err_resp := []error{}

	var wg sync.WaitGroup
	ch := make(chan ThreadSearch)
	done := make(chan bool)

	for _, vid := range videos {
		wg.Add(1)
		go findCommentThread(vid, key, ch, &wg)
	}
	go func() {
		for {
			resp, next := <-ch
			if !next {
				done <- true
				continue
			}
			if resp.Err != nil {
				err_resp = append(err_resp, resp.Err)
				continue
			}
			lost := locateLostSheep(resp.Results, pair_map, resp.VideoId)
			lost_sheep = append(lost_sheep, lost...)
		}
	}()
	wg.Wait()
	close(ch)
	<-done

	return lost_sheep
}
