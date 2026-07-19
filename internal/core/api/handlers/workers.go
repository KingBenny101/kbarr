package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/kingbenny101/kbarr/internal/logger"
)

func ExportAllLogs(w http.ResponseWriter, r *http.Request) {
	var buf bytes.Buffer

	writeServiceLogs(&buf, "core", logger.GetAll())

	for _, svc := range sidecarServices {
		resp, err := httpProbe.Get(SvcAddr(svc.envKey, svc.fallback) + "/logs")
		if err != nil {
			fmt.Fprintf(&buf, "[%-10s] --- unreachable: %v ---\n", svc.name, err)
			continue
		}
		var entries []logger.LogEntry
		if err := json.NewDecoder(resp.Body).Decode(&entries); err != nil {
			fmt.Fprintf(&buf, "[%-10s] --- decode error: %v ---\n", svc.name, err)
			resp.Body.Close()
			continue
		}
		resp.Body.Close()
		writeServiceLogs(&buf, svc.name, entries)
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename=kbarr-logs.txt")
	w.Write(buf.Bytes())
}

func writeServiceLogs(buf *bytes.Buffer, service string, entries []logger.LogEntry) {
	if len(entries) == 0 {
		fmt.Fprintf(buf, "[%-10s] --- no entries ---\n", service)
		return
	}
	for _, e := range entries {
		fmt.Fprintf(buf, "[%-10s] %s  %-5s  %s%s\n", service, e.Time, e.Level, e.Message, e.Attrs)
	}
}

var httpProbe = &http.Client{Timeout: 2 * time.Second}

func probe(url string) (bool, string) {
	resp, err := httpProbe.Get(url)
	if err != nil {
		return false, err.Error()
	}
	resp.Body.Close()
	return resp.StatusCode == http.StatusOK, ""
}

var sidecarServices = []struct {
	name        string
	displayName string
	envKey      string
	fallback    string
}{
	{"metadata", "Metadata", "METADATA_ADDR", "http://localhost:8081"},
	{"indexer", "Indexer", "INDEXER_HEALTH_ADDR", "http://localhost:8082"},
	{"downloader", "Downloader", "DOWNLOADER_HEALTH_ADDR", "http://localhost:8083"},
}

func SvcAddr(envKey, fallback string) string {
	if v := os.Getenv(envKey); v != "" {
		return v
	}
	return fallback
}

type ServiceHealth struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Running     bool   `json:"running"`
	Error       string `json:"error,omitempty"`
}

func GetWorkers() func(context.Context, *struct{}) (*WorkersOutput, error) {
	return func(ctx context.Context, _ *struct{}) (*WorkersOutput, error) {
		out := []ServiceHealth{
			{Name: "core", DisplayName: "Core", Running: true},
		}
		for _, svc := range sidecarServices {
			addr := SvcAddr(svc.envKey, svc.fallback)
			running, errMsg := probe(addr + "/health")
			out = append(out, ServiceHealth{
				Name:        svc.name,
				DisplayName: svc.displayName,
				Running:     running,
				Error:       errMsg,
			})
		}
		return &WorkersOutput{Body: out}, nil
	}
}

func GetServiceLogs() func(context.Context, *ServiceNameInput) (*ServiceLogsOutput, error) {
	return func(ctx context.Context, input *ServiceNameInput) (*ServiceLogsOutput, error) {
		if input.Name == "core" {
			data, err := json.Marshal(logger.GetAll())
			if err != nil {
				return nil, huma.Error500InternalServerError("failed to encode logs", err)
			}
			return &ServiceLogsOutput{Body: json.RawMessage(data)}, nil
		}

		for _, svc := range sidecarServices {
			if svc.name == input.Name {
				resp, err := httpProbe.Get(SvcAddr(svc.envKey, svc.fallback) + "/logs")
				if err != nil {
					return nil, huma.Error502BadGateway("sidecar unreachable", err)
				}
				defer resp.Body.Close()
				data, err := io.ReadAll(resp.Body)
				if err != nil {
					return nil, huma.Error502BadGateway("failed to read sidecar response", err)
				}
				return &ServiceLogsOutput{Body: json.RawMessage(data)}, nil
			}
		}

		return nil, huma.Error404NotFound("unknown service", nil)
	}
}
