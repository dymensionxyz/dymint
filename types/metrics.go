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

var RollappPendingSubmissionsSkewNumBatches = promauto.NewGauge(prometheus.GaugeOpts{
	Name: "rollapp_pending_submissions_skew_num_batches",
	Help: "The number of batches which have been accumulated but not yet submitted.",
})

var RollappPendingSubmissionsSkewNumBytes = promauto.NewGauge(prometheus.GaugeOpts{
	Name: "rollapp_pending_submissions_skew_num_bytes",
	Help: "The number of bytes (of blocks and commits) which have been accumulated but not yet submitted.",
})

var LastBatchSubmittedBytes = promauto.NewGauge(prometheus.GaugeOpts{
	Name: "last_batch_submitted_bytes",
	Help: "The size in bytes of the last batch submitted to DA.",
})

var LastReceivedP2PHeightGauge = promauto.NewGauge(prometheus.GaugeOpts{
	Name: "last_received_p2p_height",
	Help: "The height of the last block received from P2P.",
})

var LastReceivedDAHeightGauge = promauto.NewGauge(prometheus.GaugeOpts{
	Name: "last_received_da_height",
	Help: "The height of the last block received from DA.",
})

const SourceLabel = "source"

func init() {
	LastAppliedBlockSource.With(prometheus.Labels{SourceLabel: "none"}).Set(0)
}

var LastAppliedBlockSource = promauto.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "last_applied_block_source",
		Help: "The source of the last applied block",
	},
	[]string{SourceLabel},
)

func SetLastAppliedBlockSource(source string) {
	LastAppliedBlockSource.Reset()
	LastAppliedBlockSource.With(prometheus.Labels{SourceLabel: source}).Set(0)
}

var BlockCacheSizeGauge = promauto.NewGauge(prometheus.GaugeOpts{
	Name: "block_cache_size",
	Help: "The number of blocks in the cache.",
})

var RollappConsecutiveFailedDASubmission = promauto.NewGauge(prometheus.GaugeOpts{
	Name: "rollapp_consecutive_failed_da_submissions",
	Help: "The number of consecutive times the da fails to submit.",
})
