import { useState, useEffect } from "react"
import { useLocation, Navigate } from "react-router-dom"
import { Button } from "@/components/ui/button"
import { API_URL } from "@/lib/api"
import { showToast } from "@/lib/utils"
import { AnidbSettings } from "./settings/AnidbSettings"
import { TmdbSettings } from "./settings/TmdbSettings"
import { ProwlarrSettings } from "./settings/ProwlarrSettings"
import { GeneralSettings } from "./settings/GeneralSettings"

interface Settings {
    anidbClient: string
    anidbVersion: string
    anidbSyncInterval: string
    tmdbApiKey: string
    prowlarrUrl: string
    prowlarrApiKey: string
    prowlarrInterval: string
    autoMonitorOnAdd: string
}

export function SettingsPage() {
    const location = useLocation()
    const path = location.pathname.toLowerCase()

    if (path === "/settings" || path === "/settings/") {
        return <Navigate to="/settings/general" replace />
    }

    const [settings, setSettings] = useState<Settings | null>(null)
    const [initialSettings, setInitialSettings] = useState<Settings | null>(null)
    const [loading, setLoading] = useState(true)
    const [saving, setSaving] = useState(false)

    useEffect(() => {
        fetchSettings()
    }, [])

    const fetchSettings = async (): Promise<void> => {
        setLoading(true)
        try {
            const res = await fetch(`${API_URL}/api/settings`)
            const data = await res.json()
            const settings: Settings = {
                anidbClient: data.anidbClient || "",
                anidbVersion: data.anidbVersion || "",
                anidbSyncInterval: data.anidbSyncInterval || "86400",
                tmdbApiKey: data.tmdbApiKey || "",
                prowlarrUrl: data.prowlarrUrl || "http://localhost:9696",
                prowlarrApiKey: data.prowlarrApiKey || "",
                prowlarrInterval: data.prowlarrInterval || "60",
                autoMonitorOnAdd: data.autoMonitorOnAdd || "false",
            }
            setSettings(settings)
            setInitialSettings(settings)
        } catch (err) {
            console.error("Fetch settings error:", err)
        } finally {
            setLoading(false)
        }
    }

    const updateSetting = (key: keyof Settings, value: string): void => {
        if (settings) {
            setSettings({ ...settings, [key]: value })
        }
    }

    const isDirty =
        initialSettings &&
        settings &&
        JSON.stringify(initialSettings) !== JSON.stringify(settings)

    const handleSave = async (): Promise<void> => {
        if (!settings) return
        setSaving(true)
        try {
            const res = await fetch(`${API_URL}/api/settings`, {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({
                    anidbClient: settings.anidbClient,
                    anidbVersion: settings.anidbVersion,
                    anidbSyncInterval: settings.anidbSyncInterval,
                    tmdbApiKey: settings.tmdbApiKey.trim() || "error",
                    prowlarrUrl: settings.prowlarrUrl,
                    prowlarrApiKey: settings.prowlarrApiKey.trim() || "error",
                    prowlarrInterval: settings.prowlarrInterval,
                    autoMonitorOnAdd: settings.autoMonitorOnAdd,
                }),
            })
            if (res.ok) {
                showToast("Settings saved", "success")
                await fetchSettings()
            } else {
                showToast("Save failed", "error")
            }
        } catch (err) {
            showToast("Error saving", "error")
        } finally {
            setSaving(false)
        }
    }

    if (loading || !settings) {
        return <div className="p-8 text-center italic">Loading...</div>
    }

    return (
        <div className="max-w-4xl mx-auto space-y-16 pb-20 pt-4 px-4">
            {path.includes("general") && (
                <GeneralSettings
                    autoMonitorOnAdd={settings.autoMonitorOnAdd}
                    setAutoMonitorOnAdd={(value) =>
                        updateSetting("autoMonitorOnAdd", value)
                    }
                />
            )}
            {path.includes("anidb") && (
                <AnidbSettings
                    client={settings.anidbClient}
                    setClient={(value) => updateSetting("anidbClient", value)}
                    version={settings.anidbVersion}
                    setVersion={(value) => updateSetting("anidbVersion", value)}
                    interval={settings.anidbSyncInterval}
                    setInterval={(value) =>
                        updateSetting("anidbSyncInterval", value)
                    }
                />
            )}
            {path.includes("tmdb") && (
                <TmdbSettings
                    apiKey={settings.tmdbApiKey}
                    setApiKey={(value) => updateSetting("tmdbApiKey", value)}
                    initialApiKey={initialSettings?.tmdbApiKey}
                />
            )}
            {path.includes("prowlarr") && (
                <ProwlarrSettings
                    url={settings.prowlarrUrl}
                    setUrl={(value) => updateSetting("prowlarrUrl", value)}
                    apiKey={settings.prowlarrApiKey}
                    setApiKey={(value) => updateSetting("prowlarrApiKey", value)}
                    interval={settings.prowlarrInterval}
                    setInterval={(value) =>
                        updateSetting("prowlarrInterval", value)
                    }
                    initialApiKey={initialSettings?.prowlarrApiKey}
                />
            )}
            <div className="sticky bottom-4 z-10 flex justify-end">
                <Button onClick={handleSave} disabled={!isDirty || saving}>
                    {saving
                        ? "Saving..."
                        : isDirty
                            ? "Save Changes"
                            : "No Changes"}
                </Button>
            </div>
        </div>
    )
}
