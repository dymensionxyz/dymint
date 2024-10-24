package queue_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/dymensionxyz/dymint/utils/queue"
)

func TestDequeue(t *testing.T) {
	testCases := []struct {
		name     string
		setup    func() *queue.Queue[int]
		expected []int
	}{
		{
			name: "DequeueAll",
			setup: func() *queue.Queue[int] {
				q := queue.New[int]()
				q.Enqueue(1, 2, 3)
				return q
			},
			expected: []int{1, 2, 3},
		},
		{
			name: "DequeueAll empty",
			setup: func() *queue.Queue[int] {
				return queue.New[int]()
			},
			expected: []int{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			q := tc.setup()

			// dequeue all
			actual := q.DequeueAll()

			// check the results
			require.Equal(t, tc.expected, actual)
			require.Equal(t, 0, q.Size())

			// dequeue all once again
			actualEmpty := q.DequeueAll()
			require.Equal(t, []int{}, actualEmpty)
			require.Equal(t, 0, q.Size())

			// add new elements to the queue
			q.Enqueue(100, 101, 102)

			// check that actual is not changed
			require.Equal(t, tc.expected, actual)
			require.Equal(t, 3, q.Size())
		})
	}
}
