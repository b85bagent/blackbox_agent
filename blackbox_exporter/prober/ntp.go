package prober

import (
	"blackbox_agent/blackbox_exporter/config"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"net"
	"runtime"
	"runtime/debug"
	"strings"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	ntpEpochOffset = 2208988800
	ntpPort        = "123"
	ntpPacketSize  = 48
)

type ntpSample struct {
	DriftSeconds              float64
	RTTSeconds                float64
	ReferenceTimestampSeconds float64
	RootDelaySeconds          float64
	RootDispersionSeconds     float64
	RootDistanceSeconds       float64
	PrecisionSeconds          float64
	Stratum                   float64
	Leap                      float64
	Server                    string
	ReferenceID               string
}

func ProbeNTP(ctx context.Context, target string, module config.Module, registry *prometheus.Registry, logger log.Logger) bool {
	buildInfo := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "ntp_build_info",
		Help: "Build information for the embedded NTP probe.",
	}, []string{"goversion", "version"})
	driftGauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "ntp_drift_seconds",
		Help: "Clock drift between the local node and the NTP server.",
	})
	stratumGauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "ntp_stratum",
		Help: "Stratum reported by the NTP server.",
	})
	rttGauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "ntp_rtt_seconds",
		Help: "Round-trip time of the NTP request.",
	})
	referenceTimestampGauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "ntp_reference_timestamp_seconds",
		Help: "Reference timestamp reported by the NTP server.",
	})
	rootDelayGauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "ntp_root_delay_seconds",
		Help: "Root delay reported by the NTP server.",
	})
	rootDispersionGauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "ntp_root_dispersion_seconds",
		Help: "Root dispersion reported by the NTP server.",
	})
	rootDistanceGauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "ntp_root_distance_seconds",
		Help: "Estimated root distance for the NTP sample.",
	})
	precisionGauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "ntp_precision_seconds",
		Help: "Clock precision reported by the NTP server.",
	})
	leapGauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "ntp_leap",
		Help: "Leap indicator reported by the NTP server.",
	})
	scrapeDurationGauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "ntp_scrape_duration_seconds",
		Help: "Duration spent collecting the NTP sample.",
	})
	serverInfoGauge := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "ntp_server_info",
		Help: "Information about the NTP server used for the sample.",
	}, []string{"server", "reference_id"})
	serverReachableGauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "ntp_server_reachable",
		Help: "Whether the NTP server was reachable during the probe.",
	})

	registry.MustRegister(buildInfo)
	registry.MustRegister(driftGauge)
	registry.MustRegister(stratumGauge)
	registry.MustRegister(rttGauge)
	registry.MustRegister(referenceTimestampGauge)
	registry.MustRegister(rootDelayGauge)
	registry.MustRegister(rootDispersionGauge)
	registry.MustRegister(rootDistanceGauge)
	registry.MustRegister(precisionGauge)
	registry.MustRegister(leapGauge)
	registry.MustRegister(scrapeDurationGauge)
	registry.MustRegister(serverInfoGauge)
	registry.MustRegister(serverReachableGauge)

	version := "dev"
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" {
		version = info.Main.Version
	}
	buildInfo.WithLabelValues(runtime.Version(), version).Set(1)
	serverReachableGauge.Set(0)

	start := time.Now()
	sample, err := collectNTPSample(ctx, target, module, registry, logger)
	scrapeDurationGauge.Set(time.Since(start).Seconds())
	if err != nil {
		level.Error(logger).Log("msg", "NTP probe failed", "err", err)
		return false
	}

	driftGauge.Set(sample.DriftSeconds)
	stratumGauge.Set(sample.Stratum)
	rttGauge.Set(sample.RTTSeconds)
	referenceTimestampGauge.Set(sample.ReferenceTimestampSeconds)
	rootDelayGauge.Set(sample.RootDelaySeconds)
	rootDispersionGauge.Set(sample.RootDispersionSeconds)
	rootDistanceGauge.Set(sample.RootDistanceSeconds)
	precisionGauge.Set(sample.PrecisionSeconds)
	leapGauge.Set(sample.Leap)
	serverInfoGauge.WithLabelValues(sample.Server, sample.ReferenceID).Set(1)
	serverReachableGauge.Set(1)

	level.Info(logger).Log("msg", "NTP probe succeeded", "target", sample.Server, "drift_seconds", sample.DriftSeconds, "rtt_seconds", sample.RTTSeconds)
	return true
}

func collectNTPSample(ctx context.Context, target string, module config.Module, registry *prometheus.Registry, logger log.Logger) (ntpSample, error) {
	targetHost, targetPort := splitNTPAddress(target)
	ip, _, err := chooseProtocol(ctx, module.NTP.IPProtocol, module.NTP.IPProtocolFallback, targetHost, registry, logger)
	if err != nil {
		return ntpSample{}, err
	}

	best, err := queryNTP(ctx, ip, targetHost, targetPort, module, logger)
	if err != nil {
		return ntpSample{}, err
	}

	threshold := module.NTP.HighDriftThreshold.Seconds()
	if module.NTP.MeasurementDuration <= 0 || math.Abs(best.DriftSeconds) <= threshold {
		return best, nil
	}

	deadline := time.Now().Add(module.NTP.MeasurementDuration)
	if ctxDeadline, ok := ctx.Deadline(); ok && ctxDeadline.Before(deadline) {
		deadline = ctxDeadline
	}

	for time.Now().Before(deadline) {
		sample, err := queryNTP(ctx, ip, targetHost, targetPort, module, logger)
		if err != nil {
			level.Warn(logger).Log("msg", "Skipping failed NTP resample", "err", err)
			break
		}
		if sample.RTTSeconds < best.RTTSeconds {
			best = sample
		}
	}

	return best, nil
}

