package state

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"os"

	"github.com/dymensionxyz/dymint/log"

	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/proxy"
)

type phase = string

const (
	phaseInit       phase = "init"
	phaseBeginBlock phase = "beginBlock"
	phaseDeliverTx  phase = "deliverTx"
	phaseEndBlock   phase = "endBlock"
)

type ISR = []byte

type ISRCollector struct {
	proxyAppConsensusConn proxy.AppConnConsensus
	logger                log.Logger
	isrs                  []ISR
	simulateFraud         bool
	err                   error
}

func (c *ISRCollector) CollectNext(p phase) {
	if c.err != nil {
		return
	}
	hashRes, err := c.proxyAppConsensusConn.GetAppHashSync(abci.RequestGetAppHash{})
	if err != nil {
		c.err = fmt.Errorf("get app hash sync: %w", err)
	}
	hash := hashRes.AppHash
	simulateFraud := p == phaseDeliverTx && c.simulateFraud && rand.Float64() < 0.5
	if simulateFraud {
		c.logger.Info("simulating fraud", "phase", p)
		hash = bytes.Repeat([]byte("a"), 32)
	}
	c.isrs = append(c.isrs, hash)
	c.logger.Debug("isr collected", "phase", p, "ISR", hex.EncodeToString(hash))
}

type ISRVerifier struct {
	proxyAppConsensusConn proxy.AppConnConsensus
	logger                log.Logger
	isrs                  []ISR
	ix                    int
	fraudProofsEnabled    bool
	err                   error
}

// VerifyNext returns if the next ISR is ok
func (v *ISRVerifier) VerifyNext() bool {
	if v.err != nil {
		return true // TODO: debate
	}
	hashRes, err := v.proxyAppConsensusConn.GetAppHashSync(abci.RequestGetAppHash{})
	if err != nil {
		v.err = fmt.Errorf("get app hash sync: %w", err)
	}
	if v.fraudProofsEnabled && !bytes.Equal(hashRes.AppHash, v.isrs[v.ix]) {
		return false
	}
	v.ix++ // TODO: I guess you don't need to do it only if FP are enabled, but worth a check
	return true
}

func (e *BlockExecutor) generateFraudProof(
	beginBlockRequest *abci.RequestBeginBlock,
	deliverTxRequests []*abci.RequestDeliverTx,
	endBlockRequest *abci.RequestEndBlock,
) (*abci.FraudProof, error) {
	generateFraudProofRequest := abci.RequestGenerateFraudProof{}
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

	resp, err := e.proxyAppConsensusConn.GenerateFraudProofSync(generateFraudProofRequest)
	if err != nil {
		return nil, err
	}

	fraud := resp.FraudProof
	if fraud == nil {
		e.logger.Error("fraud proof is nil")
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
