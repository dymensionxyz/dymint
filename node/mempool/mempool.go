package nodemempool

import (
	"fmt"
	"math"

	"github.com/libp2p/go-libp2p/core/peer"
	tmsync "github.com/tendermint/tendermint/libs/sync"
)

const (
	maxActiveIDs = math.MaxUint16
)

type MempoolIDs struct {
	mtx       tmsync.RWMutex
	peerMap   map[peer.ID]uint16
	nextID    uint16
	activeIDs map[uint16]struct{}
}

func (ids *MempoolIDs) ReserveForPeer(peer peer.ID) {
	ids.mtx.Lock()
	defer ids.mtx.Unlock()

	curID := ids.nextPeerID()
	ids.peerMap[peer] = curID
	ids.activeIDs[curID] = struct{}{}
}

func (ids *MempoolIDs) nextPeerID() uint16 {
	if len(ids.activeIDs) == maxActiveIDs {
		panic(fmt.Sprintf("node has maximum %d active IDs and wanted to get one more", maxActiveIDs))
	}

	_, idExists := ids.activeIDs[ids.nextID]
	for idExists {
		ids.nextID++
		_, idExists = ids.activeIDs[ids.nextID]
	}
	curID := ids.nextID
	ids.nextID++
	return curID
}

func (ids *MempoolIDs) Reclaim(peer peer.ID) {
	ids.mtx.Lock()
	defer ids.mtx.Unlock()

	removedID, ok := ids.peerMap[peer]
	if ok {
		delete(ids.activeIDs, removedID)
		delete(ids.peerMap, peer)
	}
}

func (ids *MempoolIDs) GetForPeer(peer peer.ID) uint16 {
	ids.mtx.Lock()
	defer ids.mtx.Unlock()

	id, ok := ids.peerMap[peer]
	if !ok {
		id = ids.nextPeerID()
		ids.peerMap[peer] = id
		ids.activeIDs[id] = struct{}{}
	}

	return id
}

func NewMempoolIDs() *MempoolIDs {
	return &MempoolIDs{
		peerMap:   make(map[peer.ID]uint16),
		activeIDs: map[uint16]struct{}{0: {}},
		nextID:    1,
	}
}
