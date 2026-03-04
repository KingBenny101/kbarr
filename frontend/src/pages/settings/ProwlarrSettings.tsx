import { Field, FieldDescription, FieldGroup, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { Separator } from "@/components/ui/separator"

interface ProwlarrSettingsProps {
    url: string
    setUrl: (val: string) => void
    apiKey: string
    setApiKey: (val: string) => void
    interval: string
    setInterval: (val: string) => void
    initialApiKey?: string
}

export function ProwlarrSettings({ url, setUrl, apiKey, setApiKey, interval, setInterval, initialApiKey }: ProwlarrSettingsProps) {
    return (
        <section className="group space-y-6">
            <div className="flex flex-col gap-1">
                <h3 className="text-2xl font-bold tracking-tight">Prowlarr Settings</h3>
                <p className="text-sm text-muted-foreground">
                    Configure Prowlarr settings for monitoring and searching torrents.
                </p>
            </div>
            <Separator />
            <FieldGroup>
                <Field orientation="responsive" className="justify-between gap-4 md:gap-8">
                    <div className="flex-1 space-y-1">
                        <FieldLabel htmlFor="prowlarrUrl" className="text-base font-semibold">URL</FieldLabel>
                        <FieldDescription className="text-sm">The base URL of your Prowlarr instance.</FieldDescription>
                    </div>
                    <Input
                        id="prowlarrUrl"
                        className="w-full md:max-w-[350px] h-10"
                        value={url}
                        onChange={(e) => setUrl(e.target.value)}
                        placeholder="http://localhost:9696"
                    />
                </Field>

                <Separator className="opacity-40" />

                <Field orientation="responsive" className="justify-between gap-4 md:gap-8">
                    <div className="flex-1 space-y-1">
                        <FieldLabel htmlFor="prowlarrApiKey" className="text-base font-semibold">API Key</FieldLabel>
                        <FieldDescription className="text-sm">Your Prowlarr API Key.</FieldDescription>
                    </div>
                    <Input
                        id="prowlarrApiKey"
                        type="password"
                        className="w-full md:max-w-[350px] h-10"
                        value={apiKey}
                        onChange={(e) => setApiKey(e.target.value)}
                        onFocus={() => { if (apiKey === initialApiKey) setApiKey("") }}
                        placeholder="api_key_..."
                    />
                </Field>

                <Separator className="opacity-40" />

                <Field orientation="responsive" className="justify-between gap-4 md:gap-8">
                    <div className="flex-1 space-y-1">
                        <FieldLabel htmlFor="prowlarrInterval" className="text-base font-semibold">Scan Interval (min)</FieldLabel>
                        <FieldDescription className="text-sm">How often the monitor worker checks Prowlarr (default: 60m).</FieldDescription>
                    </div>
                    <Input
                        id="prowlarrInterval"
                        className="w-full md:max-w-[350px] h-10"
                        value={interval}
                        onChange={(e) => setInterval(e.target.value)}
                        placeholder="60"
                    />
                </Field>
            </FieldGroup>
        </section>
    )
}
