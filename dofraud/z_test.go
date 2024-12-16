package dofraud

import (
	_ "embed"
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

// TODO: path
const fp = "/Users/danwt/Documents/dym/d-dymint/dofraud/testdata/example.json"

//go:embed testdata/example.json
var testData []byte

func TestGenerateJson(t *testing.T) {
	//t.Skip("Not a real test, just handy for quickly generating an example json.")

	examples := disk{
		Instances: []diskInstance{
			{Height: 10, Variants: "da,gossip", Block: diskBlock{HeaderProposerAddr: "1092381209381923809182391823098129038"}},
			{Height: 22, Variants: "da", Block: diskBlock{HeaderNextSequencerHash: "1092381209381923809182391823098129038"}},
		},
	}

	data, err := json.MarshalIndent(examples, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal examples: %v", err)
	}

	err = os.WriteFile(fp, data, 0644)
	if err != nil {
		t.Fatalf("Failed to write examples to file: %v", err)
	}
}

func TestParseJson(t *testing.T) {
	fraud, err := Load(fp)
	require.NoError(t, err)
	_, ok := fraud.frauds["10:1"]
	require.True(t, ok)
}
