package conv

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/dymensionxyz/dymint/config"
	tmcfg "github.com/tendermint/tendermint/config"
)

func TestGetNodeConfig(t *testing.T) {
	t.Parallel()

	validCosmos := "127.0.0.1:1234"
	validOptimint := "/ip4/127.0.0.1/tcp/1234"

	cases := []struct {
		name     string
		input    *tmcfg.Config
		expected config.NodeConfig
	}{
		{"empty", nil, config.NodeConfig{}},
		{"Seeds", &tmcfg.Config{P2P: &tmcfg.P2PConfig{Seeds: "seeds"}}, config.NodeConfig{P2P: config.P2PConfig{Seeds: "seeds"}}},
		//GetNodeConfig translates the listen address, so we expect the translated address
		{"ListenAddress", &tmcfg.Config{P2P: &tmcfg.P2PConfig{ListenAddress: validCosmos}}, config.NodeConfig{P2P: config.P2PConfig{ListenAddress: validOptimint}}},
		{"RootDir", &tmcfg.Config{BaseConfig: tmcfg.BaseConfig{RootDir: "~/root"}}, config.NodeConfig{RootDir: "~/root"}},
		{"DBPath", &tmcfg.Config{BaseConfig: tmcfg.BaseConfig{DBPath: "./database"}}, config.NodeConfig{DBPath: "./database"}},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var actual config.NodeConfig
			GetNodeConfig(&actual, c.input)
			assert.Equal(t, c.expected, actual)
		})
	}
}
