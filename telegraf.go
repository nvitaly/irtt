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

	// Build tags
	tags := make(map[string]string)
	for k, v := range opts.Tags {
		tags[k] = v
	}

	// Add target tag
	if r.Config != nil && r.Config.RemoteAddress != "" {
		tags["target"] = r.Config.RemoteAddress
	}

	// Build fields map
	fields := make(map[string]interface{})
	fields["success"] = 1

	// RTT statistics
	if opts.IncludeRTT {
		addDurationStatsToFields(fields, "rtt", &stats.RTTStats)
	}

	// Send delay statistics
	if opts.IncludeSendDelay {
		addDurationStatsToFields(fields, "send_delay", &stats.SendDelayStats)
	}

	// Receive delay statistics
	if opts.IncludeReceiveDelay {
		addDurationStatsToFields(fields, "receive_delay", &stats.ReceiveDelayStats)
	}

	// IPDV (jitter) statistics
	if opts.IncludeIPDV {
		addDurationStatsToFields(fields, "ipdv_rtt", &stats.RoundTripIPDVStats)
		addDurationStatsToFields(fields, "ipdv_send", &stats.SendIPDVStats)
		addDurationStatsToFields(fields, "ipdv_receive", &stats.ReceiveIPDVStats)
	}

	// Server processing time
	if opts.IncludeServerProcessing {
		addDurationStatsToFields(fields, "server_processing", &stats.ServerProcessingTimeStats)
	}

	// Packet loss statistics
	if opts.IncludePacketLoss {
		fields["packets_sent"] = stats.PacketsSent
		fields["packets_received"] = stats.PacketsReceived
		fields["packet_loss_percent"] = stats.PacketLossPercent
		fields["upstream_loss_percent"] = stats.UpstreamLossPercent
		fields["downstream_loss_percent"] = stats.DownstreamLossPercent
		fields["duplicates"] = stats.Duplicates
		fields["duplicate_percent"] = stats.DuplicatePercent
		fields["late_packets"] = stats.LatePackets
		fields["late_packets_percent"] = stats.LatePacketsPercent
	}

	// Bitrate statistics
	if opts.IncludeBitrate {
		fields["send_rate_bps"] = uint64(stats.SendRate)
		fields["receive_rate_bps"] = uint64(stats.ReceiveRate)
		fields["bytes_sent"] = stats.BytesSent
		fields["bytes_received"] = stats.BytesReceived
	}

	// Timer error statistics
	if opts.IncludeTimerError {
		addDurationStatsToFields(fields, "timer_error", &stats.TimerErrorStats)
		fields["timer_err_percent"] = stats.TimerErrPercent
		fields["timer_misses"] = stats.TimerMisses
		fields["timer_miss_percent"] = stats.TimerMissPercent
	}

	// Create metric and write as JSON
	metric := TelegrafMetric{
		Fields:    fields,
		Tags:      tags,
		Timestamp: timestamp,
	}

	encoder := json.NewEncoder(w)
	return encoder.Encode(metric)
}

// WriteTelegrafError writes an error in JSON format for Telegraf
func WriteTelegrafError(w io.Writer, err error, target string, opts *TelegrafOptions) error {
	if opts == nil {
		opts = DefaultTelegrafOptions()
	}

	timestamp := time.Now().Unix()

	// Build tags
	tags := make(map[string]string)
	for k, v := range opts.Tags {
		tags[k] = v
	}

	if target != "" {
		tags["target"] = target
	}

	// Build fields with failure status
	fields := map[string]interface{}{
		"success": 0,
	}

	// Create metric and write as JSON
	metric := TelegrafMetric{
		Fields:    fields,
		Tags:      tags,
		Timestamp: timestamp,
	}

	encoder := json.NewEncoder(w)
	return encoder.Encode(metric)
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
