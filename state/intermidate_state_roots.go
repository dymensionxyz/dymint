package state

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"os"

	"github.com/dymensionxyz/dymint/types"
	abci "github.com/tendermint/tendermint/abci/types"
)

func (e *BlockExecutor) getAppHash() ([]byte, error) {
	isrResp, err := e.proxyAppConsensusConn.GetAppHashSync(abci.RequestGetAppHash{})
	if err != nil {
		return nil, err
	}
	return isrResp.AppHash, nil
}

func (e *BlockExecutor) setOrVerifyISR(phase string, ISRs [][]byte, generateISR bool, idx int) ([][]byte, error) {
	isr, err := e.getAppHash()
	if err != nil {
		return nil, err
	}

	//sequencer mode
	if generateISR {
		e.logger.Info(phase, "ISR", hex.EncodeToString(isr))
		simulateFraud := phase == "deliverTx" && e.simulateFraud && rand.Float64() < 0.5
		if simulateFraud {
			e.logger.Info("simulating fraud", "phase", phase)
			isr = bytes.Repeat([]byte("a"), 32)
		}

		ISRs[idx] = isr
		return ISRs, nil
	}

	//verifier mode
	e.logger.Info("verifying ISR", "phase", phase)
	if e.fraudProofsEnabled && !bytes.Equal(isr, ISRs[idx]) {
		e.logger.Error(phase, "ISR mismatch", "ISR", hex.EncodeToString(isr), "expected", hex.EncodeToString(ISRs[idx]))
		return nil, types.ErrInvalidISR
	}

	return ISRs, nil
}

func (e *BlockExecutor) generateFraudProof(beginBlockRequest *abci.RequestBeginBlock, deliverTxRequests []*abci.RequestDeliverTx, endBlockRequest *abci.RequestEndBlock) (*abci.FraudProof, error) {
	generateFraudProofRequest := abci.RequestGenerateFraudProof{}
	if beginBlockRequest == nil {
		return nil, fmt.Errorf("begin block request cannot be a nil parameter")
	}
	generateFraudProofRequest.BeginBlockRequest = *beginBlockRequest

	if deliverTxRequests != nil {
		generateFraudProofRequest.DeliverTxRequests = deliverTxRequests
	} else {
		panic("deliverTxRequests cannot be nil - fraudelet begin block not supported")
	}
	if endBlockRequest != nil {
		generateFraudProofRequest.EndBlockRequest = endBlockRequest
		panic("fraudelent endBlockRequest not supported")
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
