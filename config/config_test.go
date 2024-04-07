package config_test

import (
	"testing"
	"time"

	"github.com/dymensionxyz/dymint/config"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestViperAndCobra(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)

	cmd := &cobra.Command{}
	config.AddNodeFlags(cmd)

	dir := t.TempDir()
	nc := config.DefaultConfig("", "")
	config.EnsureRoot(dir, nc)

	assert.NoError(cmd.Flags().Set(config.FlagAggregator, "true"))
	assert.NoError(cmd.Flags().Set(config.FlagDALayer, "foobar"))
	assert.NoError(cmd.Flags().Set(config.FlagDAConfig, `{"json":true}`))
	assert.NoError(cmd.Flags().Set(config.FlagBlockTime, "1234s"))
	assert.NoError(cmd.Flags().Set(config.FlagEmptyBlocksMaxTime, "2000s"))
	assert.NoError(cmd.Flags().Set(config.FlagBatchSubmitMaxTime, "3000s"))
	assert.NoError(cmd.Flags().Set(config.FlagNamespaceID, "0102030405060708"))
	assert.NoError(cmd.Flags().Set(config.FlagBlockBatchSize, "10"))

	assert.NoError(nc.GetViperConfig(cmd, dir))

	assert.Equal(true, nc.Aggregator)
	assert.Equal("foobar", nc.DALayer)
	assert.Equal(`{"json":true}`, nc.DAConfig)
	assert.Equal(1234*time.Second, nc.BlockTime)
	assert.Equal("0102030405060708", nc.NamespaceID)
	assert.Equal(uint64(10), nc.BlockBatchSize)
}

//TODO: check invalid config
