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
	validDymint := "/ip4/127.0.0.1/tcp/1234"

	cases := []struct {
		name        string
		input       *tmcfg.Config
		expected    config.NodeConfig
		expectError bool
	}{
		{"empty", nil, config.NodeConfig{}, true},
		{"Seeds", &tmcfg.Config{P2P: &tmcfg.P2PConfig{Seeds: validCosmos + "," + validCosmos}}, config.NodeConfig{P2P: config.P2PConfig{Seeds: validDymint + "," + validDymint}}, false},
		//GetNodeConfig translates the listen address, so we expect the translated address
		{"ListenAddress", &tmcfg.Config{P2P: &tmcfg.P2PConfig{ListenAddress: validCosmos}}, config.NodeConfig{P2P: config.P2PConfig{ListenAddress: validDymint}}, false},
		{"RootDir", &tmcfg.Config{BaseConfig: tmcfg.BaseConfig{RootDir: "~/root"}}, config.NodeConfig{RootDir: "~/root"}, false},
		{"DBPath", &tmcfg.Config{BaseConfig: tmcfg.BaseConfig{DBPath: "./database"}}, config.NodeConfig{DBPath: "./database"}, false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var actual config.NodeConfig
			err := GetNodeConfig(&actual, c.input)
			if c.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, c.expected, actual)
			}
			assert.Equal(t, c.expected, actual)
		})
	}
}
