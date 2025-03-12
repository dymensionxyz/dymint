package sui_test

import (
	"context"
	"os"
	"testing"

	"github.com/block-vision/sui-go-sdk/mystenbcs"
	"github.com/dymensionxyz/dymint/da/sui"
	"github.com/stretchr/testify/require"
)

func TestClient(t *testing.T) {
	mnemonicEnv := "SUI_MNEMONIC"
	err := os.Setenv(mnemonicEnv, "catalog phone awful abuse derive type verb betray foil salad street scrub")
	require.NoError(t, err)

	client, err := sui.NewClient("https://fullnode.devnet.sui.io", mnemonicEnv)
	require.NoError(t, err)

	ctx := context.Background()
	err = client.TestMoveCall(ctx)
	require.NoError(t, err)
}

func TestDecerializePureArg(t *testing.T) {
	expected := "A0FBQQ=="

	input := "CEEwRkJRUT09"
	base64Decoded, err := mystenbcs.FromBase64(input)
	require.NoError(t, err)

	var value string
	_, err = mystenbcs.Unmarshal(base64Decoded, &value)
	require.NoError(t, err)

	require.Equal(t, expected, value)
}
