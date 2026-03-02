import { useState, useEffect } from "react"
import "./App.css"

const API = "http://localhost:8282"

export default function App() {
  const [query, setQuery] = useState("")
  const [searchResults, setSearchResults] = useState([])
  const [animeList, setAnimeList] = useState([])
  const [searching, setSearching] = useState(false)
  const [adding, setAdding] = useState(null)
  const [view, setView] = useState("list")
  const [version, setVersion] = useState("v0.1.0")

  useEffect(() => {
    fetchList()
    fetchVersion()
  }, [])

  async function fetchVersion() {
    try {
      const res = await fetch(`${API}/api/version`)
      const data = await res.json()
      setVersion(data.version || "v0.1.0")
    } catch (err) {
      console.error("Failed to fetch version:", err)
    }
  }

  async function fetchList() {
    try {
      const res = await fetch(`${API}/api/anime`)
      const data = await res.json()
      setAnimeList(data || [])
    } catch (err) {
      console.error("Failed to fetch anime list:", err)
    }
  }

  async function handleSearch() {
    if (!query.trim()) return
    setSearching(true)
    try {
      const res = await fetch(`${API}/api/anime/search?q=${encodeURIComponent(query)}`)
      const data = await res.json()
      setSearchResults(data || [])
    } catch (err) {
      console.error("Search failed:", err)
    } finally {
      setSearching(false)
    }
  }

  async function handleAdd(aid, title) {
    setAdding(aid)
    try {
      const res = await fetch(`${API}/api/anime`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ aid })
      })
      if (res.ok) {
        await fetchList()
        setView("list")
      }
    } catch (err) {
      console.error("Failed to add anime:", err)
    } finally {
      setAdding(null)
    }
  }

  return (
    <div className="app">
      {/* Left Sidebar */}
      <div className="sidebar">
        <div className="sidebar-header">
          <h1 className="logo">kbarr</h1>
        </div>
        <nav className="sidebar-nav">
          <button
            className={`nav-item ${view === "list" ? "active" : ""}`}
            onClick={() => {
              setView("list")
              fetchList()
            }}
          >
            <span>Library</span>
          </button>
          <button
            className={`nav-item ${view === "search" ? "active" : ""}`}
            onClick={() => setView("search")}
          >
            <span>Search</span>
          </button>
        </nav>
        <div className="sidebar-footer">
          <p className="version-text">v{version}</p>
        </div>
      </div>

      {/* Main Content */}
      <div className="main-content">
        {view === "list" && (
          <div>
            <h2 style={{ marginTop: 0, color: "var(--text-primary)" }}>
              My Library
            </h2>
            {animeList.length === 0 ? (
              <div className="empty-state">
                <p className="empty-text">
                  Head to Search to add some!
                </p>
              </div>
            ) : (
              <div className="results-grid">
                {animeList.map((anime) => (
                  <div key={anime.ID} className="anime-card">
                    <div className="anime-card-content">
                      <div className="anime-card-header">
                        <h3 className="anime-title">{anime.Title}</h3>
                        <span className="anime-aid">ID: {anime.ID}</span>
                      </div>
                      <div className="anime-info">
                        <div className="anime-info-item">
                          <span className="meta-label">Episodes</span>
                          <span>{anime.Episodes || "—"}</span>
                        </div>
                        <div className="anime-info-item">
                          <span className="meta-label">Status</span>
                          <span className="status-badge">{anime.Status}</span>
                        </div>
                        <div className="anime-info-item">
                          <span className="meta-label">Added</span>
                          <span>{anime.AddedAt ? new Date(anime.AddedAt).toLocaleDateString() : "—"}</span>
                        </div>
                        {anime.TitleJP && (
                          <div className="anime-info-item">
                            <span className="meta-label">Japanese</span>
                            <span style={{ fontSize: '0.85rem', fontStyle: 'italic' }}>{anime.TitleJP}</span>
                          </div>
                        )}
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>
        )}

        {view === "search" && (
          <div className="search-section">
            <div className="search-header">
              <div className="search-input-group">
                <input
                  type="text"
                  className="search-input"
                  value={query}
                  onChange={(e) => setQuery(e.target.value)}
                  onKeyDown={(e) => e.key === "Enter" && handleSearch()}
                  disabled={searching}
                />
                <button
                  className="btn btn-primary"
                  onClick={handleSearch}
                  disabled={searching || !query.trim()}
                >
                  {searching ? "Searching..." : "Search"}
                </button>
              </div>
            </div>

            {searchResults.length > 0 && (
              <div className="results-grid">
                {searchResults.map((result) => (
                  <div key={result.AID} className="anime-card">
                    <div className="anime-card-content">
                      <div className="anime-card-header">
                        <h3 className="anime-title">{result.Title}</h3>
                        <span className="anime-aid">AID: {result.AID}</span>
                      </div>
                      <div className="anime-actions">
                        <button
                          className="btn btn-success"
                          onClick={() => handleAdd(result.AID, result.Title)}
                          disabled={adding === result.AID}
                        >
                          {adding === result.AID ? "Adding..." : "Add to Library"}
                        </button>
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            )}

            {!searching && query && searchResults.length === 0 && (
              <div className="empty-state">
                <p className="empty-text">No anime found for "{query}"</p>
              </div>
            )}

            {!query && (
              <div className="empty-state">
                <p className="empty-text">
                  Enter an anime title to search AniDB
                </p>
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  )
}