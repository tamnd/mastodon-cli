package mastodon

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func newTestClient(srv *httptest.Server) *Client {
	c := NewClient()
	c.Rate = 0
	c.Retries = 0
	// Override base URL by using the test server URL as the full URL
	// Tests call methods directly with constructed URLs or use the client.Get helper.
	c.HTTP = &http.Client{Timeout: 5 * time.Second}
	return c
}

func TestGet(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") == "" {
			t.Error("request carried no User-Agent")
		}
		_, _ = w.Write([]byte("ok"))
	}))
	defer srv.Close()

	c := NewClient()
	c.Rate = 0

	body, err := c.Get(context.Background(), srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "ok" {
		t.Errorf("body = %q, want %q", body, "ok")
	}
}

func TestGetRetriesOn503(t *testing.T) {
	var hits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		if hits < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		_, _ = w.Write([]byte("recovered"))
	}))
	defer srv.Close()

	c := NewClient()
	c.Rate = 0
	c.Retries = 5

	start := time.Now()
	body, err := c.Get(context.Background(), srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "recovered" {
		t.Errorf("body = %q after retries", body)
	}
	if hits != 3 {
		t.Errorf("server saw %d hits, want 3", hits)
	}
	if time.Since(start) < 500*time.Millisecond {
		t.Error("retries did not back off")
	}
}

func TestTrends(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]map[string]any{
			{
				"name": "programming",
				"url":  "https://mastodon.social/tags/programming",
				"history": []map[string]any{
					{"day": "1718150400", "accounts": "150", "uses": "200"},
				},
			},
		})
	}))
	defer srv.Close()

	c := newTestClient(srv)
	body, err := c.Get(context.Background(), srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	var wire []wireTag
	if err := json.Unmarshal(body, &wire); err != nil {
		t.Fatal(err)
	}
	if len(wire) != 1 {
		t.Fatalf("got %d tags, want 1", len(wire))
	}
	tag := wire[0].toTag()
	if tag.Name != "programming" {
		t.Errorf("Name = %q, want programming", tag.Name)
	}
	if tag.Accounts != 150 {
		t.Errorf("Accounts = %d, want 150", tag.Accounts)
	}
	if tag.Uses != 200 {
		t.Errorf("Uses = %d, want 200", tag.Uses)
	}
}

func TestStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]map[string]any{
			{
				"id":         "116752549645818192",
				"created_at": "2026-06-15T05:48:47.725Z",
				"url":        "https://mastodon.social/@user/116752549645818192",
				"content":    "<p>Hello <b>world</b></p>",
				"account": map[string]any{
					"id":           "1",
					"username":     "user",
					"acct":         "user",
					"display_name": "User",
				},
				"reblogs_count":    5,
				"favourites_count": 10,
				"replies_count":    2,
				"language":         "en",
				"sensitive":        false,
			},
		})
	}))
	defer srv.Close()

	c := newTestClient(srv)
	body, err := c.Get(context.Background(), srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	var wire []wireStatus
	if err := json.Unmarshal(body, &wire); err != nil {
		t.Fatal(err)
	}
	if len(wire) != 1 {
		t.Fatalf("got %d statuses, want 1", len(wire))
	}
	s := wire[0].toStatus()
	if s.Content != "Hello world" {
		t.Errorf("Content = %q, want %q", s.Content, "Hello world")
	}
	if s.Author != "user" {
		t.Errorf("Author = %q, want user", s.Author)
	}
	if s.Reblogs != 5 {
		t.Errorf("Reblogs = %d, want 5", s.Reblogs)
	}
	if s.Favourites != 10 {
		t.Errorf("Favourites = %d, want 10", s.Favourites)
	}
}

func TestLinks(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]map[string]any{
			{
				"url":           "https://example.com/article",
				"title":         "Article Title",
				"description":   "Short description",
				"provider_name": "example.com",
				"history": []map[string]any{
					{"day": "1718150400", "accounts": "50", "uses": "75"},
				},
			},
		})
	}))
	defer srv.Close()

	c := newTestClient(srv)
	body, err := c.Get(context.Background(), srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	var wire []wireLink
	if err := json.Unmarshal(body, &wire); err != nil {
		t.Fatal(err)
	}
	if len(wire) != 1 {
		t.Fatalf("got %d links, want 1", len(wire))
	}
	l := wire[0].toLink()
	if l.Title != "Article Title" {
		t.Errorf("Title = %q, want %q", l.Title, "Article Title")
	}
	if l.Provider != "example.com" {
		t.Errorf("Provider = %q, want example.com", l.Provider)
	}
	if l.Accounts != 50 {
		t.Errorf("Accounts = %d, want 50", l.Accounts)
	}
}

func TestAccount(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":              "1",
			"username":        "Gargron",
			"acct":            "Gargron",
			"display_name":    "Eugen Rochko",
			"note":            "<p>Founder of Mastodon</p>",
			"url":             "https://mastodon.social/@Gargron",
			"followers_count": 380820,
			"following_count": 2100,
			"statuses_count":  75000,
			"created_at":      "2016-03-16T00:00:00.000Z",
			"bot":             false,
		})
	}))
	defer srv.Close()

	c := newTestClient(srv)
	body, err := c.Get(context.Background(), srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	var wire wireAccount
	if err := json.Unmarshal(body, &wire); err != nil {
		t.Fatal(err)
	}
	acc := wire.toAccount()
	if acc.Username != "Gargron" {
		t.Errorf("Username = %q, want Gargron", acc.Username)
	}
	if acc.Note != "Founder of Mastodon" {
		t.Errorf("Note = %q, want %q", acc.Note, "Founder of Mastodon")
	}
	if acc.Followers != 380820 {
		t.Errorf("Followers = %d, want 380820", acc.Followers)
	}
}

func TestInstance(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"uri":               "mastodon.social",
			"title":             "Mastodon",
			"short_description": "A social network.",
			"stats": map[string]any{
				"user_count":   3308743,
				"status_count": 177866300,
				"domain_count": 25000,
			},
			"version": "4.6.0",
		})
	}))
	defer srv.Close()

	c := newTestClient(srv)
	body, err := c.Get(context.Background(), srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	var wire wireInstance
	if err := json.Unmarshal(body, &wire); err != nil {
		t.Fatal(err)
	}
	inst := wire.toInstance()
	if inst.URI != "mastodon.social" {
		t.Errorf("URI = %q, want mastodon.social", inst.URI)
	}
	if inst.Users != 3308743 {
		t.Errorf("Users = %d, want 3308743", inst.Users)
	}
	if inst.Version != "4.6.0" {
		t.Errorf("Version = %q, want 4.6.0", inst.Version)
	}
}

func TestStripHTML(t *testing.T) {
	cases := []struct{ in, want string }{
		{"<p>Hello world</p>", "Hello world"},
		{"<p>Founder of <b>Mastodon</b></p>", "Founder of Mastodon"},
		{"plain text", "plain text"},
		{"", ""},
	}
	for _, tc := range cases {
		got := stripHTML(tc.in)
		if got != tc.want {
			t.Errorf("stripHTML(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
