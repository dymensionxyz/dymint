package config

import (
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestViperAndCobra(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)

	cmd := &cobra.Command{}
	AddNodeFlags(cmd)

	dir := t.TempDir()
	nc := DefaultConfig("", "")
	EnsureRoot(dir, nc)

	assert.NoError(cmd.Flags().Set(flagAggregator, "true"))
	assert.NoError(cmd.Flags().Set(flagDALayer, "foobar"))
	assert.NoError(cmd.Flags().Set(flagDAConfig, `{"json":true}`))
	assert.NoError(cmd.Flags().Set(flagBlockTime, "1234s"))
	assert.NoError(cmd.Flags().Set(flagEmptyBlocksMaxTime, "2000s"))
	assert.NoError(cmd.Flags().Set(flagBatchSubmitMaxTime, "3000s"))
	assert.NoError(cmd.Flags().Set(flagNamespaceID, "0102030405060708"))
	assert.NoError(cmd.Flags().Set(flagBlockBatchSize, "10"))

	assert.NoError(nc.GetViperConfig(cmd, dir))

	assert.Equal(true, nc.Aggregator)
	assert.Equal("foobar", nc.DALayer)
	assert.Equal(`{"json":true}`, nc.DAConfig)
	assert.Equal(1234*time.Second, nc.BlockTime)
	assert.Equal("0102030405060708", nc.NamespaceID)
	assert.Equal(uint64(10), nc.BlockBatchSize)
}

//TODO: check invalid config
