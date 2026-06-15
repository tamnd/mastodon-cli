package mastodon

import (
	"context"

	"github.com/tamnd/any-cli/kit"
)

func init() { kit.Register(Domain{}) }

// Domain is the mastodon driver for the kit framework.
type Domain struct{}

// Info describes the scheme and identity for this domain.
func (Domain) Info() kit.DomainInfo {
	return kit.DomainInfo{
		Scheme: "mastodon",
		Hosts:  []string{Host},
		Identity: kit.Identity{
			Binary: "mastodon",
			Short:  "A command line for Mastodon social network.",
			Long: `A command line for Mastodon social network.

mastodon reads public data from mastodon.social over plain HTTPS and shapes
it into clean records. No API key required.`,
			Site: "https://" + Host,
			Repo: "https://github.com/tamnd/mastodon-cli",
		},
	}
}

// Register installs the client factory and every operation onto app.
func (Domain) Register(app *kit.App) {
	app.SetClient(newClient)

	kit.Handle(app, kit.OpMeta{Name: "trends", Group: "read", List: true,
		Summary: "Trending hashtags on Mastodon"}, listTrends)

	kit.Handle(app, kit.OpMeta{Name: "posts", Group: "read", List: true,
		Summary: "Trending posts on Mastodon"}, listPosts)

	kit.Handle(app, kit.OpMeta{Name: "links", Group: "read", List: true,
		Summary: "Trending links and articles on Mastodon"}, listLinks)

	kit.Handle(app, kit.OpMeta{Name: "timeline", Group: "read", List: true,
		Summary: "Posts tagged with a hashtag",
		Args:    []kit.Arg{{Name: "hashtag", Help: "hashtag to search (without #)"}}}, listTimeline)

	kit.Handle(app, kit.OpMeta{Name: "account", Group: "read", Single: true,
		Summary: "Get a Mastodon user profile",
		Args:    []kit.Arg{{Name: "username", Help: "username or user@instance.social"}}}, getAccount)

	kit.Handle(app, kit.OpMeta{Name: "instance", Group: "read", Single: true,
		Summary: "Show Mastodon instance statistics"}, getInstance)
}

// newClient builds the client from the host-resolved config.
func newClient(_ context.Context, cfg kit.Config) (any, error) {
	c := NewClient()
	if cfg.UserAgent != "" {
		c.UserAgent = cfg.UserAgent
	}
	if cfg.Rate > 0 {
		c.Rate = cfg.Rate
	}
	if cfg.Retries > 0 {
		c.Retries = cfg.Retries
	}
	if cfg.Timeout > 0 {
		c.HTTP.Timeout = cfg.Timeout
	}
	return c, nil
}

// --- input structs ---

type listInput struct {
	Limit  int     `kit:"flag,inherit" help:"max results"`
	Client *Client `kit:"inject"`
}

type timelineInput struct {
	Hashtag string  `kit:"arg" help:"hashtag (without #)"`
	Limit   int     `kit:"flag,inherit" help:"max results"`
	Client  *Client `kit:"inject"`
}

type accountInput struct {
	Username string  `kit:"arg" help:"username or user@instance"`
	Client   *Client `kit:"inject"`
}

type instanceInput struct {
	Client *Client `kit:"inject"`
}

// --- handlers ---

func listTrends(ctx context.Context, in listInput, emit func(*Tag) error) error {
	limit := in.Limit
	if limit <= 0 {
		limit = 20
	}
	tags, err := in.Client.Trends(ctx, limit)
	if err != nil {
		return err
	}
	for _, t := range tags {
		if err := emit(t); err != nil {
			return err
		}
	}
	return nil
}

func listPosts(ctx context.Context, in listInput, emit func(*Status) error) error {
	limit := in.Limit
	if limit <= 0 {
		limit = 20
	}
	statuses, err := in.Client.Posts(ctx, limit)
	if err != nil {
		return err
	}
	for _, s := range statuses {
		if err := emit(s); err != nil {
			return err
		}
	}
	return nil
}

func listLinks(ctx context.Context, in listInput, emit func(*Link) error) error {
	limit := in.Limit
	if limit <= 0 {
		limit = 20
	}
	links, err := in.Client.Links(ctx, limit)
	if err != nil {
		return err
	}
	for _, l := range links {
		if err := emit(l); err != nil {
			return err
		}
	}
	return nil
}

func listTimeline(ctx context.Context, in timelineInput, emit func(*Status) error) error {
	limit := in.Limit
	if limit <= 0 {
		limit = 20
	}
	statuses, err := in.Client.Timeline(ctx, in.Hashtag, limit)
	if err != nil {
		return err
	}
	for _, s := range statuses {
		if err := emit(s); err != nil {
			return err
		}
	}
	return nil
}

func getAccount(ctx context.Context, in accountInput, emit func(*Account) error) error {
	acc, err := in.Client.Account(ctx, in.Username)
	if err != nil {
		return err
	}
	return emit(acc)
}

func getInstance(ctx context.Context, in instanceInput, emit func(*Instance) error) error {
	inst, err := in.Client.Instance(ctx)
	if err != nil {
		return err
	}
	return emit(inst)
}
