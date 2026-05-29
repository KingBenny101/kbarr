import { Badge } from "@mantine/core"

export function StatusPill({ label, tone = "gray" }: { label: string; tone?: "gray" | "yellow" | "blue" | "green" | "red" | "violet" }) {
    return (
        <Badge radius="xl" variant="light" color={tone} size="sm">
            {label}
        </Badge>
    )
}