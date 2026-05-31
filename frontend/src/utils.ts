import { notifications } from "@mantine/notifications"

export const API_URL = ""

export function resolvePosterUrl(posterUrl?: string | null): string {
    return posterUrl || "/placeholder.svg"
}

export function showToast(message: string, type: "success" | "error" = "success") {
    notifications.show({
        message,
        color: type === "success" ? "blue" : "red",
        title: type === "success" ? "Success" : "Error",
    })
}
