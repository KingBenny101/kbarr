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
	{Key: "preferredQuality", Type: TypeSelect, Default: "1080p", Label: "Preferred quality", Description: "Torrent results matching this quality score higher.", Group: "general", Section: "General settings", Options: []string{"4K", "1080p", "720p", "480p", "any"}},
	{Key: "minSeeders", Type: TypeInt, Default: "1", Label: "Minimum seeders", Description: "Torrents with fewer seeders than this are ignored.", Group: "general", Section: "General settings"},
	{Key: "devMode", Type: TypeBool, Default: "false", Label: "Dev mode", Description: "Enables test torrent insertion on the Download Queue page.", Group: "general", Section: "Developer options"},

	// ── Metadata ─────────────────────────────────────────────────────────
	{Key: "anidbClient", Type: TypeString, Default: "error", Label: "Client name", Group: "metadata", Section: "AniDB"},
	{Key: "anidbVersion", Type: TypeString, Default: "error", Label: "Client version", Group: "metadata", Section: "AniDB"},
	{Key: "anidbSyncInterval", Type: TypeInt, Default: "1440", Label: "Sync interval", Description: "Minutes between AniDB title dump refreshes.", Group: "metadata", Section: "AniDB", Unit: "min"},

	// ── Indexer ──────────────────────────────────────────────────────────
	{Key: "indexerProvider", Type: TypeSelect, Default: "kbdex", Label: "Search provider", Group: "indexer", Section: "Indexer", Options: []string{"kbdex", "prowlarr"}, Widget: "segmented"},
	{Key: "prowlarrInterval", Type: TypeInt, Default: "1", Label: "Scan interval", Description: "How often the monitor table is polled for new items to search.", Group: "indexer", Section: "Indexer", Unit: "sec"},
	{Key: "matchThreshold", Type: TypeInt, Default: "80", Label: "Title match threshold", Description: "Minimum similarity (0–100) between the guessit-parsed torrent title and the anime title. Lower = more permissive.", Group: "indexer", Section: "Indexer", Unit: "%"},
	{Key: "cacheFileLimit", Type: TypeInt, Default: "10", Label: "Cache file limit", Description: "Maximum files kept in each cache/debug folder. Oldest deleted first.", Group: "indexer", Section: "Indexer"},
	{Key: "kbdexUrl", Type: TypeString, Default: "http://localhost:8000", Label: "URL", Group: "indexer", Section: "kbdex"},
	{Key: "prowlarrUrl", Type: TypeString, Default: "http://host.docker.internal:9696", Label: "URL", Group: "indexer", Section: "Prowlarr"},
	{Key: "prowlarrApiKey", Type: TypePassword, Default: "error", Label: "API key", Group: "indexer", Section: "Prowlarr"},
	{Key: "prowlarrCacheAge", Type: TypeInt, Default: "3600", Label: "Result cache age", Description: "How long Prowlarr search results are cached on disk before re-querying. Set to 0 to disable.", Group: "indexer", Section: "Prowlarr", Unit: "sec"},

	// ── Downloader ───────────────────────────────────────────────────────
	{Key: "downloaderInterval", Type: TypeInt, Default: "1", Label: "Poll interval", Description: "How often the downloader checks for pending items and updates progress.", Group: "downloader", Section: "Downloader", Unit: "sec"},
	{Key: "downloadPath", Type: TypeString, Default: "/app/downloads", Label: "Download path", Description: "Absolute path inside the container where qBittorrent saves files.", Group: "downloader", Section: "Downloader"},
	{Key: "mediaPath", Type: TypeString, Default: "/app/media", Label: "Media path", Description: "Absolute path inside the container where organised hardlinks are created.", Group: "downloader", Section: "Downloader"},
	{Key: "allowedVideoExtensions", Type: TypeString, Default: ".mkv,.mp4,.avi,.mov,.wmv,.m4v", Label: "Allowed video extensions", Description: "Comma-separated list of file extensions to hardlink on completion.", Group: "downloader", Section: "Downloader"},
	{Key: "stallTimeout", Type: TypeInt, Default: "300", Label: "Stall timeout", Description: "Remove torrents with no progress for this many seconds and re-queue. Set to 0 to disable.", Group: "downloader", Section: "Downloader", Unit: "sec"},
	{Key: "availabilityCheckInterval", Type: TypeInt, Default: "10", Label: "Availability check interval", Description: "How often the media folder is scanned to update episode availability.", Group: "downloader", Section: "Downloader", Unit: "sec"},
	{Key: "qbittorrentUrl", Type: TypeString, Default: "http://host.docker.internal:8080", Label: "URL", Group: "downloader", Section: "qBittorrent"},
	{Key: "qbittorrentUsername", Type: TypeString, Default: "", Label: "Username", Group: "downloader", Section: "qBittorrent"},
	{Key: "qbittorrentPassword", Type: TypePassword, Default: "", Label: "Password", Group: "downloader", Section: "qBittorrent"},

	// ── Hidden (defaults only; not rendered in the settings UI) ──────────

	{Key: "authUsername", Type: TypeString, Default: "", Hidden: true},
	{Key: "authPasswordHash", Type: TypePassword, Default: "", Hidden: true},
}
