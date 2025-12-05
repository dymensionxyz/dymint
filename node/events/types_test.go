package events

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStringer(t *testing.T) {
	// Double checking :)

	t.Run("health: nil err", func(t *testing.T) {
		hs := DataHealthStatus{}
		s := fmt.Sprint(hs)
		require.Contains(t, s, "nil")
		t.Log(s)
	})
	t.Run("health: some err", func(t *testing.T) {
		text := "oops"
		hs := DataHealthStatus{Error: errors.New(text)}
		s := fmt.Sprint(hs)
		require.Contains(t, s, text)
	})
}
