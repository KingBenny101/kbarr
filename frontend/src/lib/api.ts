export const API_URL = import.meta.env.VITE_API_URL || "";
export const ANIDB_URL = import.meta.env.VITE_ANIDB_URL || "http://localhost:8081";

export function resolvePosterUrl(posterUrl?: string | null): string {
    if (!posterUrl) {
        return "/placeholder.svg";
    }

    if (posterUrl.startsWith("http://") || posterUrl.startsWith("https://")) {
        return posterUrl;
    }

    return `${ANIDB_URL}${posterUrl}`;
}
