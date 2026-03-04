import { useState, useEffect } from "react"
import { useNavigate, useLocation, Routes, Route, BrowserRouter } from "react-router-dom"
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

const pageTitleMapping: Record<string, string> = {
  "/": "Library",
  "/search": "Search",
  "/settings": "Settings"
}

function AppContent() {
  const [version, setVersion] = useState<string>("v0.1.0")
  const navigate = useNavigate()
  const location = useLocation()

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

  const handleMediaAdded = (): void => {
    navigate("/")
  }


  return (
    <SidebarProvider>
      <AppSidebar version={version} />
      <main className="flex flex-col flex-1">
        <header className="border-b p-4 flex justify-between items-center">
          <div className="flex items-center gap-2">
            <SidebarTrigger />
            <h2 className="text-lg font-semibold">{pageTitleMapping[location.pathname] || "kbarr"}</h2>
          </div>
          <div className="flex items-center gap-2">
            <ModeToggle />

  

          </div>
        </header>
        <div className="flex-1 overflow-auto p-6">
          <Routes>
            <Route path="/" element={<LibraryPage />} />
            <Route path="/search" element={<SearchPage onMediaAdded={handleMediaAdded} />} />
            <Route path="/settings" element={<SettingsPage />} />
          </Routes>
        </div>
      </main>
    </SidebarProvider>
  )
}

export default function App() {
  return (
    <ThemeProvider defaultTheme="dark" storageKey="vite-ui-theme">
      <Toaster />
      <BrowserRouter>
        <AppContent />
      </BrowserRouter>
    </ThemeProvider>
  )
}
