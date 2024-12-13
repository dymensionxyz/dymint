package dofraud

import (
	"encoding/json"
	"io"
	"os"

	"github.com/dymensionxyz/dymint/types"
)

type diskPair struct {
	Height uint64
	Cmd    diskCmd
}

type disk struct {
	Pairs []diskPair
}

type diskCmd struct {
	HeaderVersionBlock      uint64   `json:",omitempty"`
	HeaderVersionApp        uint64   `json:",omitempty"`
	HeaderChainID           string   `json:",omitempty"`
	HeaderHeight            uint64   `json:",omitempty"`
	HeaderTime              int64    `json:",omitempty"`
	HeaderLastHeaderHash    string   `json:",omitempty"`
	HeaderDataHash          string   `json:",omitempty"`
	HeaderConsensusHash     string   `json:",omitempty"`
	HeaderAppHash           string   `json:",omitempty"`
	HeaderLastResultsHash   string   `json:",omitempty"`
	HeaderProposerAddr      string   `json:",omitempty"`
	HeaderLastCommitHash    string   `json:",omitempty"`
	HeaderSequencerHash     string   `json:",omitempty"`
	HeaderNextSequencerHash string   `json:",omitempty"`
	Data                    struct{} `json:",omitempty"` // TODO:
	LastCommit              struct{} `json:",omitempty"` // TODO:
}

func Load(fn string) (Cmds, error) {
	file, err := os.Open(fn)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	var d disk
	err = json.Unmarshal(data, &d)
	if err != nil {
		return nil, err
	}

	frauds := []FraudType{}
	for _, pair := range d.Pairs {
		frauds := []FraudType{}
		if pair.Cmd.HeaderVersionBlock != 0 {
			frauds = append(frauds, HeaderVersionBlock)
		}
		if pair.Cmd.HeaderVersionApp != 0 {
			frauds = append(frauds, HeaderVersionApp)
		}
		if pair.Cmd.HeaderChainID != "" {
			frauds = append(frauds, HeaderChainID)
		}
		if pair.Cmd.HeaderHeight != 0 {
			frauds = append(frauds, HeaderHeight)
		}
		if pair.Cmd.HeaderTime != 0 {
			frauds = append(frauds, HeaderTime)
		}
		if pair.Cmd.HeaderLastHeaderHash != "" {
			frauds = append(frauds, HeaderLastHeaderHash)
		}
		if pair.Cmd.HeaderDataHash != "" {
			frauds = append(frauds, HeaderDataHash)
		}
		if pair.Cmd.HeaderConsensusHash != "" {
			frauds = append(frauds, HeaderConsensusHash)
		}
		if pair.Cmd.HeaderAppHash != "" {
			frauds = append(frauds, HeaderAppHash)
		}
		if pair.Cmd.HeaderLastResultsHash != "" {
			frauds = append(frauds, HeaderLastResultsHash)
		}
		if pair.Cmd.HeaderProposerAddr != "" {
			frauds = append(frauds, HeaderProposerAddr)
		}
		if pair.Cmd.HeaderLastCommitHash != "" {
			frauds = append(frauds, HeaderLastCommitHash)
		}
		if pair.Cmd.HeaderSequencerHash != "" {
			frauds = append(frauds, HeaderSequencerHash)
		}
		if pair.Cmd.HeaderNextSequencerHash != "" {
			frauds = append(frauds, HeaderNextSequencerHash)
		}
		// TODO: Data and LastCommit
	}

	cmds := make(Cmds)
	for _, pair := range d.Pairs {
		cmd := Cmd{
			Block: &types.Block{
				Header: types.Header{
					Version: types.Version{
						Block: pair.Cmd.HeaderVersionBlock,
						App:   pair.Cmd.HeaderVersionApp,
					},
					ChainID:            pair.Cmd.HeaderChainID,
					Height:             pair.Cmd.HeaderHeight,
					Time:               pair.Cmd.HeaderTime,
					LastHeaderHash:     parseHash(pair.Cmd.HeaderLastHeaderHash),
					DataHash:           parseHash(pair.Cmd.HeaderDataHash),
					ConsensusHash:      parseHash(pair.Cmd.HeaderConsensusHash),
					AppHash:            parseHash(pair.Cmd.HeaderAppHash),
					LastResultsHash:    parseHash(pair.Cmd.HeaderLastResultsHash),
					ProposerAddress:    []byte(pair.Cmd.HeaderProposerAddr),
					LastCommitHash:     parseHash(pair.Cmd.HeaderLastCommitHash),
					SequencerHash:      parseHash(pair.Cmd.HeaderSequencerHash),
					NextSequencersHash: parseHash(pair.Cmd.HeaderNextSequencerHash),
				},
				// Data and LastCommit fields need to be populated as per your requirements
			},
			frauds: frauds,
		}
		cmds[pair.Height] = cmd
	}

	return cmds, nil
}

func parseHash(hashStr string) [32]byte {
	var hash [32]byte
	copy(hash[:], hashStr)
	return hash
}

type FraudType int

const (
	None = iota
	HeaderVersionBlock
	HeaderVersionApp
	HeaderChainID
	HeaderHeight
	HeaderTime
	HeaderLastHeaderHash
	HeaderDataHash
	HeaderConsensusHash
	HeaderAppHash
	HeaderLastResultsHash
	HeaderProposerAddr
	HeaderLastCommitHash
	HeaderSequencerHash
	HeaderNextSequencerHash
	Data
	LastCommit
)

// height -> cmd
type Cmds map[uint64]Cmd

type Cmd struct {
	*types.Block
	frauds []FraudType
}

func ApplyFraud(cmd Cmd, b *types.Block) *types.Block {
	for _, fraud := range cmd.frauds {
		switch fraud {
		case HeaderVersionBlock:
			b.Header.Version.Block = cmd.Header.Version.Block
		case HeaderVersionApp:
			b.Header.Version.App = cmd.Header.Version.App
		case HeaderChainID:
			b.Header.ChainID = cmd.Header.ChainID
		case HeaderHeight:
			b.Header.Height = cmd.Header.Height
		case HeaderTime:
			b.Header.Time = cmd.Header.Time
		case HeaderLastHeaderHash:
			b.Header.LastHeaderHash = cmd.Header.LastHeaderHash
		case HeaderDataHash:
			b.Header.DataHash = cmd.Header.DataHash
		case HeaderConsensusHash:
			b.Header.ConsensusHash = cmd.Header.ConsensusHash
		case HeaderAppHash:
			b.Header.AppHash = cmd.Header.AppHash
		case HeaderLastResultsHash:
			b.Header.LastResultsHash = cmd.Header.LastResultsHash
		case HeaderProposerAddr:
			b.Header.ProposerAddress = cmd.Header.ProposerAddress
		case HeaderLastCommitHash:
			b.Header.LastCommitHash = cmd.Header.LastCommitHash
		case HeaderSequencerHash:
			b.Header.SequencerHash = cmd.Header.SequencerHash
		case HeaderNextSequencerHash:
			b.Header.NextSequencersHash = cmd.Header.NextSequencersHash
		case Data:
			b.Data = cmd.Data
		case LastCommit:
			b.LastCommit = cmd.LastCommit
		default:
		}
	}
	return b
}
