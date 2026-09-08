package main

import (
	"database/sql"
	"fmt"
	"time"
)

type Totals struct {
	Comments   int
	Replies    int
	Videos     int
	Quotes     int
	Channels   int
	QuotesUsed int
}

type Quota struct {
	Points    int
	Max       int
	Percent   int
	UpdatedAt string
	LastLogin string
	Stale     bool
}

type Day struct {
	Date     string
	Comments int
	Replies  int
	Total    int
	Percent  int
}

type Post struct {
	ID      string
	Likes   int
	Quote   string
	Author  string
	VideoID string
	Created string
}

type QuoteUse struct {
	Quote    string
	Author   string
	Uses     int
	Channels int
}

type Channel struct {
	Handle          string
	Title           string
	Frequency       int
	VideosSincePost int
	QuotesUsed      int
}

type Health struct {
	LastComment string
	LastReply   string
	DaysIdle    int
	Idle        bool
}

type Stats struct {
	Totals      Totals
	Quota       Quota
	Health      Health
	Activity    []Day
	TopComments []Post
	TopReplies  []Post
	TopQuotes   []QuoteUse
	UnusedPool  int
	Channels    []Channel
	Jobs        []Job
	BotOnline   bool
	BotError    string
	Generated   string
}

const quotaMax = 10000

func openDB(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", fmt.Sprintf("file:%s?mode=ro", path))
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		return nil, err
	}
	return db, nil
}

func scanCount(db *sql.DB, query string) (int, error) {
	var n int
	err := db.QueryRow(query).Scan(&n)
	return n, err
}

func loadTotals(db *sql.DB) (Totals, error) {
	var t Totals
	var err error

	if t.Comments, err = scanCount(db, "select count(*) from comments"); err != nil {
		return t, err
	}
	if t.Replies, err = scanCount(db, "select count(*) from replies"); err != nil {
		return t, err
	}
	if t.Videos, err = scanCount(db, "select count(*) from videos"); err != nil {
		return t, err
	}
	if t.Quotes, err = scanCount(db, "select count(*) from cornucopia"); err != nil {
		return t, err
	}
	if t.Channels, err = scanCount(db, "select count(*) from channels"); err != nil {
		return t, err
	}
	t.QuotesUsed, err = scanCount(db, "select count(distinct quote_id) from channel_quote_usage")
	return t, err
}

func loadQuota(db *sql.DB) (Quota, error) {
	var q Quota
	var updated, login sql.NullString

	err := db.QueryRow("select quota, updated_at from quota limit 1").Scan(&q.Points, &updated)
	if err != nil && err != sql.ErrNoRows {
		return q, err
	}
	if err := db.QueryRow("select last_login from login limit 1").Scan(&login); err != nil && err != sql.ErrNoRows {
		return q, err
	}
	q.Max = quotaMax
	q.UpdatedAt = prettyTime(updated.String)
	q.LastLogin = prettyTime(login.String)

	if q.Max > 0 {
		q.Percent = q.Points * 100 / q.Max
	}
	if ts, ok := parseTime(updated.String); ok {
		q.Stale = time.Since(ts) > 24*time.Hour
	}
	return q, nil
}

func loadHealth(db *sql.DB) (Health, error) {
	var h Health
	var comment, reply sql.NullString

	if err := db.QueryRow("select max(created_at) from comments").Scan(&comment); err != nil {
		return h, err
	}
	if err := db.QueryRow("select max(created_at) from replies").Scan(&reply); err != nil {
		return h, err
	}
	h.LastComment = prettyTime(comment.String)
	h.LastReply = prettyTime(reply.String)

	if ts, ok := parseTime(comment.String); ok {
		h.DaysIdle = int(time.Since(ts).Hours() / 24)
		h.Idle = h.DaysIdle > 2
	}
	return h, nil
}

