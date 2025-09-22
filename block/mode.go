package block

import "fmt"

type RunMode struct {
	Name string
}

var (
	RunModeFullNode RunMode = RunMode{Name: "full node"}
	RunModeProposer RunMode = RunMode{Name: "proposer"}
)

// checks if the the current node is the proposer either on rollapp or on the hub.
// In case of sequencer rotation, there's a phase where proposer rotated on Rollapp but hasn't yet rotated on hub.
// for this case, 2 nodes will get `true` for `AmIProposer` so the l2 proposer can produce blocks and the hub proposer can submit his last batch.
// The hub proposer, after sending the last state update, will panic and restart as full node.
func (m *Manager) runMode() RunMode {
	if m.RunModeCached != nil {
		return *m.RunModeCached
	}
	isProposeronSL, err := m.AmIProposerOnSL()
	if err != nil {
		panic(fmt.Errorf("am i proposer on SL: %w", err))
	}

	isProposer := isProposeronSL || m.AmIProposerOnRollapp()
	if isProposer {
		m.RunModeCached = &RunModeProposer
		return RunModeProposer
	}
	m.RunModeCached = &RunModeFullNode
	return RunModeFullNode
}
