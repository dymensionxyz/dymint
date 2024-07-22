package types

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var RollappHeightGauge = promauto.NewGauge(prometheus.GaugeOpts{
	Name: "rollapp_height",
	Help: "The height of the local rollapp",
})

var RollappHubHeightGauge = promauto.NewGauge(prometheus.GaugeOpts{
	Name: "rollapp_hub_height",
	Help: "The latest height of the Rollapp that has been synced to the hub.",
})

var RollappBlockSizeBytesGauge = promauto.NewGauge(prometheus.GaugeOpts{
	Name: "rollapp_block_size_bytes",
	Help: "Rollapp ",
})

var RollappBlockSizeTxsGauge = promauto.NewGauge(prometheus.GaugeOpts{
	Name: "rollapp_block_size_txs",
	Help: "Rollapp ",
})

var LastReceivedP2PHeightGauge = promauto.NewGauge(prometheus.GaugeOpts{
	Name: "last_received_p2p_height",
	Help: "The height of the last block received from P2P.",
})

var LastReceivedDAHeight = promauto.NewGauge(prometheus.GaugeOpts{
	Name: "last_received_da_height",
	Help: "The height of the last block received from DA.",
})

var LastAppliedBlockSource = promauto.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "last_applied_block_source",
		Help: "The source of the last applied block",
	},
	[]string{"source"},
)

var LowestPendingBlockHeight = promauto.NewGauge(prometheus.GaugeOpts{
	Name: "lowest_pending_block_height",
	Help: "The lowest height of the pending blocks.",
})

var HighestReceivedBlockHeight = promauto.NewGauge(prometheus.GaugeOpts{
	Name: "highest_received_block_height",
	Help: "The highest height of the received blocks.",
})

var NumberOfAccumulatedBlocks = promauto.NewGauge(prometheus.GaugeOpts{
	Name: "number_of_accumulated_blocks",
	Help: "The number of accumulated blocks.",
})
