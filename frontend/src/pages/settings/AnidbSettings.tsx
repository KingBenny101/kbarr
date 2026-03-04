import { Field, FieldDescription, FieldGroup, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { Separator } from "@/components/ui/separator"

interface AnidbSettingsProps {
    client: string
    setClient: (val: string) => void
    version: string
    setVersion: (val: string) => void
    interval: string
    setInterval: (val: string) => void
}

export function AnidbSettings({ client, setClient, version, setVersion, interval, setInterval }: AnidbSettingsProps) {
    return (
        <section className="group space-y-6">
            <div className="flex flex-col gap-1">
                <h3 className="text-2xl font-bold tracking-tight">AniDB Settings</h3>
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
                        className="w-full md:max-w-[350px] h-10"
                        value={client}
                        onChange={(e) => setClient(e.target.value)}
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
                        className="w-full md:max-w-[350px] h-10"
                        value={version}
                        onChange={(e) => setVersion(e.target.value)}
                        placeholder="1"
                    />
                </Field>

                <Separator className="opacity-40" />

                <Field orientation="responsive" className="justify-between gap-4 md:gap-8">
                    <div className="flex-1 space-y-1">
                        <FieldLabel htmlFor="anidbSyncInterval" className="text-base font-semibold">Sync Interval (s)</FieldLabel>
                        <FieldDescription className="text-sm">How often to sync with AniDB (default: 86400s).</FieldDescription>
                    </div>
                    <Input
                        id="anidbSyncInterval"
                        className="w-full md:max-w-[350px] h-10"
                        value={interval}
                        onChange={(e) => setInterval(e.target.value)}
                        placeholder="86400"
                    />
                </Field>
            </FieldGroup>
        </section>
    )
}
