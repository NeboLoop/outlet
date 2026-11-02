package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zeromicro/go-zero/core/conf"
)

// The shipped config leaves the pool off and carries every pool number.
func TestPoolConfigDefaultsOff(t *testing.T) {
	t.Setenv("POOL_ENABLED", "")
	t.Setenv("POOL_BOUNCE_DOMAIN", "")
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
	}, c.Pool)

	t.Setenv("POOL_ENABLED", "true")
	var on Config
	require.NoError(t, conf.LoadFromYamlBytes([]byte(os.ExpandEnv(string(raw))), &on))
	assert.True(t, on.Pool.IsEnabled())
}
