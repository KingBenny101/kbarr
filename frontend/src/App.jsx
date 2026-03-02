import { useState } from "react"

const API = "http://localhost:8282"

export default function App() {
  const [query, setQuery] = useState("")
  const [searchResults, setSearchResults] = useState([])
  const [animeList, setAnimeList] = useState([])
  const [searching, setSearching] = useState(false)
  const [adding, setAdding] = useState(null)
  const [view, setView] = useState("list")

  async function fetchList() {
    const res = await fetch(`${API}/api/anime`)
    const data = await res.json()
    setAnimeList(data || [])
  }

  async function handleSearch() {
    if (!query.trim()) return
    setSearching(true)
    const res = await fetch(`${API}/api/anime/search?q=${encodeURIComponent(query)}`)
    const data = await res.json()
    setSearchResults(data || [])
    setSearching(false)
  }

  async function handleAdd(aid, title) {
    setAdding(aid)
    await fetch(`${API}/api/anime`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ aid })
    })
    setAdding(null)
    alert(`${title} added to your list`)
  }

  useState(() => {
    fetchList()
  }, [])

  return (
    <div style={{ maxWidth: 800, margin: "0 auto", padding: 24, fontFamily: "sans-serif" }}>
      <h1>KBArr</h1>

      <div style={{ marginBottom: 16 }}>
        <button onClick={() => { setView("list"); fetchList() }}>My List</button>
        <button onClick={() => setView("search")} style={{ marginLeft: 8 }}>Search</button>
      </div>

      {view === "search" && (
        <div>
          <h2>Search Anime</h2>
          <input
            value={query}
            onChange={e => setQuery(e.target.value)}
            onKeyDown={e => e.key === "Enter" && handleSearch()}
            placeholder="Search AniDB..."
            style={{ width: 300, marginRight: 8, padding: 6 }}
          />
          <button onClick={handleSearch} disabled={searching}>
            {searching ? "Searching..." : "Search"}
          </button>

          <div style={{ marginTop: 16 }}>
            {searchResults.map(r => (
              <div key={r.AID} style={{ padding: 12, borderBottom: "1px solid #eee", display: "flex", justifyContent: "space-between", alignItems: "center" }}>
                <div>
                  <strong>{r.Title}</strong>
                  <span style={{ marginLeft: 8, color: "#888", fontSize: 12 }}>AID: {r.AID}</span>
                </div>
                <button onClick={() => handleAdd(r.AID, r.Title)} disabled={adding === r.AID}>
                  {adding === r.AID ? "Adding..." : "Add"}
                </button>
              </div>
            ))}
          </div>
        </div>
      )}

      {view === "list" && (
        <div>
          <h2>My Anime List</h2>
          {animeList.length === 0 && <p>No anime added yet.</p>}
          {animeList.map(a => (
            <div key={a.ID} style={{ padding: 12, borderBottom: "1px solid #eee" }}>
              <strong>{a.Title}</strong>
              <span style={{ marginLeft: 8, color: "#888", fontSize: 12 }}>{a.Episodes} episodes</span>
              <span style={{ marginLeft: 8, color: "#888", fontSize: 12 }}>{a.Status}</span>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}