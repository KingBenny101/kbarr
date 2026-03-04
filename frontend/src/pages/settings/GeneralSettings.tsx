import { Field, FieldDescription, FieldGroup, FieldLabel } from "@/components/ui/field"
import { Separator } from "@/components/ui/separator"
import { Switch } from "@/components/ui/switch"

interface GeneralSettingsProps {
    autoMonitorOnAdd: string
    setAutoMonitorOnAdd: (value: string) => void
}

export function GeneralSettings({ autoMonitorOnAdd, setAutoMonitorOnAdd }: GeneralSettingsProps) {
    const isEnabled = autoMonitorOnAdd === "true"

    return (
        <section className="group space-y-6">
            <div className="flex flex-col gap-1">
                <h3 className="text-2xl font-bold tracking-tight">General Settings</h3>
                <p className="text-sm text-muted-foreground">
                    Configure general application behavior.
                </p>
            </div>
            <Separator />
            <FieldGroup>
                <Field orientation="responsive" className="justify-between gap-4 md:gap-8">
                    <div className="flex-1 space-y-1">
                        <FieldLabel htmlFor="auto-monitor" className="text-base font-semibold">Auto-Monitor on Media Add</FieldLabel>
                        <FieldDescription className="text-sm">Automatically mark newly added media as monitored and trigger search.</FieldDescription>
                    </div>
                    <Switch
                        id="auto-monitor"
                        checked={isEnabled}
                        onCheckedChange={(checked) => setAutoMonitorOnAdd(checked ? "true" : "false")}
                    />
                </Field>
            </FieldGroup>
        </section>
    )
}