func queryNTP(ctx context.Context, ip *net.IPAddr, targetHost, targetPort string, module config.Module, logger log.Logger) (ntpSample, error) {
	network := "udp4"
	if ip.IP.To4() == nil {
		network = "udp6"
	}

	var localAddr net.Addr
	if module.NTP.SourceIPAddress != "" {
		srcIP := net.ParseIP(module.NTP.SourceIPAddress)
		if srcIP == nil {
			return ntpSample{}, fmt.Errorf("invalid source_ip_address %q", module.NTP.SourceIPAddress)
		}
		if network == "udp6" {
			if srcIP.To4() != nil {
				return ntpSample{}, fmt.Errorf("source_ip_address %q is not IPv6", module.NTP.SourceIPAddress)
			}
			localAddr = &net.UDPAddr{IP: srcIP}
		} else {
			if srcIP.To4() == nil {
				return ntpSample{}, fmt.Errorf("source_ip_address %q is not IPv4", module.NTP.SourceIPAddress)
			}
			localAddr = &net.UDPAddr{IP: srcIP.To4()}
		}
	}

	dialer := &net.Dialer{LocalAddr: localAddr}
	serverAddr := net.JoinHostPort(ip.String(), targetPort)
	conn, err := dialer.DialContext(ctx, network, serverAddr)
	if err != nil {
		return ntpSample{}, err
	}
	defer conn.Close()

	if deadline, ok := ctx.Deadline(); ok {
		if err := conn.SetDeadline(deadline); err != nil {
			return ntpSample{}, err
		}
	}

	request := make([]byte, ntpPacketSize)
	request[0] = byte(module.NTP.ProtocolVersion<<3) | 3
	t1 := time.Now().UTC()
	writeNTPTimestamp(request[40:], t1)

	if _, err := conn.Write(request); err != nil {
		return ntpSample{}, err
	}

	response := make([]byte, ntpPacketSize)
	if _, err := io.ReadFull(conn, response); err != nil {
		return ntpSample{}, err
	}
	t4 := time.Now().UTC()

	mode := response[0] & 0x7
	if mode != 4 && mode != 5 {
		return ntpSample{}, fmt.Errorf("unexpected NTP mode %d", mode)
	}

	origin := parseNTPTimestamp(response[24:32])
	if !origin.Equal(t1) && math.Abs(origin.Sub(t1).Seconds()) > 1 {
		level.Warn(logger).Log("msg", "Origin timestamp does not match request", "origin", origin.Unix(), "request", t1.Unix())
	}

	t2 := parseNTPTimestamp(response[32:40])
	t3 := parseNTPTimestamp(response[40:48])
	rootDelay := parseShort(response[4:8], true)
	rootDispersion := parseShort(response[8:12], false)
	rtt := t4.Sub(t1).Seconds() - t3.Sub(t2).Seconds()
	offset := (t2.Sub(t1).Seconds() + t3.Sub(t4).Seconds()) / 2

	sample := ntpSample{
		DriftSeconds:              offset,
		RTTSeconds:                rtt,
		ReferenceTimestampSeconds: float64(parseNTPTimestamp(response[16:24]).UnixNano()) / float64(time.Second),
		RootDelaySeconds:          rootDelay,
		RootDispersionSeconds:     rootDispersion,
		RootDistanceSeconds:       rootDispersion + rootDelay/2 + math.Abs(rtt)/2,
		PrecisionSeconds:          math.Pow(2, float64(int8(response[3]))),
		Stratum:                   float64(response[1]),
		Leap:                      float64((response[0] >> 6) & 0x3),
		Server:                    normalizeNTPServer(targetHost, targetPort),
		ReferenceID:               parseReferenceID(response[12:16], int(response[1]), module.NTP.ProtocolVersion),
	}

	return sample, nil
}

func splitNTPAddress(target string) (string, string) {
	if host, port, err := net.SplitHostPort(target); err == nil {
		return host, port
	}
	return strings.TrimSpace(target), ntpPort
}

func normalizeNTPServer(host, port string) string {
	if port == "" || port == ntpPort {
		return host
	}
	return net.JoinHostPort(host, port)
}

func writeNTPTimestamp(dst []byte, t time.Time) {
	seconds := uint64(uint32(t.Unix() + ntpEpochOffset))
	fraction := uint64(uint32((uint64(t.Nanosecond()) << 32) / uint64(time.Second)))
	binary.BigEndian.PutUint64(dst, (seconds<<32)|fraction)
}

func parseNTPTimestamp(src []byte) time.Time {
	value := binary.BigEndian.Uint64(src)
	seconds := int64(value>>32) - ntpEpochOffset
	nanos := (int64(value&0xffffffff) * int64(time.Second)) >> 32
	return time.Unix(seconds, nanos).UTC()
}

func parseShort(src []byte, signed bool) float64 {
	if signed {
		return float64(int32(binary.BigEndian.Uint32(src))) / 65536
	}
	return float64(binary.BigEndian.Uint32(src)) / 65536
}

func parseReferenceID(src []byte, stratum int, version int) string {
	if stratum <= 1 {
		return strings.TrimRight(string(src), "\x00 ")
	}
	if version == 3 {
		return net.IP(src).String()
	}
	return fmt.Sprintf("%08x", binary.BigEndian.Uint32(src))
}
