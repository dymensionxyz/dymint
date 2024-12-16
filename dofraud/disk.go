package dofraud

import (
	"encoding/json"
	"io"
	"os"
	"slices"
	"strings"

	"github.com/dymensionxyz/dymint/types"
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
	ret.frauds = make(map[string]Cmd)
	for _, ins := range d.Instances {
		cmd := Cmd{
			Block: &types.Block{
				Header: types.Header{Version: types.Version{}},
			},
		}
		if ins.Block.HeaderVersionBlock != 0 {
			cmd.Block.Header.Version.Block = ins.Block.HeaderVersionBlock
		}
		if ins.Block.HeaderVersionApp != 0 {
			cmd.Block.Header.Version.App = ins.Block.HeaderVersionApp
		}
		if ins.Block.HeaderChainID != "" {
			cmd.Block.Header.ChainID = ins.Block.HeaderChainID
		}
		if ins.Block.HeaderHeight != 0 {
			cmd.Block.Header.Height = ins.Block.HeaderHeight
		}
		if ins.Block.HeaderTime != 0 {
			cmd.Block.Header.Time = ins.Block.HeaderTime
		}
		if ins.Block.HeaderLastHeaderHash != "" {
			cmd.Block.Header.LastHeaderHash = parseHash(ins.Block.HeaderLastHeaderHash)
		}
		if ins.Block.HeaderDataHash != "" {
			cmd.Block.Header.DataHash = parseHash(ins.Block.HeaderDataHash)
		}
		if ins.Block.HeaderConsensusHash != "" {
			cmd.Block.Header.ConsensusHash = parseHash(ins.Block.HeaderConsensusHash)
		}
		if ins.Block.HeaderAppHash != "" {
			cmd.Block.Header.AppHash = parseHash(ins.Block.HeaderAppHash)
		}
		if ins.Block.HeaderLastResultsHash != "" {
			cmd.Block.Header.LastResultsHash = parseHash(ins.Block.HeaderLastResultsHash)
		}
		if ins.Block.HeaderProposerAddr != "" {
			cmd.Block.Header.ProposerAddress = []byte(ins.Block.HeaderProposerAddr)
		}
		if ins.Block.HeaderLastCommitHash != "" {
			cmd.Block.Header.LastCommitHash = parseHash(ins.Block.HeaderLastCommitHash)
		}
		if ins.Block.HeaderSequencerHash != "" {
			cmd.Block.Header.SequencerHash = parseHash(ins.Block.HeaderSequencerHash)
		}
		if ins.Block.HeaderNextSequencerHash != "" {
			cmd.Block.Header.NextSequencersHash = parseHash(ins.Block.HeaderNextSequencerHash)
		}
		// TODO: Data and LastCommit
		vs := parseVariants(ins.Variants)
		for _, v := range vs {
			ret.frauds[key{ins.Height, v}.String()] = cmd
		}
	}
	return &ret, nil
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
