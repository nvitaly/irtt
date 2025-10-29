package irtt

import (
	"encoding/json"
	"fmt"
	"io"
	"time"
)

// TelegrafOptions controls Telegraf output behavior
type TelegrafOptions struct {
	// Tags to add to all metrics
	Tags map[string]string

	// Which metrics to include
	IncludeRTT              bool
	IncludeSendDelay        bool
	IncludeReceiveDelay     bool
	IncludeIPDV             bool
	IncludePacketLoss       bool
	IncludeBitrate          bool
	IncludeServerProcessing bool
	IncludeTimerError       bool
}

// DefaultTelegrafOptions returns default options
func DefaultTelegrafOptions() *TelegrafOptions {
	return &TelegrafOptions{
		Tags:                    make(map[string]string),
		IncludeRTT:              true,
		IncludeSendDelay:        true,
		IncludeReceiveDelay:     true,
		IncludeIPDV:             true,
		IncludePacketLoss:       true,
		IncludeBitrate:          true,
		IncludeServerProcessing: true,
		IncludeTimerError:       false,
	}
}

// TelegrafMetric represents a metric in Telegraf JSON format
type TelegrafMetric struct {
	Fields    map[string]interface{} `json:"fields"`
	Tags      map[string]string      `json:"tags"`
	Timestamp int64                  `json:"timestamp"`
}

// WriteResultTelegraf writes result in JSON format for Telegraf
func WriteResultTelegraf(w io.Writer, r *Result, opts *TelegrafOptions) error {
	if opts == nil {
		opts = DefaultTelegrafOptions()
	}

	stats := r.Stats
	if stats == nil {
		return fmt.Errorf("no stats in result")
	}

	timestamp := time.Now().Unix()

	// Build flat map with all data
	flat := make(map[string]interface{})

	// Add timestamp
	flat["timestamp"] = timestamp

	// Add all tags
	for k, v := range opts.Tags {
		flat[k] = v
	}

	// Add target tag
	if r.Config != nil && r.Config.RemoteAddress != "" {
		flat["target"] = r.Config.RemoteAddress
	}

	// Add success field
	flat["success"] = 1

	// RTT statistics
	if opts.IncludeRTT {
		addDurationStatsToFields(flat, "rtt", &stats.RTTStats)
	}

	// Send delay statistics
	if opts.IncludeSendDelay {
		addDurationStatsToFields(flat, "send_delay", &stats.SendDelayStats)
	}

	// Receive delay statistics
	if opts.IncludeReceiveDelay {
		addDurationStatsToFields(flat, "receive_delay", &stats.ReceiveDelayStats)
	}

	// IPDV (jitter) statistics
	if opts.IncludeIPDV {
		addDurationStatsToFields(flat, "ipdv_rtt", &stats.RoundTripIPDVStats)
		addDurationStatsToFields(flat, "ipdv_send", &stats.SendIPDVStats)
		addDurationStatsToFields(flat, "ipdv_receive", &stats.ReceiveIPDVStats)
	}

	// Server processing time
	if opts.IncludeServerProcessing {
		addDurationStatsToFields(flat, "server_processing", &stats.ServerProcessingTimeStats)
	}

	// Packet loss statistics
	if opts.IncludePacketLoss {
		flat["packets_sent"] = stats.PacketsSent
		flat["packets_received"] = stats.PacketsReceived
		flat["packet_loss_percent"] = stats.PacketLossPercent
		flat["upstream_loss_percent"] = stats.UpstreamLossPercent
		flat["downstream_loss_percent"] = stats.DownstreamLossPercent
		flat["duplicates"] = stats.Duplicates
		flat["duplicate_percent"] = stats.DuplicatePercent
		flat["late_packets"] = stats.LatePackets
		flat["late_packets_percent"] = stats.LatePacketsPercent
	}

	// Bitrate statistics
	if opts.IncludeBitrate {
		flat["send_rate_bps"] = uint64(stats.SendRate)
		flat["receive_rate_bps"] = uint64(stats.ReceiveRate)
		flat["bytes_sent"] = stats.BytesSent
		flat["bytes_received"] = stats.BytesReceived
	}

	// Timer error statistics
	if opts.IncludeTimerError {
		addDurationStatsToFields(flat, "timer_error", &stats.TimerErrorStats)
		flat["timer_err_percent"] = stats.TimerErrPercent
		flat["timer_misses"] = stats.TimerMisses
		flat["timer_miss_percent"] = stats.TimerMissPercent
	}

	// Write flat map as JSON
	encoder := json.NewEncoder(w)
	return encoder.Encode(flat)
}

// WriteTelegrafError writes an error in JSON format for Telegraf
func WriteTelegrafError(w io.Writer, err error, target string, opts *TelegrafOptions) error {
	if opts == nil {
		opts = DefaultTelegrafOptions()
	}

	timestamp := time.Now().Unix()

	// Build flat map with all data
	flat := make(map[string]interface{})

	// Add timestamp
	flat["timestamp"] = timestamp

	// Add all tags
	for k, v := range opts.Tags {
		flat[k] = v
	}

	if target != "" {
		flat["target"] = target
	}

	// Add failure status
	flat["success"] = 0

	// Write flat map as JSON
	encoder := json.NewEncoder(w)
	return encoder.Encode(flat)
}

// addDurationStatsToFields adds duration statistics to the fields map
func addDurationStatsToFields(fields map[string]interface{}, prefix string, ds *DurationStats) {
	if ds == nil || ds.N == 0 {
		return
	}

	fields[prefix+"_n"] = ds.N
	fields[prefix+"_min_ns"] = ds.Min.Nanoseconds()
	fields[prefix+"_max_ns"] = ds.Max.Nanoseconds()
	fields[prefix+"_mean_ns"] = ds.Mean().Nanoseconds()

	if median, ok := ds.Median(); ok {
		fields[prefix+"_median_ns"] = median.Nanoseconds()
	}

	stddev := ds.Stddev()
	fields[prefix+"_stddev_ns"] = stddev.Nanoseconds()
}
