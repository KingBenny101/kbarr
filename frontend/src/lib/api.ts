export const API_URL = ""

export function resolvePosterUrl(posterUrl?: string | null): string {
    if (!posterUrl) {
        return "/placeholder.svg"
    }

    if (posterUrl.startsWith("http://") || posterUrl.startsWith("https://")) {
        return posterUrl
    }

    return posterUrl
}