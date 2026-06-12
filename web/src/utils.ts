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

const TOKEN_KEY = "kbarr_token"

export function getToken(): string | null {
    return localStorage.getItem(TOKEN_KEY)
}

export function setToken(token: string) {
    localStorage.setItem(TOKEN_KEY, token)
}

export function clearToken() {
    localStorage.removeItem(TOKEN_KEY)
}

let redirectingToLogin = false

export async function apiFetch(input: string, init?: RequestInit): Promise<Response> {
    const token = getToken()
    const headers = new Headers(init?.headers)
    if (token) headers.set("Authorization", `Bearer ${token}`)

    const res = await fetch(input, { ...init, headers })

    if (res.status === 401 && !redirectingToLogin) {
        redirectingToLogin = true
        clearToken()
        showToast("Session expired — please log in again", "error")
        setTimeout(() => { window.location.href = "/login" }, 1000)
    }

    return res
}
