package dofraud

import (
	"encoding/json"
	"github.com/dymensionxyz/dymint/types"
	"io"
	"os"
	"slices"
	"strings"
)

type disk struct {
	Instances []diskInstance `json:",omitempty"`
}

type diskInstance struct {
	Height  uint64
	Variant string
	Block   diskBlock
}

type diskBlock struct {
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

func Load(fn string) (*Frauds, error) {
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

	ret := Frauds{}
	for _, pair := range d.Instances {
		cmd := Cmd{
			Block: &types.Block{},
		}
		key := key{pair.Height, parseVariant(pair.Variant)}
		if pair.Block.HeaderVersionBlock != 0 {
			frauds = append(frauds, HeaderVersionBlock)
		}
		if pair.Block.HeaderVersionApp != 0 {
			frauds = append(frauds, HeaderVersionApp)
		}
		if pair.Block.HeaderChainID != "" {
			frauds = append(frauds, HeaderChainID)
		}
		if pair.Block.HeaderHeight != 0 {
			frauds = append(frauds, HeaderHeight)
		}
		if pair.Block.HeaderTime != 0 {
			frauds = append(frauds, HeaderTime)
		}
		if pair.Block.HeaderLastHeaderHash != "" {
			frauds = append(frauds, HeaderLastHeaderHash)
		}
		if pair.Block.HeaderDataHash != "" {
			frauds = append(frauds, HeaderDataHash)
		}
		if pair.Block.HeaderConsensusHash != "" {
			frauds = append(frauds, HeaderConsensusHash)
		}
		if pair.Block.HeaderAppHash != "" {
			frauds = append(frauds, HeaderAppHash)
		}
		if pair.Block.HeaderLastResultsHash != "" {
			frauds = append(frauds, HeaderLastResultsHash)
		}
		if pair.Block.HeaderProposerAddr != "" {
			frauds = append(frauds, HeaderProposerAddr)
		}
		if pair.Block.HeaderLastCommitHash != "" {
			frauds = append(frauds, HeaderLastCommitHash)
		}
		if pair.Block.HeaderSequencerHash != "" {
			frauds = append(frauds, HeaderSequencerHash)
		}
		if pair.Block.HeaderNextSequencerHash != "" {
			frauds = append(frauds, HeaderNextSequencerHash)
		}
		// TODO: Data and LastCommit
	}

	cmds := make(Cmds)
	for _, pair := range d.Instances {
		cmd := Cmd{
			Block: &types.Block{
				Header: types.Header{
					Version: types.Version{
						Block: pair.Block.HeaderVersionBlock,
						App:   pair.Block.HeaderVersionApp,
					},
					ChainID:            pair.Block.HeaderChainID,
					Height:             pair.Block.HeaderHeight,
					Time:               pair.Block.HeaderTime,
					LastHeaderHash:     parseHash(pair.Block.HeaderLastHeaderHash),
					DataHash:           parseHash(pair.Block.HeaderDataHash),
					ConsensusHash:      parseHash(pair.Block.HeaderConsensusHash),
					AppHash:            parseHash(pair.Block.HeaderAppHash),
					LastResultsHash:    parseHash(pair.Block.HeaderLastResultsHash),
					ProposerAddress:    []byte(pair.Block.HeaderProposerAddr),
					LastCommitHash:     parseHash(pair.Block.HeaderLastCommitHash),
					SequencerHash:      parseHash(pair.Block.HeaderSequencerHash),
					NextSequencersHash: parseHash(pair.Block.HeaderNextSequencerHash),
				},
				// Data and LastCommit fields need to be populated as per your requirements
			},
			ts: frauds,
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

func parseVariants(s string) []int {
	ret := []int{}
	s = strings.ToLower(s)
	if strings.Contains(s, ",") {
		l := strings.Split(s, ",")
		for _, v := range l {
			sub := parseVariants(v)
			ret = append(ret, sub...)
		}
		return ret
	}
	if s == "da" {
		return []int{DA}
	}
	if s == "gossip" {
		return []int{DA}
	}
}
