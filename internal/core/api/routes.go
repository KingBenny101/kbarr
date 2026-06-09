package api

import (
	"github.com/danielgtaylor/huma/v2"
	"github.com/kingbenny101/kbarr/internal/core/api/handlers"
	"github.com/kingbenny101/kbarr/internal/core/auth"
	"github.com/kingbenny101/kbarr/internal/core/clients"
)

var secured = []map[string][]string{{"bearerAuth": {}}}

func RegisterRoutes(api huma.API, mc *clients.MetadataClient, authStore *auth.Store, version string) {
	// Public
	huma.Register(api, huma.Operation{OperationID: "health-check", Method: "GET", Path: "/api/health", Tags: []string{"system"}, Summary: "Health check"}, handlers.Health())
	huma.Register(api, huma.Operation{OperationID: "login", Method: "POST", Path: "/api/auth/login", Tags: []string{"auth"}, Summary: "Login"}, handlers.Login(authStore))

	// Auth
	huma.Register(api, huma.Operation{OperationID: "get-me", Method: "GET", Path: "/api/auth/me", Security: secured, Tags: []string{"auth"}, Summary: "Get current user"}, handlers.Me(authStore))
	huma.Register(api, huma.Operation{OperationID: "logout", Method: "POST", Path: "/api/auth/logout", Security: secured, Tags: []string{"auth"}, Summary: "Logout", DefaultStatus: 204}, handlers.Logout(authStore))
	huma.Register(api, huma.Operation{OperationID: "update-credentials", Method: "POST", Path: "/api/auth/credentials", Security: secured, Tags: []string{"auth"}, Summary: "Update credentials", DefaultStatus: 204}, handlers.ChangeCredentials())

	// System
	huma.Register(api, huma.Operation{OperationID: "get-version", Method: "GET", Path: "/api/version", Security: secured, Tags: []string{"system"}, Summary: "Get version"}, handlers.GetVersion(version))
	huma.Register(api, huma.Operation{OperationID: "check-availability", Method: "POST", Path: "/api/availability/check", Security: secured, Tags: []string{"system"}, Summary: "Trigger availability check", DefaultStatus: 204}, handlers.CheckAvailability())
	huma.Register(api, huma.Operation{OperationID: "get-workers", Method: "GET", Path: "/api/workers", Security: secured, Tags: []string{"system"}, Summary: "Get service health"}, handlers.GetWorkers())
	huma.Register(api, huma.Operation{OperationID: "get-svc-logs", Method: "GET", Path: "/api/workers/{name}/logs", Security: secured, Tags: []string{"system"}, Summary: "Get service logs"}, handlers.GetServiceLogs())

	// Search
	huma.Register(api, huma.Operation{OperationID: "search-media", Method: "GET", Path: "/api/search", Security: secured, Tags: []string{"search"}, Summary: "Search media"}, handlers.Search(mc))

	// Library
	huma.Register(api, huma.Operation{OperationID: "list-media", Method: "GET", Path: "/api/library", Security: secured, Tags: []string{"library"}, Summary: "List library"}, handlers.GetMediaList())
	huma.Register(api, huma.Operation{OperationID: "add-media", Method: "POST", Path: "/api/library", Security: secured, Tags: []string{"library"}, Summary: "Add media", DefaultStatus: 201}, handlers.AddMedia(mc))
	huma.Register(api, huma.Operation{OperationID: "get-detailed", Method: "GET", Path: "/api/library/{id}", Security: secured, Tags: []string{"library"}, Summary: "Get media details"}, handlers.GetDetailedByMediaID())
	huma.Register(api, huma.Operation{OperationID: "get-episodes", Method: "GET", Path: "/api/library/{id}/episodes", Security: secured, Tags: []string{"library"}, Summary: "Get episodes"}, handlers.GetEpisodes())
	huma.Register(api, huma.Operation{OperationID: "get-library-monitors", Method: "GET", Path: "/api/library/{id}/monitored", Security: secured, Tags: []string{"library"}, Summary: "Get monitors for library item"}, handlers.GetMonitorsByLibraryID())
	huma.Register(api, huma.Operation{OperationID: "delete-media", Method: "DELETE", Path: "/api/library/{id}", Security: secured, Tags: []string{"library"}, Summary: "Delete media", DefaultStatus: 204}, handlers.DeleteMedia())
	huma.Register(api, huma.Operation{OperationID: "update-monitor-status", Method: "PUT", Path: "/api/library/{id}/monitor", Security: secured, Tags: []string{"library"}, Summary: "Update monitor status", DefaultStatus: 204}, handlers.UpdateMonitorStatus())

	// Monitor
	huma.Register(api, huma.Operation{OperationID: "list-monitored", Method: "GET", Path: "/api/monitor", Security: secured, Tags: []string{"monitor"}, Summary: "List monitored items"}, handlers.GetMonitoredList())
	huma.Register(api, huma.Operation{OperationID: "add-monitor", Method: "POST", Path: "/api/monitor", Security: secured, Tags: []string{"monitor"}, Summary: "Add monitor", DefaultStatus: 201}, handlers.AddMonitor())
	huma.Register(api, huma.Operation{OperationID: "bulk-add-monitor", Method: "POST", Path: "/api/monitor/bulk", Security: secured, Tags: []string{"monitor"}, Summary: "Bulk add monitors", DefaultStatus: 201}, handlers.BulkAddMonitor())
	huma.Register(api, huma.Operation{OperationID: "delete-monitor", Method: "DELETE", Path: "/api/monitor/{id}", Security: secured, Tags: []string{"monitor"}, Summary: "Delete monitor", DefaultStatus: 204}, handlers.DeleteMonitor())
	huma.Register(api, huma.Operation{OperationID: "unmonitor", Method: "POST", Path: "/api/unmonitor", Security: secured, Tags: []string{"monitor"}, Summary: "Unmonitor episode", DefaultStatus: 204}, handlers.Unmonitor())
	huma.Register(api, huma.Operation{OperationID: "unmonitor-season", Method: "POST", Path: "/api/unmonitor/season", Security: secured, Tags: []string{"monitor"}, Summary: "Unmonitor season", DefaultStatus: 204}, handlers.UnmonitorSeason())

	// Downloads
	huma.Register(api, huma.Operation{OperationID: "list-downloads", Method: "GET", Path: "/api/downloads", Security: secured, Tags: []string{"downloads"}, Summary: "List downloads"}, handlers.GetDownloadList())
	huma.Register(api, huma.Operation{OperationID: "add-test-download", Method: "POST", Path: "/api/downloads/test", Security: secured, Tags: []string{"downloads"}, Summary: "Add test download", DefaultStatus: 201}, handlers.AddTestDownload())
	huma.Register(api, huma.Operation{OperationID: "trigger-downloader", Method: "POST", Path: "/api/downloads/trigger", Security: secured, Tags: []string{"downloads"}, Summary: "Trigger downloader"}, handlers.TriggerDownloader(handlers.SvcAddr("DOWNLOADER_HEALTH_ADDR", "http://localhost:8083")))
	huma.Register(api, huma.Operation{OperationID: "create-symlinks", Method: "POST", Path: "/api/downloads/{id}/symlink", Security: secured, Tags: []string{"downloads"}, Summary: "Create symlinks", DefaultStatus: 204}, handlers.CreateSymlinks(handlers.SvcAddr("DOWNLOADER_HEALTH_ADDR", "http://localhost:8083")))
	huma.Register(api, huma.Operation{OperationID: "delete-download", Method: "DELETE", Path: "/api/downloads/{id}", Security: secured, Tags: []string{"downloads"}, Summary: "Delete download", DefaultStatus: 204}, handlers.DeleteDownload())
	huma.Register(api, huma.Operation{OperationID: "clear-blacklist", Method: "DELETE", Path: "/api/downloads/blacklist", Security: secured, Tags: []string{"downloads"}, Summary: "Clear blacklist", DefaultStatus: 204}, handlers.ClearBlacklist())

	// Settings
	huma.Register(api, huma.Operation{OperationID: "get-settings", Method: "GET", Path: "/api/settings", Security: secured, Tags: []string{"settings"}, Summary: "Get settings"}, handlers.GetSettings())
	huma.Register(api, huma.Operation{OperationID: "update-settings", Method: "POST", Path: "/api/settings", Security: secured, Tags: []string{"settings"}, Summary: "Update settings", DefaultStatus: 204}, handlers.UpdateSettings())
	huma.Register(api, huma.Operation{OperationID: "test-indexer", Method: "POST", Path: "/api/settings/test/indexer", Security: secured, Tags: []string{"settings"}, Summary: "Test Prowlarr connection"}, handlers.TestIndexer(handlers.SvcAddr("INDEXER_HEALTH_ADDR", "http://localhost:8082")))
	huma.Register(api, huma.Operation{OperationID: "test-kbdex", Method: "POST", Path: "/api/settings/test/kbdex", Security: secured, Tags: []string{"settings"}, Summary: "Test kbdex connection"}, handlers.TestKbdex(handlers.SvcAddr("INDEXER_HEALTH_ADDR", "http://localhost:8082")))
	huma.Register(api, huma.Operation{OperationID: "test-downloader", Method: "POST", Path: "/api/settings/test/downloader", Security: secured, Tags: []string{"settings"}, Summary: "Test downloader connection"}, handlers.TestDownloader(handlers.SvcAddr("DOWNLOADER_HEALTH_ADDR", "http://localhost:8083")))
}
