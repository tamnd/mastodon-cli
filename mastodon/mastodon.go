// Package mastodon is the library behind the mastodon command line:
// the HTTP client, request shaping, and the typed data models for mastodon.social.
//
// All endpoints used here are public and require no authentication.
package mastodon

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Host is the Mastodon instance this client talks to.
const Host = "mastodon.social"

// BaseURL is the root every v1 request is built from.
const BaseURL = "https://" + Host + "/api/v1"

// htmlRE strips HTML tags from content fields.
var htmlRE = regexp.MustCompile(`<[^>]+>`)

func stripHTML(s string) string {
	s = htmlRE.ReplaceAllString(s, "")
	s = strings.Join(strings.Fields(s), " ")
	return strings.TrimSpace(s)
}

// --- Output types ---

// Tag is a trending hashtag.
type Tag struct {
	Name     string `json:"name"`
	URL      string `json:"url"`
	Accounts int    `json:"accounts"`
	Uses     int    `json:"uses"`
}

// Status is a Mastodon post/toot.
type Status struct {
	ID          string `json:"id"`
	CreatedAt   string `json:"created_at"`
	URL         string `json:"url"`
	Content     string `json:"content"`
	Author      string `json:"author"`
	DisplayName string `json:"display_name"`
	Reblogs     int    `json:"reblogs"`
	Favourites  int    `json:"favourites"`
	Replies     int    `json:"replies"`
	Language    string `json:"language"`
	Sensitive   bool   `json:"sensitive"`
}

// Link is a trending article/link shared on Mastodon.
type Link struct {
	URL         string `json:"url"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Provider    string `json:"provider"`
	Accounts    int    `json:"accounts"`
	Uses        int    `json:"uses"`
}

// Account is a Mastodon user profile.
type Account struct {
	ID          string `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	Note        string `json:"note"`
	URL         string `json:"url"`
	Followers   int    `json:"followers"`
	Following   int    `json:"following"`
	Statuses    int    `json:"statuses"`
	CreatedAt   string `json:"created_at"`
	Bot         bool   `json:"bot"`
}

