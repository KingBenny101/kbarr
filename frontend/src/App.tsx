import { HashRouter as Router, Routes, Route, Navigate } from 'react-router-dom';
import { SearchPage } from './pages/SearchPage';
import { LibraryPage } from './pages/LibraryPage';
import { SettingsPage } from './pages/SettingsPage';
import { MediaDetailPage } from './pages/MediaDetailPage';
import { AppSidebar } from './components/app-sidebar';
import { SidebarProvider, SidebarInset, SidebarTrigger } from './components/ui/sidebar';
import { Toaster } from "@/components/ui/sonner"
import { ModeToggle } from './components/mode-toggle';
import { ThemeProvider } from './components/theme-provider';

function App() {
  return (
    <ThemeProvider defaultTheme="dark" storageKey="kbarr-theme">
      <Router>
        <SidebarProvider>
          <AppSidebar version="v0.0.1" />
          <SidebarInset>
            <header className="flex h-16 shrink-0 items-center justify-between px-4 transition-[width,height] ease-linear group-has-[[data-collapsible=icon]]/sidebar-wrapper:h-12 border-b">
              <div className="flex items-center gap-2">
                <SidebarTrigger className="-ml-1" />
              </div>
              <div className="flex items-center gap-4">
                <ModeToggle />
              </div>
            </header>
            <div className="p-4 md:p-8">
              <Routes>
                <Route path="/" element={<LibraryPage />} />
                <Route path="/search" element={<SearchPage />} />
                <Route path="/settings/*" element={<SettingsPage />} />
                <Route path="/media/:id" element={<MediaDetailPage />} />
                <Route path="*" element={<Navigate to="/" replace />} />
              </Routes>
            </div>
          </SidebarInset>
        </SidebarProvider>
        <Toaster />
      </Router>
    </ThemeProvider>
  );
}

export default App;
