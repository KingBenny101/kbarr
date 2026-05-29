import { Card, type CardProps } from "@mantine/core"

type SectionCardProps = CardProps & {
}

export function SectionCard({ style, ...props }: SectionCardProps) {
    return (
        <Card
            radius="md"
            withBorder
            style={{
                background: "var(--mantine-color-body)",
                borderColor: "var(--mantine-color-default-border)",
                ...style,
            }}
            {...props}
        />
    )
}