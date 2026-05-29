import { ActionIcon, useComputedColorScheme, useMantineColorScheme } from "@mantine/core"
import { IconMoonFilled, IconSunFilled } from "@tabler/icons-react"

export function ColorModeToggle() {
    const { setColorScheme } = useMantineColorScheme()
    const computedColorScheme = useComputedColorScheme("light")

    return (
        <ActionIcon
            variant="subtle"
            color="gray"
            onClick={() => setColorScheme(computedColorScheme === "dark" ? "light" : "dark")}
            aria-label="Toggle color scheme"
        >
            {computedColorScheme === "dark" ? <IconSunFilled size={18} /> : <IconMoonFilled size={18} />}
        </ActionIcon>
    )
}