func loadActivity(db *sql.DB, days int) ([]Day, error) {
	counts := map[string]*Day{}
	window := fmt.Sprintf("-%d days", days-1)

	for _, src := range []string{"comments", "replies"} {
		query := fmt.Sprintf("select date(created_at), count(*) from %s where date(created_at) >= date('now', ?) group by 1", src)
		rows, err := db.Query(query, window)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			var date string
			var n int
			if err := rows.Scan(&date, &n); err != nil {
				rows.Close()
				return nil, err
			}
			if counts[date] == nil {
				counts[date] = &Day{Date: date}
			}
			if src == "comments" {
				counts[date].Comments = n
			} else {
				counts[date].Replies = n
			}
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return nil, err
		}
		rows.Close()
	}

	activity := make([]Day, 0, days)
	peak := 0

	for i := days - 1; i >= 0; i-- {
		date := time.Now().AddDate(0, 0, -i).Format("2006-01-02")
		day := Day{Date: date}
		if found := counts[date]; found != nil {
			day.Comments = found.Comments
			day.Replies = found.Replies
		}
		day.Total = day.Comments + day.Replies
		if day.Total > peak {
			peak = day.Total
		}
		activity = append(activity, day)
	}
	for i := range activity {
		if peak > 0 {
			activity[i].Percent = activity[i].Total * 100 / peak
		}
	}
	return activity, nil
}

func loadTopPosts(db *sql.DB, table string, limit int) ([]Post, error) {
	query := fmt.Sprintf(`select p.id, p.likes, coalesce(c.quote, ''), coalesce(c.author, ''), p.created_at
		from %s p left join cornucopia c on c.id = p.quote_id
		order by p.likes desc, p.created_at desc limit ?`, table)

	rows, err := db.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	posts := []Post{}
	for rows.Next() {
		var p Post
		var created sql.NullString
		if err := rows.Scan(&p.ID, &p.Likes, &p.Quote, &p.Author, &created); err != nil {
			return nil, err
		}
		p.Created = prettyTime(created.String)
		posts = append(posts, p)
	}
	return posts, rows.Err()
}

func loadTopQuotes(db *sql.DB, limit int) ([]QuoteUse, error) {
	query := `select c.quote, c.author, count(*) as uses, count(distinct u.channel_id)
		from channel_quote_usage u join cornucopia c on c.id = u.quote_id
		group by u.quote_id order by uses desc, c.author limit ?`

	rows, err := db.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	quotes := []QuoteUse{}
	for rows.Next() {
		var q QuoteUse
		if err := rows.Scan(&q.Quote, &q.Author, &q.Uses, &q.Channels); err != nil {
			return nil, err
		}
		quotes = append(quotes, q)
	}
	return quotes, rows.Err()
}

func loadChannels(db *sql.DB) ([]Channel, error) {
	query := `select ch.handle, ch.title, ch.frequency, ch.videos_since_post,
		(select count(*) from channel_quote_usage u where u.channel_id = ch.id)
		from channels ch order by ch.handle`

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	channels := []Channel{}
	for rows.Next() {
		var c Channel
		if err := rows.Scan(&c.Handle, &c.Title, &c.Frequency, &c.VideosSincePost, &c.QuotesUsed); err != nil {
			return nil, err
		}
		channels = append(channels, c)
	}
	return channels, rows.Err()
}

func loadStats(db *sql.DB) (Stats, error) {
	var s Stats
	var err error

	if s.Totals, err = loadTotals(db); err != nil {
		return s, err
	}
	if s.Quota, err = loadQuota(db); err != nil {
		return s, err
	}
	if s.Health, err = loadHealth(db); err != nil {
		return s, err
	}
	if s.Activity, err = loadActivity(db, 30); err != nil {
		return s, err
	}
	if s.TopComments, err = loadTopPosts(db, "comments", 8); err != nil {
		return s, err
	}
	if s.TopReplies, err = loadTopPosts(db, "replies", 8); err != nil {
		return s, err
	}
	if s.TopQuotes, err = loadTopQuotes(db, 8); err != nil {
		return s, err
	}
	if s.Channels, err = loadChannels(db); err != nil {
		return s, err
	}
	s.UnusedPool = s.Totals.Quotes - s.Totals.QuotesUsed
	s.Generated = time.Now().Format("Mon 02 Jan 15:04:05")

	return s, nil
}
