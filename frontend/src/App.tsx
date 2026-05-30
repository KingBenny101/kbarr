import '@mantine/core/styles.css';
import { MantineProvider } from '@mantine/core';
import { Navigate, Route, Routes } from "react-router-dom"
import { PageShell } from "@/components/PageShell"
import { LibraryPage } from "@/pages/LibraryPage"
import { MediaDetailPage } from "@/pages/MediaDetailPage"
import { MonitorPage } from "@/pages/MonitorPage"
import { SearchPage } from "@/pages/SearchPage"
import { SearchQueuePage } from "@/pages/SearchQueuePage"
import { SettingsPage } from "@/pages/SettingsPage"
import { WorkersPage } from "@/pages/WorkersPage"

export default function App() {
    return (
        <MantineProvider>
            <PageShell>
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
            </PageShell>
        </MantineProvider>
    )
}

