import { notifications } from "@mantine/notifications"

export function showToast(message: string, type: "success" | "error" = "success") {
    notifications.show({
        message,
        color: type === "success" ? "blue" : "red",
        title: type === "success" ? "Success" : "Error",
    })
}