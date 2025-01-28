package celestia_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/celestiaorg/celestia-openrpc/types/blob"
	"github.com/celestiaorg/celestia-openrpc/types/header"
	uretry "github.com/dymensionxyz/dymint/utils/retry"

	"github.com/dymensionxyz/dymint/da"
	"github.com/dymensionxyz/dymint/da/celestia"
	mocks "github.com/dymensionxyz/dymint/mocks/github.com/dymensionxyz/dymint/da/celestia/types"
	"github.com/dymensionxyz/dymint/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/libs/log"

	"github.com/tendermint/tendermint/libs/pubsub"

	"github.com/celestiaorg/nmt"
)

const (
	submitPFBFuncName   = "Submit"
	getProofFuncName    = "GetProof"
	includedFuncName    = "Included"
	getByHeightFuncName = "GetByHeight"
)

// exampleNMT creates a new NamespacedMerkleTree with the given namespace ID size and leaf namespace IDs. Each byte in the leavesNIDs parameter corresponds to one leaf's namespace ID. If nidSize is greater than 1, the function repeats each NID in leavesNIDs nidSize times before prepending it to the leaf data.
func exampleNMT(nidSize int, ignoreMaxNamespace bool, leavesNIDs ...byte) *nmt.NamespacedMerkleTree {
	tree := nmt.New(sha256.New(), nmt.NamespaceIDSize(nidSize), nmt.IgnoreMaxNamespace(ignoreMaxNamespace))
	for i, nid := range leavesNIDs {
		namespace := bytes.Repeat([]byte{nid}, nidSize)
		d := append(namespace, []byte(fmt.Sprintf("leaf_%d", i))...)
		if err := tree.Push(d); err != nil {
			panic(fmt.Sprintf("unexpected error: %v", err))
		}
	}
	return tree
}

func TestSubmitBatch(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)
	configBytes, err := json.Marshal(celestia.TestConfig)
	require.NoError(err)
	batch := testutil.MustGenerateBatchAndKey(0, 1)
	nIDSize := 1

	tree := exampleNMT(nIDSize, true, 1, 2, 3, 4)

	// build a proof for an NID that is within the namespace range of the tree
	nID := []byte{1}
	proof, err := tree.ProveNamespace(nID)
	require.NoError(err)
	blobProof := blob.Proof([]*nmt.Proof{&proof})

	timeOutErr := errors.New("timeout")
	cases := []struct {
		name                    string
		submitPFBReturn         []interface{}
		submitPFBRun            func(args mock.Arguments)
		expectedInclusionHeight uint64
		getProofReturn          []interface{}
		getProofDRun            func(args mock.Arguments)
		includedReturn          []interface{}
		includedRun             func(args mock.Arguments)
	}{
		{
			name:                    "TestSubmitPFBResponseCodeSuccess",
			submitPFBReturn:         []interface{}{uint64(1234), nil},
			getProofReturn:          []interface{}{&blobProof, nil},
			includedReturn:          []interface{}{true, nil},
			submitPFBRun:            func(args mock.Arguments) { time.Sleep(10 * time.Millisecond) },
			getProofDRun:            func(args mock.Arguments) { time.Sleep(10 * time.Millisecond) },
			includedRun:             func(args mock.Arguments) { time.Sleep(10 * time.Millisecond) },
			expectedInclusionHeight: uint64(1234),
		},
		{
			name:            "TestSubmitPFBErrored",
			submitPFBReturn: []interface{}{uint64(0), timeOutErr},
			getProofReturn:  []interface{}{&blobProof, nil},
			includedReturn:  []interface{}{true, nil},
			submitPFBRun:    func(args mock.Arguments) { time.Sleep(10 * time.Millisecond) },
			getProofDRun:    func(args mock.Arguments) { time.Sleep(10 * time.Millisecond) },
			includedRun:     func(args mock.Arguments) { time.Sleep(10 * time.Millisecond) },
		},
	}
	for _, tc := range cases {
		t.Log("Case name ", tc.name)
		// Create mock clients
		mockRPCClient := mocks.NewMockCelestiaRPCClient(t)
		// Configure DALC options
		options := []da.Option{
			celestia.WithSubmitBackoff(uretry.NewBackoffConfig(uretry.WithInitialDelay(10*time.Millisecond), uretry.WithMaxDelay(10*time.Millisecond))),
			celestia.WithRPCClient(mockRPCClient),
			celestia.WithRPCAttempts(1),
		}
		// Subscribe to the health status event
		pubsubServer := pubsub.NewServer()
		err = pubsubServer.Start()
		require.NoError(err, tc.name)
		// Start the DALC
		dalc := celestia.DataAvailabilityLayerClient{}
		err = dalc.Init(configBytes, pubsubServer, nil, log.TestingLogger(), options...)
		require.NoError(err, tc.name)
		err = dalc.Start()
		require.NoError(err, tc.name)

		roots := [][]byte{[]byte("apple"), []byte("watermelon"), []byte("kiwi")}
		dah := &header.DataAvailabilityHeader{
			RowRoots:    roots,
			ColumnRoots: roots,
		}
		header := &header.ExtendedHeader{
			DAH: dah,
		}

		mockRPCClient.On(submitPFBFuncName, mock.Anything, mock.Anything, mock.Anything).Return(tc.submitPFBReturn...).Run(tc.submitPFBRun)
		if tc.name == "TestSubmitPFBResponseCodeSuccess" {
			mockRPCClient.On(getProofFuncName, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(tc.getProofReturn...).Run(tc.getProofDRun)
			mockRPCClient.On(includedFuncName, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(tc.includedReturn...).Run(tc.includedRun)
			mockRPCClient.On(getByHeightFuncName, mock.Anything, mock.Anything).Return(header, nil).Once().Run(func(args mock.Arguments) { time.Sleep(10 * time.Millisecond) })

		}
		done := make(chan bool)
		go func() {
			res := dalc.SubmitBatch(batch)
			if res.SubmitMetaData != nil {
				assert.Equal(res.SubmitMetaData.Height, tc.expectedInclusionHeight, tc.name)
			}
			done <- true
		}()

		time.Sleep(100 * time.Millisecond)
		err = dalc.Stop()
		require.NoError(err, tc.name)
		// Wait for the goroutines to finish before accessing the mock calls
		<-done
		assert.GreaterOrEqual(testutil.CountMockCalls(mockRPCClient.Calls, submitPFBFuncName), 1, tc.name)
	}
}
