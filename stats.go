package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"net/http"
	"runtime"
	"strconv"
	"strings"

	"golang.zx2c4.com/wireguard/device"
)

func statsHandler(next http.Handler, dev *device.Device) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		if r.URL.Host != "" || r.URL.Path != "/stats" {
			next.ServeHTTP(rw, r)
			return
		}

		stats := struct {
			Endpoint               string
			LastHandshakeTimestamp int64
			ReceivedBytes          int64
			SentBytes              int64

			NumGoroutine int
		}{
			NumGoroutine: runtime.NumGoroutine(),
		}

		var buf bytes.Buffer
		if err := dev.IpcGetOperation(&buf); err != nil {
			logger.Errorf("Get device config: %v", err)
			rw.WriteHeader(http.StatusInternalServerError)
		} else {
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
			resp, _ := json.MarshalIndent(stats, "", "  ")
			rw.Header().Set("Content-Type", "application/json")
			rw.Write(append(resp, '\n'))
		}
	})
}
