package irtt

import (
	"fmt"
	"io"
	"strings"
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

const telegrafMeasurement = "irtt"

// WriteResultTelegraf writes result in InfluxDB line protocol format
func WriteResultTelegraf(w io.Writer, r *Result, opts *TelegrafOptions) error {
	if opts == nil {
		opts = DefaultTelegrafOptions()
	}

	stats := r.Stats
	if stats == nil {
		return fmt.Errorf("no stats in result")
	}

	timestamp := time.Now().UnixNano()

	// Build base tags
	tags := make(map[string]string)
	for k, v := range opts.Tags {
		tags[k] = v
	}

	// Add target tag
	if r.Config != nil && r.Config.RemoteAddress != "" {
		tags["target"] = r.Config.RemoteAddress
	}

	tagStr := buildTelegrafTagString(tags)

	// Write status metric
	fmt.Fprintf(w, "%s%s success=1i %d\n", telegrafMeasurement, tagStr, timestamp)

	// RTT statistics
	if opts.IncludeRTT {
		if fields := formatDurationStats("rtt", &stats.RTTStats); fields != "" {
			fmt.Fprintf(w, "%s%s %s %d\n", telegrafMeasurement, tagStr, fields, timestamp)
		}
	}

	// Send delay statistics
	if opts.IncludeSendDelay {
		if fields := formatDurationStats("send_delay", &stats.SendDelayStats); fields != "" {
			fmt.Fprintf(w, "%s%s %s %d\n", telegrafMeasurement, tagStr, fields, timestamp)
		}
	}

	// Receive delay statistics
	if opts.IncludeReceiveDelay {
		if fields := formatDurationStats("receive_delay", &stats.ReceiveDelayStats); fields != "" {
			fmt.Fprintf(w, "%s%s %s %d\n", telegrafMeasurement, tagStr, fields, timestamp)
		}
	}

	// IPDV (jitter) statistics
	if opts.IncludeIPDV {
		if fields := formatDurationStats("ipdv_rtt", &stats.RoundTripIPDVStats); fields != "" {
			fmt.Fprintf(w, "%s%s %s %d\n", telegrafMeasurement, tagStr, fields, timestamp)
		}
		if fields := formatDurationStats("ipdv_send", &stats.SendIPDVStats); fields != "" {
			fmt.Fprintf(w, "%s%s %s %d\n", telegrafMeasurement, tagStr, fields, timestamp)
		}
		if fields := formatDurationStats("ipdv_receive", &stats.ReceiveIPDVStats); fields != "" {
			fmt.Fprintf(w, "%s%s %s %d\n", telegrafMeasurement, tagStr, fields, timestamp)
		}
	}

	// Server processing time
	if opts.IncludeServerProcessing {
		if fields := formatDurationStats("server_processing", &stats.ServerProcessingTimeStats); fields != "" {
			fmt.Fprintf(w, "%s%s %s %d\n", telegrafMeasurement, tagStr, fields, timestamp)
		}
	}

	// Packet loss statistics
	if opts.IncludePacketLoss {
		lossFields := []string{
			fmt.Sprintf("packets_sent=%di", stats.PacketsSent),
			fmt.Sprintf("packets_received=%di", stats.PacketsReceived),
			fmt.Sprintf("packet_loss_percent=%.4f", stats.PacketLossPercent),
			fmt.Sprintf("upstream_loss_percent=%.4f", stats.UpstreamLossPercent),
			fmt.Sprintf("downstream_loss_percent=%.4f", stats.DownstreamLossPercent),
			fmt.Sprintf("duplicates=%di", stats.Duplicates),
			fmt.Sprintf("duplicate_percent=%.4f", stats.DuplicatePercent),
			fmt.Sprintf("late_packets=%di", stats.LatePackets),
			fmt.Sprintf("late_packets_percent=%.4f", stats.LatePacketsPercent),
		}
		fmt.Fprintf(w, "%s%s %s %d\n", telegrafMeasurement, tagStr, strings.Join(lossFields, ","), timestamp)
	}

	// Bitrate statistics
	if opts.IncludeBitrate {
		bitrateFields := []string{
			fmt.Sprintf("send_rate_bps=%di", uint64(stats.SendRate)),
			fmt.Sprintf("receive_rate_bps=%di", uint64(stats.ReceiveRate)),
			fmt.Sprintf("bytes_sent=%di", stats.BytesSent),
			fmt.Sprintf("bytes_received=%di", stats.BytesReceived),
		}
		fmt.Fprintf(w, "%s%s %s %d\n", telegrafMeasurement, tagStr, strings.Join(bitrateFields, ","), timestamp)
	}

	// Timer error statistics
	if opts.IncludeTimerError {
		var timerFields []string
		if fields := formatDurationStats("timer_error", &stats.TimerErrorStats); fields != "" {
			timerFields = append(timerFields, fields)
		}
		timerFields = append(timerFields,
			fmt.Sprintf("timer_err_percent=%.4f", stats.TimerErrPercent),
			fmt.Sprintf("timer_misses=%di", stats.TimerMisses),
			fmt.Sprintf("timer_miss_percent=%.4f", stats.TimerMissPercent),
		)
		if len(timerFields) > 0 {
			fmt.Fprintf(w, "%s%s %s %d\n", telegrafMeasurement, tagStr, strings.Join(timerFields, ","), timestamp)
		}
	}

	return nil
}

// WriteTelegrafError writes an error in InfluxDB line protocol format
func WriteTelegrafError(w io.Writer, err error, target string, opts *TelegrafOptions) error {
	if opts == nil {
		opts = DefaultTelegrafOptions()
	}

	timestamp := time.Now().UnixNano()

	// Build tags
	tags := make(map[string]string)
	for k, v := range opts.Tags {
		tags[k] = v
	}

	if target != "" {
		tags["target"] = target
	}

	tagStr := buildTelegrafTagString(tags)

	// Create metric with failure status
	fmt.Fprintf(w, "%s%s success=0i %d\n", telegrafMeasurement, tagStr, timestamp)

	return nil
}

// buildTelegrafTagString creates the tag portion of InfluxDB line protocol
func buildTelegrafTagString(tags map[string]string) string {
	if len(tags) == 0 {
		return ""
	}

	var tagPairs []string
	for k, v := range tags {
		// Escape special characters in tag values
		v = strings.ReplaceAll(v, ",", "\\,")
		v = strings.ReplaceAll(v, "=", "\\=")
		v = strings.ReplaceAll(v, " ", "\\ ")
		tagPairs = append(tagPairs, fmt.Sprintf("%s=%s", k, v))
	}

	return "," + strings.Join(tagPairs, ",")
}

// formatDurationStats creates field string from DurationStats
func formatDurationStats(prefix string, ds *DurationStats) string {
	if ds == nil || ds.N == 0 {
		return ""
	}

	fields := []string{
		fmt.Sprintf("%s_n=%di", prefix, ds.N),
		fmt.Sprintf("%s_min_ns=%di", prefix, ds.Min.Nanoseconds()),
		fmt.Sprintf("%s_max_ns=%di", prefix, ds.Max.Nanoseconds()),
		fmt.Sprintf("%s_mean_ns=%di", prefix, ds.Mean().Nanoseconds()),
	}

	if median, ok := ds.Median(); ok {
		fields = append(fields, fmt.Sprintf("%s_median_ns=%di", prefix, median.Nanoseconds()))
	}

	stddev := ds.Stddev()
	fields = append(fields, fmt.Sprintf("%s_stddev_ns=%di", prefix, stddev.Nanoseconds()))

	return strings.Join(fields, ",")
}
