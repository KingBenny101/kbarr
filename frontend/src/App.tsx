import { useState, useEffect } from "react"
import { SidebarProvider, SidebarTrigger } from "@/components/ui/sidebar"
import { AppSidebar } from "@/components/app-sidebar"
import { ThemeProvider } from "@/components/theme-provider"
import { ModeToggle } from "@/components/mode-toggle"
import { LibraryPage } from "@/pages/LibraryPage"
import { SearchPage } from "@/pages/SearchPage"
import { SettingsPage } from "@/pages/SettingsPage"
import { API_URL } from "@/lib/api"
import { Toaster } from "@/components/ui/sonner"

interface ApiVersionResponse {
  version: string
}

type ViewType = "list" | "search" | "settings"

const pageTitle: Record<ViewType, string> = {
  list: "Library",
  search: "Search",
  settings: "Settings"
}

export default function App() {
  const [view, setView] = useState<ViewType>("list")
  const [version, setVersion] = useState<string>("v0.1.0")

  useEffect(() => {
    fetchVersion()
  }, [])

  const fetchVersion = async (): Promise<void> => {
    try {
      const res = await fetch(`${API_URL}/api/version`)
      const data = (await res.json()) as ApiVersionResponse
      setVersion(data.version || "v0.1.0")
    } catch (err) {
      console.error("Failed to fetch version:", err)
    }
  }

  const handleAnimeAdded = (): void => {
    setView("list")
  }

  return (
    <ThemeProvider defaultTheme="dark" storageKey="vite-ui-theme">
      <Toaster />
      <SidebarProvider>
        <AppSidebar onNavigate={setView} currentView={view} version={version} />
        <main className="flex flex-col flex-1">
          <header className="border-b p-4 flex justify-between items-center">
            <div className="flex items-center gap-2">
              <SidebarTrigger />
              <h2 className="text-lg font-semibold">{pageTitle[view]}</h2>
            </div>
            <ModeToggle />
          </header>
          <div className="flex-1 overflow-auto p-6">
            {view === "list" && (
              <LibraryPage />
            )}
            {view === "search" && (
              <SearchPage onAnimeAdded={handleAnimeAdded} />
            )}
            {view === "settings" && (
              <SettingsPage />
            )}
          </div>
        </main>
      </SidebarProvider>
    </ThemeProvider>
  )
}
