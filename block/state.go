package block

import (
	"bytes"
	"errors"
	"fmt"

	errorsmod "cosmossdk.io/errors"

	"github.com/cometbft/cometbft/crypto/merkle"
	abci "github.com/tendermint/tendermint/abci/types"
	tmstate "github.com/tendermint/tendermint/proto/tendermint/state"
	tmversion "github.com/tendermint/tendermint/proto/tendermint/version"
	tmtypes "github.com/tendermint/tendermint/types"
	"github.com/tendermint/tendermint/version"

	"github.com/dymensionxyz/dymint/mempool"
	"github.com/dymensionxyz/dymint/store"
	"github.com/dymensionxyz/dymint/types"
)


func (m *Manager) LoadStateOnInit(store store.Store, genesis *tmtypes.GenesisDoc, logger types.Logger) error {
	s, err := store.LoadState()
	if errors.Is(err, types.ErrNoStateFound) {
		logger.Info("failed to find state in the store, creating new state from genesis")
		s, err = NewStateFromGenesis(genesis)
	}

	if err != nil {
		return fmt.Errorf("get initial state: %w", err)
	}

	m.State = s

	return nil
}



func NewStateFromGenesis(genDoc *tmtypes.GenesisDoc) (*types.State, error) {
	err := genDoc.ValidateAndComplete()
	if err != nil {
		return nil, fmt.Errorf("in genesis doc: %w", err)
	}

	
	
	
	
	InitStateVersion := tmstate.Version{
		Consensus: tmversion.Consensus{
			Block: version.BlockProtocol,
			App:   0,
		},
		Software: version.TMCoreSemVer,
	}

	s := types.State{
		Version:         InitStateVersion,
		ChainID:         genDoc.ChainID,
		InitialHeight:   uint64(genDoc.InitialHeight), 
		ConsensusParams: *genDoc.ConsensusParams,
	}
	s.SetHeight(0)
	copy(s.AppHash[:], genDoc.AppHash)

	err = s.SetRollappParamsFromGenesis(genDoc.AppState)
	if err != nil {
		return nil, fmt.Errorf("in genesis doc: %w", err)
	}

	return &s, nil
}


func (m *Manager) UpdateStateFromApp(blockHeaderHash [32]byte) error {
	proxyAppInfo, err := m.Executor.GetAppInfo()
	if err != nil {
		return errorsmod.Wrap(err, "get app info")
	}

	appHeight := uint64(proxyAppInfo.LastBlockHeight) 
	resp, err := m.Store.LoadBlockResponses(appHeight)
	if err != nil {
		return errorsmod.Wrap(err, "load block responses")
	}

	
	m.Executor.UpdateStateAfterCommit(m.State, resp, proxyAppInfo.LastBlockAppHash, appHeight, blockHeaderHash)

	return nil
}

func (e *Executor) UpdateStateAfterInitChain(s *types.State, res *abci.ResponseInitChain) {
	
	
	
	if len(res.AppHash) > 0 {
		copy(s.AppHash[:], res.AppHash)
	}
	if res.ConsensusParams != nil {
		params := res.ConsensusParams
		if params.Block != nil {
			s.ConsensusParams.Block.MaxBytes = params.Block.MaxBytes
			s.ConsensusParams.Block.MaxGas = params.Block.MaxGas
		}
	}
	
	copy(s.LastResultsHash[:], merkle.HashFromByteSlices(nil))
}

func (e *Executor) UpdateMempoolAfterInitChain(s *types.State) {
	e.mempool.SetPreCheckFn(mempool.PreCheckMaxBytes(s.ConsensusParams.Block.MaxBytes))
	e.mempool.SetPostCheckFn(mempool.PostCheckMaxGas(s.ConsensusParams.Block.MaxGas))
}


func (e *Executor) UpdateStateAfterCommit(s *types.State, resp *tmstate.ABCIResponses, appHash []byte, height uint64, lastHeaderHash [32]byte) {
	copy(s.AppHash[:], appHash[:])
	copy(s.LastResultsHash[:], tmtypes.NewResults(resp.DeliverTxs).Hash())
	copy(s.LastHeaderHash[:], lastHeaderHash[:])

	s.SetHeight(height)
	if resp.EndBlock.ConsensusParamUpdates != nil {
		s.ConsensusParams.Block.MaxGas = resp.EndBlock.ConsensusParamUpdates.Block.MaxGas
		s.ConsensusParams.Block.MaxBytes = resp.EndBlock.ConsensusParamUpdates.Block.MaxBytes
	}
	if resp.EndBlock.RollappParamUpdates != nil {
		s.RollappParams.Da = resp.EndBlock.RollappParamUpdates.Da
		s.RollappParams.DrsVersion = resp.EndBlock.RollappParamUpdates.DrsVersion
	}
}




func (e *Executor) UpdateProposerFromBlock(s *types.State, seqSet *types.SequencerSet, block *types.Block) bool {
	
	if bytes.Equal(block.Header.SequencerHash[:], block.Header.NextSequencersHash[:]) {
		return false
	}

	if block.Header.NextSequencersHash == [32]byte{} {
		
		
		e.logger.Info("rollapp left with no proposer. chain is halted")
		s.SetProposer(nil)
		return true
	}

	
	
	
	seq, found := seqSet.GetByHash(block.Header.NextSequencersHash[:])
	if !found {
		e.logger.Error("cannot find proposer by hash")
		panic("cannot find proposer by hash")
	}
	s.SetProposer(&seq)
	return true
}
