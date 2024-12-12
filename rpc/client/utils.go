package client

import (
	"encoding/base64"
	"encoding/json"

	tmtypes "github.com/tendermint/tendermint/types"
)

const (
	// genesisChunkSize is the maximum size, in bytes, of each
	// chunk in the genesis structure for the chunked API
	genesisChunkSize = 16 * 1024 * 1024 // 16 MiB
)

// GetGenesisChunks returns chunked version of genesis.
func (c *Client) GetGenesisChunks() ([]string, error) {
	if c.genChunks != nil {
		return c.genChunks, nil
	}

	err := c.initGenesisChunks(c.node.GetGenesis())
	if err != nil {
		return nil, err
	}
	return c.genChunks, err
}

// initGenesisChunks creates a chunked format of the genesis document to make it easier to
// iterate through larger genesis structures.
func (c *Client) initGenesisChunks(genesis *tmtypes.GenesisDoc) error {
	if genesis == nil {
		return nil
	}

	data, err := json.Marshal(genesis)
	if err != nil {
		return err
	}

	for i := 0; i < len(data); i += genesisChunkSize {
		end := i + genesisChunkSize
		end = min(end, len(data))
		c.genChunks = append(c.genChunks, base64.StdEncoding.EncodeToString(data[i:end]))
	}

	return nil
}
