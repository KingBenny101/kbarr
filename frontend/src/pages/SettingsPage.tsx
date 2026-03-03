import { useState, useEffect } from "react"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Field, FieldDescription, FieldGroup, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { Separator } from "@/components/ui/separator"
import { API_URL } from "@/lib/api"
import { showToast } from "@/lib/utils"

interface Settings {
    anidbClient: string
    anidbVersion: string
}

export function SettingsPage() {
    const [initialSettings, setInitialSettings] = useState<Settings | null>(null)
    const [anidbClient, setAnidbClient] = useState<string>("")
    const [anidbVersion, setAnidbVersion] = useState<string>("")
    const [loading, setLoading] = useState(true)
    const [saving, setSaving] = useState(false)

    useEffect(() => {
        fetchSettings()
    }, [])

    const fetchSettings = async (): Promise<void> => {
        setLoading(true)
        try {
            const res = await fetch(`${API_URL}/api/settings`)
            const data = (await res.json()) as Record<string, string>
            
            const settings: Settings = {
                anidbClient: data.anidbClient,
                anidbVersion: data.anidbVersion
            }

            setInitialSettings(settings)
            setAnidbClient(settings.anidbClient)
            setAnidbVersion(settings.anidbVersion)
        } catch (err) {
            console.error("Failed to fetch settings:", err)
        } finally {
            setLoading(false)
        }
    }

    const isDirty = initialSettings && (
        anidbClient !== initialSettings.anidbClient || 
        anidbVersion !== initialSettings.anidbVersion
    )

    const handleSave = async (): Promise<void> => {
        setSaving(true)
        try {
            const res = await fetch(`${API_URL}/api/settings`, {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({
                    anidbClient: anidbClient,
                    anidbVersion: anidbVersion
                })
            })
            if (res.ok) {
                setInitialSettings({
                    anidbClient: anidbClient,
                    anidbVersion: anidbVersion
                })

                showToast("Settings saved successfully", "success")
            }

            if (!res.ok) {
                showToast("Failed to save settings", "error")
            }

        } catch (err) {
            showToast("An error occurred while saving settings", "error")   
            console.error("Failed to save settings:", err)
        } finally {
            setSaving(false)
        }
    }

    if (loading) {
        return <div className="p-8 text-center text-muted-foreground italic">Loading settings...</div>
    }

    return (
        <div className="max-w-3xl mx-auto space-y-12 pb-20 pt-4 px-4 font-sans">
            <section className="group space-y-6">
                <div>
                    <h3 className="text-xl font-bold tracking-tight">AniDB</h3>
                    <p className="text-sm text-muted-foreground">
                        Configure how your client identifies itself to the AniDB API.
                    </p>
                </div>
                <Separator />
                <FieldGroup>
                    <Field orientation="responsive" className="justify-between gap-4 md:gap-8">
                        <div className="flex-1 space-y-1">
                            <FieldLabel htmlFor="anidbclient" className="text-base font-semibold">Client Name</FieldLabel>
                            <FieldDescription className="text-sm">The registered name for your application.</FieldDescription>
                        </div>
                        <Input
                            id="anidbclient"
                            className="w-full md:max-w-[300px] h-10"
                            value={anidbClient}
                            onChange={(e) => setAnidbClient(e.target.value)}
                            placeholder="kbarr"
                        />
                    </Field>

                    <Separator className="opacity-40" />

                    <Field orientation="responsive" className="justify-between gap-4 md:gap-8">
                        <div className="flex-1 space-y-1">
                            <FieldLabel htmlFor="anidbversion" className="text-base font-semibold">Client Version</FieldLabel>
                            <FieldDescription className="text-sm">The version string used in API headers.</FieldDescription>
                        </div>
                        <Input
                            id="anidbversion"
                            className="w-full md:max-w-[300px] h-10"
                            value={anidbVersion}
                            onChange={(e) => setAnidbVersion(e.target.value)}
                            placeholder="1"
                        />
                    </Field>
                </FieldGroup>
            </section>

            <section className="space-y-6">
                <div className="flex items-center gap-2">
                    <h3 className="text-xl font-bold tracking-tight">Download Client</h3>
                    <Badge variant="outline" className="h-5 text-[10px]">Upcoming</Badge>
                </div>
                <Separator />
                <p className="text-sm text-muted-foreground italic">
                    Integration with qBittorrent and Prowlarr will be configured here.
                </p>
            </section>

            <div className="flex justify-end pt-8 border-t">
                <Button 
                    onClick={handleSave} 
                    disabled={!isDirty || saving}
                    className="w-full md:w-auto px-10 h-11 text-base transition-opacity font-bold"
                >
                    {saving ? "Saving..." : isDirty ? "Save Changes" : "No Changes"}
                </Button>
            </div>
        </div>
    )
}
