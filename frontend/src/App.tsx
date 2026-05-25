import { HashRouter as Router, Routes, Route, Navigate } from 'react-router-dom';
import { SearchPage } from './pages/SearchPage';
import { LibraryPage } from './pages/LibraryPage';
import { SettingsPage } from './pages/SettingsPage';
import { MediaDetailPage } from './pages/MediaDetailPage';
import { MonitorPage } from './pages/MonitorPage';
import { WorkersPage } from './pages/WorkersPage';
import { SearchQueuePage } from './pages/SearchQueuePage';
import { AppSidebar } from './components/app-sidebar';
import { SidebarProvider, SidebarTrigger } from './components/ui/sidebar';
import { Toaster } from "@/components/ui/sonner"
import { ModeToggle } from './components/mode-toggle';
import { ThemeProvider } from './components/theme-provider';

function App() {
  return (
    <ThemeProvider defaultTheme="dark" storageKey="kbarr-theme">
      <Router>
        <SidebarProvider>
          <AppSidebar />
          <main className="relative flex min-h-svh w-full flex-1 flex-col bg-background">
            <header className="flex h-16 shrink-0 items-center justify-between px-4 border-b">
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
                <Route path="/monitored" element={<MonitorPage />} />
                <Route path="/workers" element={<WorkersPage />} />
                <Route path="/search-queue" element={<SearchQueuePage />} />
                <Route path="*" element={<Navigate to="/" replace />} />
              </Routes>
            </div>
          </main>
        </SidebarProvider>
        <Toaster />
      </Router>
    </ThemeProvider>
  );
}

export default App;
