package handlers

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"
)

type ServiceHealth struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Address     string `json:"address"`
	Running     bool   `json:"running"`
	Error       string `json:"error,omitempty"`
}

func HandleGetWorkers(metadataAddr, indexerAddr, downloaderAddr string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		services := []ServiceHealth{
			checkService("metadata", "Metadata", metadataAddr),
			checkService("indexer", "Indexer", indexerAddr),
			checkService("downloader", "Downloader", downloaderAddr),
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(services)
	}
}

func checkService(name, displayName, address string) ServiceHealth {
	health := ServiceHealth{
		Name:        name,
		DisplayName: displayName,
		Address:     address,
	}

	if strings.TrimSpace(address) == "" {
		health.Error = "address not configured"
		return health
	}

	conn, err := net.DialTimeout("tcp", address, 1500*time.Millisecond)
	if err != nil {
		health.Error = fmt.Sprintf("unreachable: %v", err)
		return health
	}
	defer conn.Close()

	health.Running = true
	return health
}
