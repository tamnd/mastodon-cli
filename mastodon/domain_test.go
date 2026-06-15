package mastodon

import (
	"testing"
)

func TestDomainInfo(t *testing.T) {
	info := Domain{}.Info()
	if info.Scheme != "mastodon" {
		t.Errorf("Scheme = %q, want mastodon", info.Scheme)
	}
	if len(info.Hosts) == 0 || info.Hosts[0] != Host {
		t.Errorf("Hosts = %v, want [%s]", info.Hosts, Host)
	}
	if info.Identity.Binary != "mastodon" {
		t.Errorf("Identity.Binary = %q, want mastodon", info.Identity.Binary)
	}
}
