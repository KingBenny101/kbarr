package handlers

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
)

type TriggerCycleInput struct {
	Service string `path:"service"`
	Cycle   string `path:"cycle"`
}

// TriggerCycle wakes the given cycle's poll loop so it runs immediately.
// Core cycles are triggered in-process; worker-service cycles are proxied to
// the service's /trigger endpoint. Unknown service/cycle → 404; an unreachable
// service → 502.
func TriggerCycle(coreTriggers map[string]func(), indexerAddr, downloaderAddr, metadataAddr string) func(context.Context, *TriggerCycleInput) (*struct{}, error) {
	return func(ctx context.Context, in *TriggerCycleInput) (*struct{}, error) {
		slog.Info("Cycle triggered", "service", in.Service, "cycle", in.Cycle)
		switch in.Service {
		case "core":
			fn, ok := coreTriggers[in.Cycle]
			if !ok {
				return nil, huma.Error404NotFound(fmt.Sprintf("unknown core cycle %q", in.Cycle))
			}
			fn()
			return nil, nil
		case "indexer":
			if in.Cycle != "monitor_poll" && in.Cycle != "process_missing" {
				return nil, huma.Error404NotFound(fmt.Sprintf("unknown indexer cycle %q", in.Cycle))
			}
			return proxyTrigger(ctx, indexerAddr)
		case "downloader":
			if in.Cycle != "downloader_poll" {
				return nil, huma.Error404NotFound(fmt.Sprintf("unknown downloader cycle %q", in.Cycle))
			}
			return proxyTrigger(ctx, downloaderAddr)
		case "metadata":
			if in.Cycle != "anidb_sync" {
				return nil, huma.Error404NotFound(fmt.Sprintf("unknown metadata cycle %q", in.Cycle))
			}
			return proxyTrigger(ctx, metadataAddr)
		default:
			return nil, huma.Error404NotFound(fmt.Sprintf("unknown service %q", in.Service))
		}
	}
}

func proxyTrigger(ctx context.Context, addr string) (*struct{}, error) {
	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, addr+"/trigger", nil)
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to build trigger request", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, huma.Error502BadGateway("service unreachable: "+err.Error(), err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, huma.Error502BadGateway(fmt.Sprintf("service returned status %d", resp.StatusCode))
	}
	return nil, nil
}