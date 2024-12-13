package dofraud

import (
	"encoding/json"
	"os"
	"testing"
)

func TestDoFraud(t *testing.T) {
	const fn = "/Users/danwt/Documents/dym/aaa-dym-notes/all_tasks/tasks/202412_testing_playground/frauds.json"

	// Generate a few example diskPairs
	examples := disk{
		Pairs: []diskPair{
			{Height: 10, Cmd: diskCmd{HeaderVersionBlock: 1, HeaderChainID: "chain-1"}},
			{Height: 20, Cmd: diskCmd{HeaderVersionApp: 2, HeaderHeight: 100}},
			{Height: 30, Cmd: diskCmd{HeaderTime: 1234567890, HeaderDataHash: "datahash"}},
			{Height: 40, Cmd: diskCmd{HeaderNextSequencerHash: "foobar", HeaderDataHash: "datahash"}},
		},
	}

	// Marshal the examples to JSON
	data, err := json.MarshalIndent(examples, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal examples: %v", err)
	}

	// Write the JSON data to the specified file
	err = os.WriteFile(fn, data, 0644)
	if err != nil {
		t.Fatalf("Failed to write examples to file: %v", err)
	}
}
