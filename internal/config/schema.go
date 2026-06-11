package config

// SettingType is the UI rendering hint for a SettingDef.
type SettingType string

const (
	TypeString   SettingType = "string"
	TypePassword SettingType = "password"
	TypeBool     SettingType = "bool"
	TypeInt      SettingType = "int"
	TypeSelect   SettingType = "select"
)

// SettingDef describes a single application setting.
// Adding an entry to Schema is all that is needed to give a setting a default
// value and have it appear in the correct settings tab.
type SettingDef struct {
	Key         string      `json:"key"`
	Type        SettingType `json:"type"`
	Default     string      `json:"default"`
	Label       string      `json:"label,omitempty"`
	Description string      `json:"description,omitempty"`
	Group       string      `json:"group,omitempty"`
	Section     string      `json:"section,omitempty"`
	Options     []string    `json:"options,omitempty"`
	Unit        string      `json:"unit,omitempty"`
	Widget      string      `json:"widget,omitempty"` // "segmented" overrides Select rendering
	Hidden      bool        `json:"hidden,omitempty"` // defaults only; not exposed via /api/settings/schema
}

// Schema is the authoritative list of every application setting.
var Schema = []SettingDef{
	// ── General ──────────────────────────────────────────────────────────
	{Key: "autoMonitorOnAdd", Type: TypeBool, Default: "false", Label: "Auto-monitor on media add", Description: "Automatically mark newly added media as monitored and trigger search.", Group: "general", Section: "General settings"},
	{Key: "monitorSyncInterval", Type: TypeInt, Default: "1", Label: "Monitor sync interval", Description: "Interval in minutes for adding monitored items to the search queue. Minimum 1 minute.", Group: "general", Section: "General settings", Unit: "min"},
	{Key: "downloadPath", Type: TypeString, Default: "/app/downloads", Label: "Download path", Description: "Absolute path where qBittorrent saves files.", Group: "general", Section: "File management"},
	{Key: "mediaPath", Type: TypeString, Default: "/app/media", Label: "Media path", Description: "Absolute path where organised hardlinks are created.", Group: "general", Section: "File management"},
	{Key: "allowedVideoExtensions", Type: TypeString, Default: ".mkv,.mp4,.avi,.mov,.wmv,.m4v", Label: "Allowed video extensions", Description: "Comma-separated list of file extensions to hardlink on completion.", Group: "general", Section: "File management"},
	{Key: "availabilityCheckInterval", Type: TypeInt, Default: "60", Label: "Availability check interval", Description: "How often the media folder is scanned to update episode availability.", Group: "general", Section: "File management", Unit: "sec"},
	{Key: "devMode", Type: TypeBool, Default: "false", Label: "Dev mode", Description: "Enables test torrent insertion on the Download Queue page.", Group: "general", Section: "Developer options"},

	// ── Metadata ─────────────────────────────────────────────────────────
	{Key: "anidbClient", Type: TypeString, Default: "kbarr", Label: "Client name", Group: "metadata", Section: "AniDB"},
	{Key: "anidbVersion", Type: TypeString, Default: "1", Label: "Client version", Group: "metadata", Section: "AniDB"},
	{Key: "anidbSyncInterval", Type: TypeInt, Default: "1440", Label: "Sync interval", Description: "Minutes between AniDB title dump refreshes.", Group: "metadata", Section: "AniDB", Unit: "min"},

	// ── Indexer ──────────────────────────────────────────────────────────
	{Key: "prowlarrInterval", Type: TypeInt, Default: "10", Label: "Scan interval", Description: "How often the monitor table is polled for new items to search.", Group: "indexer", Section: "Indexer", Unit: "sec"},
	{Key: "matchThreshold", Type: TypeInt, Default: "80", Label: "Title match threshold", Description: "Minimum similarity (0–100) between the guessit-parsed torrent title and the anime title. Lower = more permissive.", Group: "indexer", Section: "Indexer", Unit: "%"},
	{Key: "cacheFileLimit", Type: TypeInt, Default: "10", Label: "Cache file limit", Description: "Maximum files kept in each cache/debug folder. Oldest deleted first.", Group: "indexer", Section: "Indexer"},
	{Key: "preferredQuality", Type: TypeSelect, Default: "1080p", Label: "Preferred quality", Description: "Torrent results matching this quality score higher.", Group: "indexer", Section: "Indexer", Options: []string{"4K", "1080p", "720p", "480p", "any"}},
	{Key: "minSeeders", Type: TypeInt, Default: "1", Label: "Minimum seeders", Description: "Torrents with fewer seeders than this are ignored.", Group: "indexer", Section: "Indexer"},
	{Key: "missingRetryInterval", Type: TypeInt, Default: "1440", Label: "Missing retry interval", Description: "How long to wait before re-searching an item marked missing (no qualifying torrent found). Default 1440 = 1 day.", Group: "indexer", Section: "Indexer", Unit: "min"},
	{Key: "kbdexEnabled", Type: TypeBool, Default: "true", Label: "Enabled", Description: "Search this indexer when looking for releases. Multiple indexers can be enabled at once; results are merged.", Group: "indexer", Section: "kbdex"},
	{Key: "kbdexUrl", Type: TypeString, Default: "http://host.docker.internal:8000", Label: "URL", Group: "indexer", Section: "kbdex"},
	{Key: "prowlarrEnabled", Type: TypeBool, Default: "false", Label: "Enabled", Description: "Search this indexer when looking for releases. Multiple indexers can be enabled at once; results are merged.", Group: "indexer", Section: "Prowlarr"},
	{Key: "prowlarrUrl", Type: TypeString, Default: "http://host.docker.internal:9696", Label: "URL", Group: "indexer", Section: "Prowlarr"},
	{Key: "prowlarrApiKey", Type: TypePassword, Default: "error", Label: "API key", Group: "indexer", Section: "Prowlarr"},
	{Key: "prowlarrCacheAge", Type: TypeInt, Default: "3600", Label: "Result cache age", Description: "How long Prowlarr search results are cached on disk before re-querying. Set to 0 to disable.", Group: "indexer", Section: "Prowlarr", Unit: "sec"},

	// ── Downloader ───────────────────────────────────────────────────────
	{Key: "downloaderInterval", Type: TypeInt, Default: "5", Label: "Poll interval", Description: "How often the downloader checks for pending items and updates progress.", Group: "downloader", Section: "Downloader", Unit: "sec"},
	{Key: "stallTimeout", Type: TypeInt, Default: "300", Label: "Stall timeout", Description: "Remove torrents with no progress for this many seconds and re-queue. Set to 0 to disable.", Group: "downloader", Section: "Downloader", Unit: "sec"},
	{Key: "jellyfinUrl", Type: TypeString, Default: "", Label: "URL", Description: "Jellyfin server URL. Leave blank to disable library scan after hardlink creation.", Group: "downloader", Section: "Jellyfin"},
	{Key: "jellyfinApiKey", Type: TypePassword, Default: "", Label: "API key", Group: "downloader", Section: "Jellyfin"},
	{Key: "qbittorrentUrl", Type: TypeString, Default: "http://host.docker.internal:8080", Label: "URL", Group: "downloader", Section: "qBittorrent"},
	{Key: "qbittorrentUsername", Type: TypeString, Default: "", Label: "Username", Group: "downloader", Section: "qBittorrent"},
	{Key: "qbittorrentPassword", Type: TypePassword, Default: "", Label: "Password", Group: "downloader", Section: "qBittorrent"},

	// ── Hidden (defaults only; not rendered in the settings UI) ──────────

	{Key: "authUsername", Type: TypeString, Default: "", Hidden: true},
	{Key: "authPasswordHash", Type: TypePassword, Default: "", Hidden: true},
}
