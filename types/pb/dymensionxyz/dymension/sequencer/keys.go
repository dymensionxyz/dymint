package sequencer

import (
	"encoding/binary"
	fmt "fmt"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

var _ binary.ByteOrder

const (
	
	ModuleName = "sequencer"

	
	StoreKey = ModuleName

	
	RouterKey = ModuleName

	
	QuerierRoute = ModuleName

	
	MemStoreKey = "mem_sequencer"
)

var (
	
	KeySeparator = "/"

	
	SequencersKeyPrefix = []byte{0x00} 

	
	SequencersByRollappKeyPrefix = []byte{0x01} 
	BondedSequencersKeyPrefix    = []byte{0xa1}
	UnbondedSequencersKeyPrefix  = []byte{0xa2}
	UnbondingSequencersKeyPrefix = []byte{0xa3}

	UnbondingQueueKey = []byte{0x41} 
)


func SequencerKey(sequencerAddress string) []byte {
	sequencerAddrBytes := []byte(sequencerAddress)
	return []byte(fmt.Sprintf("%s%s%s", SequencersKeyPrefix, KeySeparator, sequencerAddrBytes))
}


func SequencerByRollappByStatusKey(rollappId, seqAddr string, status OperatingStatus) []byte {
	return append(SequencersByRollappByStatusKey(rollappId, status), []byte(seqAddr)...)
}


func SequencersKey() []byte {
	return SequencersKeyPrefix
}


func SequencersByRollappKey(rollappId string) []byte {
	rollappIdBytes := []byte(rollappId)
	return []byte(fmt.Sprintf("%s%s%s", SequencersByRollappKeyPrefix, KeySeparator, rollappIdBytes))
}


func SequencersByRollappByStatusKey(rollappId string, status OperatingStatus) []byte {
	
	var prefix []byte
	switch status {
	case Bonded:
		prefix = BondedSequencersKeyPrefix
	case Unbonded:
		prefix = UnbondedSequencersKeyPrefix
	case Unbonding:
		prefix = UnbondingSequencersKeyPrefix
	}

	return []byte(fmt.Sprintf("%s%s%s", SequencersByRollappKey(rollappId), KeySeparator, prefix))
}


func UnbondingQueueByTimeKey(endTime time.Time) []byte {
	timeBz := sdk.FormatTimeBytes(endTime)
	prefixL := len(UnbondingQueueKey)

	bz := make([]byte, prefixL+len(timeBz))

	
	copy(bz[:prefixL], UnbondingQueueKey)
	
	copy(bz[prefixL:prefixL+len(timeBz)], timeBz)

	return bz
}

func UnbondingSequencerKey(sequencerAddress string, endTime time.Time) []byte {
	key := UnbondingQueueByTimeKey(endTime)
	key = append(key, KeySeparator...)
	key = append(key, []byte(sequencerAddress)...)
	return key
}
