package main

import (
	"database/sql"
	"fmt"
	"strings"
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
	Points        int
	Max           int
	Percent       int
	Margin        int
	MarginMax     int
	MarginPercent int
	Channels      int
	Cost          int
	Spendable     int
	Breached      bool
	UpdatedAt     string
	LastLogin     string
	Stale         bool
}

type Day struct {
	Date     string
	Comments int
	Replies  int
	Total    int
	Percent  int
	X        int
	W        int
	Y        int
}

type Wave struct {
	Line   string
	Width  int
	Height int
	Peak   int
	Mid    int
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
	LastPost    string
	DaysIdle    int
	Idle        bool
}

type TaskCounts struct {
	Pending int
	Running int
	Done    int
	Failed  int
	Total   int
}

type Task struct {
	ID       string
	Short    string
	Status   string
	VideoID  string
	Quote    string
	Author   string
	Attempts int
	Error    string
	Stamp    string
}

type Tasks struct {
	Counts TaskCounts
	Recent []Task
}

type Stats struct {
	Totals      Totals
	Quota       Quota
	Health      Health
	Activity    []Day
	Wave        Wave
	Tasks       Tasks
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

const (
	quotaMax         = 10000
	commentCost      = 50
	maxWebhookMargin = 2500
)

func webhookMargin(channels int) int {
	margin := channels * commentCost

	if margin > maxWebhookMargin {
		return maxWebhookMargin
	}
	return margin
}

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
	if q.Channels, err = scanCount(db, "select count(*) from channels"); err != nil {
		return q, err
	}
	q.Max = quotaMax
	q.MarginMax = maxWebhookMargin
	q.Margin = webhookMargin(q.Channels)
	q.Cost = commentCost
	q.UpdatedAt = prettyTime(updated.String)
	q.LastLogin = prettyTime(login.String)

	q.Spendable = q.Points - q.Margin
	q.Breached = q.Spendable < 0

	if q.Spendable < 0 {
		q.Spendable = 0
	}
	if q.Max > 0 {
		q.Percent = q.Points * 100 / q.Max
		q.MarginPercent = q.Margin * 100 / q.Max
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

	latest := time.Time{}

	for _, raw := range []string{comment.String, reply.String} {
		ts, ok := parseTime(raw)
		if ok && ts.After(latest) {
			latest = ts
		}
	}
	if latest.IsZero() {
		h.LastPost = "never"
		return h, nil
	}
	h.LastPost = prettyTime(latest.Format("2006-01-02 15:04:05"))
	h.DaysIdle = int(time.Since(latest).Hours() / 24)
	h.Idle = h.DaysIdle > 2

	return h, nil
}

func loadTaskCounts(db *sql.DB) (TaskCounts, error) {
	var counts TaskCounts

	rows, err := db.Query("select status, count(*) from tasks group by status")
	if err != nil {
		return counts, err
	}
	defer rows.Close()

	for rows.Next() {
		var status string
		var n int
		if err := rows.Scan(&status, &n); err != nil {
			return counts, err
		}
		switch status {
		case "pending":
			counts.Pending = n
		case "running":
			counts.Running = n
		case "done":
			counts.Done = n
		case "failed":
			counts.Failed = n
		}
		counts.Total += n
	}
	return counts, rows.Err()
}

func loadRecentTasks(db *sql.DB, limit int) ([]Task, error) {
	query := `select t.id, t.status, t.video_id, t.attempts,
		coalesce(c.quote, ''), coalesce(c.author, ''), coalesce(t.error, ''),
		coalesce(t.completed_at, t.claimed_at, t.active_at)
		from tasks t left join cornucopia c on c.id = t.quote_id
		order by t.created_at desc limit ?`

	rows, err := db.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tasks := []Task{}
	for rows.Next() {
		var t Task
		var stamp sql.NullString

		if err := rows.Scan(&t.ID, &t.Status, &t.VideoID, &t.Attempts, &t.Quote, &t.Author, &t.Error, &stamp); err != nil {
			return nil, err
		}
		t.Short = t.ID
		if len(t.Short) > 8 {
			t.Short = t.Short[:8]
		}
		t.Stamp = stamp.String
		tasks = append(tasks, t)
	}
	return tasks, rows.Err()
}

func loadTasks(db *sql.DB, limit int) (Tasks, error) {
	var t Tasks
	var err error

	if t.Counts, err = loadTaskCounts(db); err != nil {
		return t, err
	}
	t.Recent, err = loadRecentTasks(db, limit)
	return t, err
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

const (
	waveWidth  = 1200
	waveHeight = 200
	waveFloor  = 3
)

func buildWave(activity []Day) Wave {
	wave := Wave{Width: waveWidth, Height: waveHeight, Mid: waveHeight}

	if len(activity) == 0 {
		wave.Line = fmt.Sprintf("0,%d %d,%d", waveHeight, waveWidth, waveHeight)
		return wave
	}
	slot := float64(waveWidth) / float64(len(activity))
	points := make([]string, 0, len(activity)*3+2)
	points = append(points, fmt.Sprintf("0,%d", waveHeight))

	for i := range activity {
		day := &activity[i]
		left := float64(i) * slot

		day.X = int(left)
		day.W = int(slot) + 1

		spike := 0
		if day.Total > 0 {
			spike = day.Percent * (waveHeight - waveFloor) / 100
			if spike < waveFloor {
				spike = waveFloor
			}
		}
		day.Y = waveHeight - spike

		if day.Total > wave.Peak {
			wave.Peak = day.Total
		}
		points = append(points,
			fmt.Sprintf("%d,%d", int(left), waveHeight),
			fmt.Sprintf("%d,%d", int(left+slot/2), day.Y),
			fmt.Sprintf("%d,%d", int(left+slot), waveHeight),
		)
	}
	points = append(points, fmt.Sprintf("%d,%d", waveWidth, waveHeight))
	wave.Line = strings.Join(points, " ")

	return wave
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
	s.Wave = buildWave(s.Activity)

	if s.Tasks, err = loadTasks(db, 25); err != nil {
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
