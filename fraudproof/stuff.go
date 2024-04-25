package fraudproof

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"os"

	dtypes "github.com/dymensionxyz/dymint/types"
	"github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/proxy"
)

type Phase = string

const (
	PhaseInit       Phase = "init"
	PhaseBeginBlock Phase = "beginBlock"
	PhaseDeliverTx  Phase = "deliverTx"
	PhaseEndBlock   Phase = "endBlock"
)

type ISR = []byte

type ISRCollector struct {
	ProxyAppConsensusConn proxy.AppConnConsensus
	Logger                dtypes.Logger
	Isrs                  []ISR
	SimulateFraud         bool
	err                   error
}

func (c *ISRCollector) CollectNext(p Phase) {
	if c.err != nil {
		return
	}
	hashRes, err := c.ProxyAppConsensusConn.GetAppHashSync(types.RequestGetAppHash{})
	if err != nil {
		c.err = fmt.Errorf("get app hash sync: %w", err)
		return
	}
	hash := hashRes.AppHash
	simulateFraud := p == PhaseDeliverTx && c.SimulateFraud && rand.Float64() < 0.5
	if simulateFraud {
		c.Logger.Info("simulating fraud", "phase", p)
		hash = bytes.Repeat([]byte("a"), 32)
	}
	c.Isrs = append(c.Isrs, hash)
	c.Logger.Debug("isr collected", "phase", p, "ISR", hex.EncodeToString(hash))
}

func (c *ISRCollector) Err() error {
	return c.err
}

type ISRVerifier struct {
	ProxyAppConsensusConn proxy.AppConnConsensus
	Logger                dtypes.Logger
	Isrs                  []ISR
	Ix                    int
	FraudProofsEnabled    bool
	err                   error
}

// VerifyNext returns if the next ISR is ok
func (v *ISRVerifier) VerifyNext() bool {
	if v.err != nil {
		return true // TODO: debate
	}
	hashRes, err := v.ProxyAppConsensusConn.GetAppHashSync(types.RequestGetAppHash{})
	if err != nil {
		v.err = fmt.Errorf("get app hash sync: %w", err)
		return true // TODO: debate
	}
	if v.FraudProofsEnabled && !bytes.Equal(hashRes.AppHash, v.Isrs[v.Ix]) {
		return false
	}
	v.Ix++ // TODO: I guess you don't need to do it only if FP are enabled, but worth a check
	return true
}

func (v *ISRVerifier) Err() error {
	return v.err
}

func Generate(proxyAppConsensusConn proxy.AppConnConsensus, beginBlockRequest *types.RequestBeginBlock, deliverTxRequests []*types.RequestDeliverTx, endBlockRequest *types.RequestEndBlock) (*types.FraudProof, error) {
	generateFraudProofRequest := types.RequestGenerateFraudProof{}
	if beginBlockRequest == nil {
		return nil, fmt.Errorf("begin block request cannot be a nil parameter")
	}
	generateFraudProofRequest.BeginBlockRequest = *beginBlockRequest

	if deliverTxRequests != nil {
		generateFraudProofRequest.DeliverTxRequests = deliverTxRequests
	} else {
		panic("deliverTxRequests cannot be nil - fraudulent begin block not supported")
	}
	if endBlockRequest != nil {
		generateFraudProofRequest.EndBlockRequest = endBlockRequest
		panic("fraudulent endBlockRequest not supported")
	}

	resp, err := proxyAppConsensusConn.GenerateFraudProofSync(generateFraudProofRequest)
	if err != nil {
		return nil, err
	}

	fraud := resp.FraudProof
	if fraud == nil {
		return nil, errors.New("fraud proof is nil")
	}

	// Open a new file for writing only
	file, err := os.Create("fraudProof_rollapp_with_tx.json")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// Serialize the struct to JSON and write it to the file
	jsonEncoder := json.NewEncoder(file)
	err = jsonEncoder.Encode(fraud)
	if err != nil {
		return nil, err
	}

	panic("fraud proof generated")
}
