package main

import (
	"bufio"
	"bytes"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"

	"golang.zx2c4.com/wireguard/device"
)

func stats(dev *device.Device) func() (any, error) {
	return func() (any, error) {
		var buf bytes.Buffer
		if err := dev.IpcGetOperation(&buf); err != nil {
			logger.Errorf("Get device config: %v", err)
			return nil, err
		}

		stats := struct {
			Endpoint               string
			LastHandshakeTimestamp int64
			ReceivedBytes          int64
			SentBytes              int64

			NumGoroutine int
			Version      string
		}{
			NumGoroutine: runtime.NumGoroutine(),
			Version:      version(),
		}

		scanner := bufio.NewScanner(&buf)
		for scanner.Scan() {
			line := scanner.Text()
			if prefix := "endpoint="; strings.HasPrefix(line, prefix) {
				stats.Endpoint = strings.TrimPrefix(line, prefix)
			}
			if prefix := "last_handshake_time_sec="; strings.HasPrefix(line, prefix) {
				stats.LastHandshakeTimestamp, _ = strconv.ParseInt(strings.TrimPrefix(line, prefix), 10, 64)
			}
			if prefix := "rx_bytes="; strings.HasPrefix(line, prefix) {
				stats.ReceivedBytes, _ = strconv.ParseInt(strings.TrimPrefix(line, prefix), 10, 64)
			}
			if prefix := "tx_bytes="; strings.HasPrefix(line, prefix) {
				stats.SentBytes, _ = strconv.ParseInt(strings.TrimPrefix(line, prefix), 10, 64)
			}
		}
		return stats, nil
	}
}

func version() string {
	info, ok := debug.ReadBuildInfo()
	if ok {
		return info.Main.Version
	}
	return "(devel)"
}
