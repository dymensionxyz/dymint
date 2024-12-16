package dofraud

import (
	"encoding/json"
	"os"
	"testing"
)

func TestDoFraud(t *testing.T) {
	t.Skip("Not a real test, just handy for quickly generating an example json.")
	const fn = "/Users/danwt/Documents/dym/aaa-dym-notes/all_tasks/tasks/202412_testing_playground/ts.json"

	// Generate a few example diskPairs
	examples := disk{
		Instances: []diskInstance{
			{Height: 10, Block: diskBlock{HeaderProposerAddr: "1092381209381923809182391823098129038"}},
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