// Instance holds Mastodon instance statistics.
type Instance struct {
	URI         string `json:"uri"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Users       int    `json:"users"`
	Statuses    int    `json:"statuses"`
	Domains     int    `json:"domains"`
	Version     string `json:"version"`
}

// --- Wire types (API response shapes) ---

type wireTag struct {
	Name    string `json:"name"`
	URL     string `json:"url"`
	History []struct {
		Day      string `json:"day"`
		Accounts string `json:"accounts"`
		Uses     string `json:"uses"`
	} `json:"history"`
}

type wireAccount struct {
	ID            string `json:"id"`
	Username      string `json:"username"`
	Acct          string `json:"acct"`
	DisplayName   string `json:"display_name"`
	Note          string `json:"note"`
	URL           string `json:"url"`
	FollowersCount int   `json:"followers_count"`
	FollowingCount int   `json:"following_count"`
	StatusesCount  int   `json:"statuses_count"`
	CreatedAt     string `json:"created_at"`
	Locked        bool   `json:"locked"`
	Bot           bool   `json:"bot"`
}

type wireStatus struct {
	ID          string      `json:"id"`
	CreatedAt   string      `json:"created_at"`
	URL         string      `json:"url"`
	Content     string      `json:"content"`
	Account     wireAccount `json:"account"`
	ReblogsCount   int     `json:"reblogs_count"`
	FavouritesCount int    `json:"favourites_count"`
	RepliesCount   int     `json:"replies_count"`
	Language    string      `json:"language"`
	Sensitive   bool        `json:"sensitive"`
}

type wireLink struct {
	URL         string `json:"url"`
	Title       string `json:"title"`
	Description string `json:"description"`
	ProviderName string `json:"provider_name"`
	History []struct {
		Day      string `json:"day"`
		Accounts string `json:"accounts"`
		Uses     string `json:"uses"`
	} `json:"history"`
}

type wireInstance struct {
	URI              string `json:"uri"`
	Title            string `json:"title"`
	Description      string `json:"description"`
	ShortDescription string `json:"short_description"`
	Stats            struct {
		UserCount   int `json:"user_count"`
		StatusCount int `json:"status_count"`
		DomainCount int `json:"domain_count"`
	} `json:"stats"`
	Version string `json:"version"`
}

// --- converters ---

func (w wireTag) toTag() *Tag {
	t := &Tag{Name: w.Name, URL: w.URL}
	if len(w.History) > 0 {
		t.Accounts, _ = strconv.Atoi(w.History[0].Accounts)
		t.Uses, _ = strconv.Atoi(w.History[0].Uses)
	}
	return t
}

func (w wireStatus) toStatus() *Status {
	return &Status{
		ID:          w.ID,
		CreatedAt:   w.CreatedAt,
		URL:         w.URL,
		Content:     stripHTML(w.Content),
		Author:      w.Account.Acct,
		DisplayName: w.Account.DisplayName,
		Reblogs:     w.ReblogsCount,
		Favourites:  w.FavouritesCount,
		Replies:     w.RepliesCount,
		Language:    w.Language,
		Sensitive:   w.Sensitive,
	}
}

func (w wireLink) toLink() *Link {
	l := &Link{
		URL:         w.URL,
		Title:       w.Title,
		Description: w.Description,
		Provider:    w.ProviderName,
	}
	if len(w.History) > 0 {
		l.Accounts, _ = strconv.Atoi(w.History[0].Accounts)
		l.Uses, _ = strconv.Atoi(w.History[0].Uses)
	}
	return l
}

func (w wireAccount) toAccount() *Account {
	return &Account{
		ID:          w.ID,
		Username:    w.Username,
		DisplayName: w.DisplayName,
		Note:        stripHTML(w.Note),
		URL:         w.URL,
		Followers:   w.FollowersCount,
		Following:   w.FollowingCount,
		Statuses:    w.StatusesCount,
		CreatedAt:   w.CreatedAt,
		Bot:         w.Bot,
	}
}

func (w wireInstance) toInstance() *Instance {
	desc := w.ShortDescription
	if desc == "" {
		desc = w.Description
	}
	return &Instance{
		URI:         w.URI,
		Title:       w.Title,
		Description: stripHTML(desc),
		Users:       w.Stats.UserCount,
		Statuses:    w.Stats.StatusCount,
		Domains:     w.Stats.DomainCount,
		Version:     w.Version,
	}
}

// --- Client ---

// Client talks to mastodon.social over HTTP.
type Client struct {
	HTTP      *http.Client
	UserAgent string
	Rate      time.Duration
	Retries   int

	last time.Time
}

// NewClient returns a Client with sensible defaults.
func NewClient() *Client {
	return &Client{
		HTTP:      &http.Client{Timeout: 15 * time.Second},
		UserAgent: "mastodon-cli/0.1 (tamnd87@gmail.com)",
		Rate:      500 * time.Millisecond,
		Retries:   3,
	}
}

// Get fetches a URL and returns the body bytes.
func (c *Client) Get(ctx context.Context, url string) ([]byte, error) {
	var lastErr error
	for attempt := 0; attempt <= c.Retries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff(attempt)):
			}
		}
		body, retry, err := c.do(ctx, url)
		if err == nil {
			return body, nil
		}
		lastErr = err
		if !retry {
			return nil, err
		}
	}
	return nil, fmt.Errorf("get %s: %w", url, lastErr)
}

func (c *Client) do(ctx context.Context, url string) (body []byte, retry bool, err error) {
	c.pace()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, false, err
	}
	req.Header.Set("User-Agent", c.UserAgent)
	req.Header.Set("Accept", "application/json")

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, true, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
		return nil, true, fmt.Errorf("http %d", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, false, fmt.Errorf("http %d", resp.StatusCode)
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, true, err
	}
	return b, false, nil
}

func (c *Client) pace() {
	if c.Rate <= 0 {
		return
	}
	if wait := c.Rate - time.Since(c.last); wait > 0 {
		time.Sleep(wait)
	}
	c.last = time.Now()
}

func backoff(attempt int) time.Duration {
	d := time.Duration(attempt) * 500 * time.Millisecond
	if d > 5*time.Second {
		d = 5 * time.Second
	}
	return d
}

// --- API methods ---

// Trends fetches trending hashtags.
func (c *Client) Trends(ctx context.Context, limit int) ([]*Tag, error) {
	url := fmt.Sprintf("%s/trends/tags?limit=%d", BaseURL, limit)
	body, err := c.Get(ctx, url)
	if err != nil {
		return nil, err
	}
	var wire []wireTag
	if err := json.Unmarshal(body, &wire); err != nil {
		return nil, fmt.Errorf("trends: %w", err)
	}
	out := make([]*Tag, 0, len(wire))
	for _, w := range wire {
		out = append(out, w.toTag())
	}
	return out, nil
}

// Posts fetches trending statuses.
func (c *Client) Posts(ctx context.Context, limit int) ([]*Status, error) {
	url := fmt.Sprintf("%s/trends/statuses?limit=%d", BaseURL, limit)
	body, err := c.Get(ctx, url)
	if err != nil {
		return nil, err
	}
	var wire []wireStatus
	if err := json.Unmarshal(body, &wire); err != nil {
		return nil, fmt.Errorf("posts: %w", err)
	}
	out := make([]*Status, 0, len(wire))
	for _, w := range wire {
		out = append(out, w.toStatus())
	}
	return out, nil
}

// Links fetches trending links/articles.
func (c *Client) Links(ctx context.Context, limit int) ([]*Link, error) {
	url := fmt.Sprintf("%s/trends/links?limit=%d", BaseURL, limit)
	body, err := c.Get(ctx, url)
	if err != nil {
		return nil, err
	}
	var wire []wireLink
	if err := json.Unmarshal(body, &wire); err != nil {
		return nil, fmt.Errorf("links: %w", err)
	}
	out := make([]*Link, 0, len(wire))
	for _, w := range wire {
		out = append(out, w.toLink())
	}
	return out, nil
}

// Timeline fetches statuses tagged with a hashtag.
func (c *Client) Timeline(ctx context.Context, hashtag string, limit int) ([]*Status, error) {
	hashtag = strings.TrimPrefix(hashtag, "#")
	url := fmt.Sprintf("%s/timelines/tag/%s?limit=%d", BaseURL, hashtag, limit)
	body, err := c.Get(ctx, url)
	if err != nil {
		return nil, err
	}
	var wire []wireStatus
	if err := json.Unmarshal(body, &wire); err != nil {
		return nil, fmt.Errorf("timeline: %w", err)
	}
	out := make([]*Status, 0, len(wire))
	for _, w := range wire {
		out = append(out, w.toStatus())
	}
	return out, nil
}

// Account fetches a user profile by username.
func (c *Client) Account(ctx context.Context, username string) (*Account, error) {
	url := fmt.Sprintf("%s/accounts/lookup?acct=%s", BaseURL, username)
	body, err := c.Get(ctx, url)
	if err != nil {
		return nil, err
	}
	var wire wireAccount
	if err := json.Unmarshal(body, &wire); err != nil {
		return nil, fmt.Errorf("account: %w", err)
	}
	return wire.toAccount(), nil
}

// Instance fetches instance statistics.
func (c *Client) Instance(ctx context.Context) (*Instance, error) {
	url := BaseURL + "/instance"
	body, err := c.Get(ctx, url)
	if err != nil {
		return nil, err
	}
	var wire wireInstance
	if err := json.Unmarshal(body, &wire); err != nil {
		return nil, fmt.Errorf("instance: %w", err)
	}
	return wire.toInstance(), nil
}
