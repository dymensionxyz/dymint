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
			{Height: 33, Cmd: diskCmd{HeaderProposerAddr: "1092381209381923809182391823098129038"}},
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
