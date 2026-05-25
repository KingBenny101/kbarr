import { Field, FieldDescription, FieldGroup, FieldLabel } from "@/components/ui/field"
import { Separator } from "@/components/ui/separator"
import { Switch } from "@/components/ui/switch"
import { Input } from "@/components/ui/input"

interface GeneralSettingsProps {
    autoMonitorOnAdd: string
    setAutoMonitorOnAdd: (value: string) => void
    monitorSyncInterval: string
    setMonitorSyncInterval: (value: string) => void
}

export function GeneralSettings({ 
    autoMonitorOnAdd, 
    setAutoMonitorOnAdd,
    monitorSyncInterval,
    setMonitorSyncInterval
}: GeneralSettingsProps) {
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

                <Field orientation="responsive" className="justify-between gap-4 md:gap-8">
                    <div className="flex-1 space-y-1">
                        <FieldLabel htmlFor="monitor-interval" className="text-base font-semibold">Monitor Sync Interval</FieldLabel>
                        <FieldDescription className="text-sm">Interval (in minutes) for the background worker to add monitored items to the search queue. Minimum 1 minute.</FieldDescription>
                    </div>
                    <div className="flex items-center gap-2">
                        <Input
                            id="monitor-interval"
                            type="text"
                            value={monitorSyncInterval}
                            onChange={(e) => {
                                const val = e.target.value;
                                if (val === "" || /^[0-9]+$/.test(val)) {
                                    setMonitorSyncInterval(val);
                                }
                            }}
                            className="w-24 bg-background border rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-ring"
                        />
                        <span className="text-sm text-muted-foreground whitespace-nowrap">min</span>
                    </div>
                </Field>
            </FieldGroup>
        </section>
    )
}
