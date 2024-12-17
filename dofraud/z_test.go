package dofraud

import (
	_ "embed"
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

const fp = "/Users/danwt/Documents/dym/d-dymint/dofraud/testdata/example.json"

//go:embed testdata/example.json
var testData []byte

func TestGenerateJson(t *testing.T) {
	t.Skip("Not a real test, just handy for quickly generating an example json.")

	examples := disk{
		Instances: []diskInstance{
			{Height: 10, Variants: "da,gossip", Block: diskBlock{HeaderProposerAddr: "1092381209381923809182391823098129038"}},
			{Height: 44, Variants: "da,gossip,produce", Block: diskBlock{
				HeaderNextSequencerHash: "1092381209381923809182391823098129038",
				HeaderDataHash:          "1290830918230981239812903819823909123",
				HeaderLastResultsHash:   "129038120938120938120938120938120938",
			}},
			{Height: 105, Variants: "gossip", Block: diskBlock{
				HeaderNextSequencerHash: "1092381209381923809182391823098129038",
				HeaderTime:              1734374656000000000,
			}},
			{Height: 120, Variants: "gossip", Block: diskBlock{
				HeaderVersionApp: 3,
			}},
		},
	}

	data, err := json.MarshalIndent(examples, "", "  ")
	require.NoError(t, err)

	err = os.WriteFile(fp, data, 0o644)
	require.NoError(t, err)
}

func TestParseJson(t *testing.T) {
	fraud, err := Load(fp)
	require.NoError(t, err)
	cmd, ok := fraud.frauds[key{10, DA}.String()]
	require.True(t, ok)
	require.Contains(t, cmd.ts, HeaderProposerAddr)
}
