import { Box, Button, Paper, Stack, Text, Title } from "@mantine/core"
import type { ReactNode } from "react"

type EmptyStateProps = {
    icon: ReactNode
    title: string
    description: string
    actionLabel?: string
    onAction?: () => void
}

export function EmptyState({ icon, title, description, actionLabel, onAction }: EmptyStateProps) {
    return (
        <Paper withBorder radius="md" p="lg">
            <Stack align="center" ta="center" gap="sm">
                <Box style={{ width: 48, height: 48, borderRadius: 12, display: "grid", placeItems: "center", color: "var(--mantine-color-dimmed)" }}>
                    {icon}
                </Box>
                <Title order={4}>{title}</Title>
                <Text c="dimmed" maw={420}>
                    {description}
                </Text>
                {actionLabel ? <Button variant="light" color="gray" onClick={onAction}>{actionLabel}</Button> : null}
            </Stack>
        </Paper>
    )
}