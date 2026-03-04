import { Field, FieldDescription, FieldGroup, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { Separator } from "@/components/ui/separator"

interface TmdbSettingsProps {
    apiKey: string
    setApiKey: (val: string) => void
    initialApiKey?: string
}

export function TmdbSettings({ apiKey, setApiKey, initialApiKey }: TmdbSettingsProps) {
    return (
        <section className="group space-y-6">
            <div className="flex flex-col gap-1">
                <h3 className="text-2xl font-bold tracking-tight">TMDB Settings</h3>
                <p className="text-sm text-muted-foreground">
                    Configure your The Movie Database API settings for fetching metadata.
                </p>
            </div>
            <Separator />
            <FieldGroup>
                <Field orientation="responsive" className="justify-between gap-4 md:gap-8">
                    <div className="flex-1 space-y-1">
                        <FieldLabel htmlFor="tmdbApiKey" className="text-base font-semibold">API Key</FieldLabel>
                        <FieldDescription className="text-sm">Your TMDB API Read Access Token.</FieldDescription>
                    </div>
                    <Input
                        id="tmdbApiKey"
                        type="password"
                        className="w-full md:max-w-[350px] h-10"
                        value={apiKey}
                        onChange={(e) => setApiKey(e.target.value)}
                        onFocus={() => { if (apiKey === initialApiKey) setApiKey("") }}
                        placeholder="ey..."
                    />
                </Field>
            </FieldGroup>
        </section>
    )
}
