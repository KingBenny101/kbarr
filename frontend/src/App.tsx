import { useState, useEffect } from "react"
import { SidebarProvider, SidebarTrigger } from "@/components/ui/sidebar"
import { AppSidebar } from "@/components/app-sidebar"
import { ThemeProvider } from "@/components/theme-provider"
import { ModeToggle } from "@/components/mode-toggle"
import { LibraryPage } from "@/pages/LibraryPage"
import { SearchPage } from "@/pages/SearchPage"

const API = "http://localhost:8282"

interface Anime {
  ID: number
  Title: string
  TitleJP?: string
  Episodes?: number
  Status: string
  AddedAt?: string
}

interface ApiVersionResponse {
  version: string
}

type ViewType = "list" | "search"

export default function App() {
  const [animeList, setAnimeList] = useState<Anime[]>([])
  const [adding, setAdding] = useState<number | null>(null)
  const [view, setView] = useState<ViewType>("list")
  const [version, setVersion] = useState<string>("v0.1.0")

  useEffect(() => {
    fetchVersion()
    if (view === "list") {
      fetchList()
    }
  }, [view])

  const fetchVersion = async (): Promise<void> => {
    try {
      const res = await fetch(`${API}/api/version`)
      const data = (await res.json()) as ApiVersionResponse
      setVersion(data.version || "v0.1.0")
    } catch (err) {
      console.error("Failed to fetch version:", err)
    }
  }

  const fetchList = async (): Promise<void> => {
    try {
      const res = await fetch(`${API}/api/anime`)
      const data = (await res.json()) as Anime[]
      setAnimeList(data || [])
    } catch (err) {
      console.error("Failed to fetch anime list:", err)
    }
  }

  const handleAdd = async (aid: number): Promise<void> => {
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
    <ThemeProvider defaultTheme="dark" storageKey="vite-ui-theme">
      <SidebarProvider>
        <AppSidebar onNavigate={setView} currentView={view} version={version} />
        <main className="flex flex-col flex-1">
          <header className="border-b p-4 flex justify-between items-center">
            <div className="flex items-center gap-2">
              <SidebarTrigger />
              <h2 className="text-lg font-semibold">
                {view === "list" ? "Library" : "Search"}
              </h2>
            </div>
            <ModeToggle />
          </header>
          <div className="flex-1 overflow-auto p-6">
            {view === "list" && (
              <LibraryPage animeList={animeList} />
            )}
            {view === "search" && (
              <SearchPage onAdd={handleAdd} adding={adding} />
            )}
          </div>
        </main>
      </SidebarProvider>
    </ThemeProvider>
  )
}
