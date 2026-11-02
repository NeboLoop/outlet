package config

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zeromicro/go-zero/core/conf"
)

// The shipped config leaves the pool off and carries every pool number.
func TestPoolConfigDefaultsOff(t *testing.T) {
	t.Setenv("POOL_ENABLED", "")
	t.Setenv("POOL_BOUNCE_DOMAIN", "")
	t.Setenv("POOL_RESERVED_DOMAINS", "")
	raw, err := os.ReadFile("../../etc/outlet.yaml")
	require.NoError(t, err)
	var c Config
	require.NoError(t, conf.LoadFromYamlBytes([]byte(os.ExpandEnv(string(raw))), &c))

	assert.False(t, c.Pool.IsEnabled())
	assert.Equal(t, PoolConfig{
		RampStart: 5, RampStep: 2, Ceiling: 30, MinGapSeconds: 240, RestDays: 30,
		BouncePausePct: 3.0, BounceWindow: 100, BounceMinSample: 20,
		ComplaintPausePct: 0.1, ComplaintWindow: 1000, ComplaintMinCount: 1,
		PolicyBlocksPerDay: 3, ProviderFaultsToPause: 3, DomainPauseShare: 0.5,
		RepeatPauses: 2, RepeatPauseDays: 30,
		ReservedDomains: []string{"neboai.com", "nebo.bot", "neboloop.com", ""},
	}, c.Pool)

	t.Setenv("POOL_ENABLED", "true")
	var on Config
	require.NoError(t, conf.LoadFromYamlBytes([]byte(os.ExpandEnv(string(raw))), &on))
	assert.True(t, on.Pool.IsEnabled())
}

// The pool never sends as, or bounces to, a reserved domain or its
// subdomains; the deployment's own domain and base URL host are reserved too.
func TestPoolReservedDomains(t *testing.T) {
	var c Config
	c.App.BaseURL = "https://pool.outlet.test"
	c.App.Domain = "outlet.test"
	c.Pool.ReservedDomains = []string{"neboai.com", "nebo.bot", "neboloop.com", "send.brand.test, track.brand.test", ""}
	p := c.PoolSettings()
	for _, d := range []string{"neboai.com", "NEBOAI.COM.", "b.neboai.com", "nebo.bot", "x.y.nebo.bot", "neboloop.com",
		"send.brand.test", "a.track.brand.test", "outlet.test", "pool.outlet.test"} {
		assert.True(t, p.IsReserved(d), d)
	}
	for _, d := range []string{"", "notneboai.com", "neboai.co", "brand.test", "outreach-a.test"} {
		assert.False(t, p.IsReserved(d), d)
	}
	assert.Len(t, c.Pool.ReservedDomains, 5, "PoolSettings never changes the config it reads")

	c.Pool.Enabled = "true"
	c.Pool.BounceDomain = "bounce.neboai.com"
	assert.Contains(t, strings.Join(c.Validate(), "\n"), "POOL_BOUNCE_DOMAIN bounce.neboai.com is reserved")
	c.Pool.BounceDomain = "bounce.outreach-a.test"
	assert.Empty(t, c.Validate())
}
