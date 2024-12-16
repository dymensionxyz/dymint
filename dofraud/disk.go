package dofraud

import (
	"encoding/json"
	"io"
	"os"
	"slices"
	"strings"

	"github.com/dymensionxyz/dymint/types"
	"github.com/dymensionxyz/gerr-cosmos/gerrc"
)

type disk struct {
	Instances []diskInstance `json:",omitempty"`
}

type diskInstance struct {
	Height   uint64
	Variants string
	Block    diskBlock
}

type diskBlock struct {
	HeaderVersionBlock      uint64    `json:",omitempty"`
	HeaderVersionApp        uint64    `json:",omitempty"`
	HeaderChainID           string    `json:",omitempty"`
	HeaderHeight            uint64    `json:",omitempty"`
	HeaderTime              int64     `json:",omitempty"`
	HeaderLastHeaderHash    string    `json:",omitempty"`
	HeaderDataHash          string    `json:",omitempty"`
	HeaderConsensusHash     string    `json:",omitempty"`
	HeaderAppHash           string    `json:",omitempty"`
	HeaderLastResultsHash   string    `json:",omitempty"`
	HeaderProposerAddr      string    `json:",omitempty"`
	HeaderLastCommitHash    string    `json:",omitempty"`
	HeaderSequencerHash     string    `json:",omitempty"`
	HeaderNextSequencerHash string    `json:",omitempty"`
	Data                    *struct{} `json:",omitempty"` // TODO:
	LastCommit              *struct{} `json:",omitempty"` // TODO:
}

func Load(fn string) (Frauds, error) {
	file, err := os.Open(fn) //nolint:gosec
	if err != nil {
		return Frauds{}, err
	}
	defer file.Close() //nolint:errcheck

	data, err := io.ReadAll(file)
	if err != nil {
		return Frauds{}, err
	}

	var d disk
	err = json.Unmarshal(data, &d)
	if err != nil {
		return Frauds{}, err
	}

	ret := Frauds{}
	ret.frauds = make(map[string]Cmd)
	for _, ins := range d.Instances {
		cmd := Cmd{
			Block: &types.Block{
				Header: types.Header{Version: types.Version{}},
			},
		}
		if ins.Block.HeaderVersionBlock != 0 {
			cmd.Block.Header.Version.Block = ins.Block.HeaderVersionBlock
			cmd.ts = append(cmd.ts, HeaderVersionBlock)
		}
		if ins.Block.HeaderVersionApp != 0 {
			cmd.Block.Header.Version.App = ins.Block.HeaderVersionApp
			cmd.ts = append(cmd.ts, HeaderVersionApp)
		}
		if ins.Block.HeaderChainID != "" {
			cmd.Block.Header.ChainID = ins.Block.HeaderChainID
			cmd.ts = append(cmd.ts, HeaderChainID)
		}
		if ins.Block.HeaderHeight != 0 {
			cmd.Block.Header.Height = ins.Block.HeaderHeight
			cmd.ts = append(cmd.ts, HeaderHeight)
		}
		if ins.Block.HeaderTime != 0 {
			cmd.Block.Header.Time = ins.Block.HeaderTime
			cmd.ts = append(cmd.ts, HeaderTime)
		}
		if ins.Block.HeaderLastHeaderHash != "" {
			cmd.Block.Header.LastHeaderHash = parseHash(ins.Block.HeaderLastHeaderHash)
			cmd.ts = append(cmd.ts, HeaderLastHeaderHash)
		}
		if ins.Block.HeaderDataHash != "" {
			cmd.Block.Header.DataHash = parseHash(ins.Block.HeaderDataHash)
			cmd.ts = append(cmd.ts, HeaderDataHash)
		}
		if ins.Block.HeaderConsensusHash != "" {
			cmd.Block.Header.ConsensusHash = parseHash(ins.Block.HeaderConsensusHash)
			cmd.ts = append(cmd.ts, HeaderConsensusHash)
		}
		if ins.Block.HeaderAppHash != "" {
			cmd.Block.Header.AppHash = parseHash(ins.Block.HeaderAppHash)
			cmd.ts = append(cmd.ts, HeaderAppHash)
		}
		if ins.Block.HeaderLastResultsHash != "" {
			cmd.Block.Header.LastResultsHash = parseHash(ins.Block.HeaderLastResultsHash)
			cmd.ts = append(cmd.ts, HeaderLastResultsHash)
		}
		if ins.Block.HeaderProposerAddr != "" {
			cmd.Block.Header.ProposerAddress = []byte(ins.Block.HeaderProposerAddr)
			cmd.ts = append(cmd.ts, HeaderProposerAddr)
		}
		if ins.Block.HeaderLastCommitHash != "" {
			cmd.Block.Header.LastCommitHash = parseHash(ins.Block.HeaderLastCommitHash)
			cmd.ts = append(cmd.ts, HeaderLastCommitHash)
		}
		if ins.Block.HeaderSequencerHash != "" {
			cmd.Block.Header.SequencerHash = parseHash(ins.Block.HeaderSequencerHash)
			cmd.ts = append(cmd.ts, HeaderSequencerHash)
		}
		if ins.Block.HeaderNextSequencerHash != "" {
			cmd.Block.Header.NextSequencersHash = parseHash(ins.Block.HeaderNextSequencerHash)
			cmd.ts = append(cmd.ts, HeaderNextSequencerHash)
		}
		if ins.Block.Data != nil {
			return Frauds{}, gerrc.ErrUnimplemented.Wrap("block data")
		}
		if ins.Block.LastCommit != nil {
			return Frauds{}, gerrc.ErrUnimplemented.Wrap("last commit")
		}
		vs := parseVariants(ins.Variants)
		for _, v := range vs {
			ret.frauds[key{ins.Height, v}.String()] = cmd
		}
	}
	return ret, nil
}

func parseHash(hashStr string) [32]byte {
	var hash [32]byte
	copy(hash[:], hashStr)
	return hash
}

func parseVariants(s string) []FraudVariant {
	ret := []FraudVariant{}
	l := strings.Split(s, ",")
	if slices.Contains(l, "da") {
		ret = append(ret, DA)
	}
	if slices.Contains(l, "gossip") {
		ret = append(ret, Gossip)
	}
	if slices.Contains(l, "produce") {
		ret = append(ret, Produce)
	}
	return ret
}
