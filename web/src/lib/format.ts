function unit(ms: number): { value: number; unit: string } {
    const abs = Math.abs(ms)
    if (abs < 60_000) return { value: Math.floor(abs / 1000), unit: "s" }
    if (abs < 3_600_000) return { value: Math.floor(abs / 60_000), unit: "m" }
    if (abs < 86_400_000) return { value: Math.floor(abs / 3_600_000), unit: "h" }
    return { value: Math.floor(abs / 86_400_000), unit: "d" }
}

export function formatRelative(ts: Date | null, now: Date): string {
    if (!ts) return "never"
    const diffMs = ts.getTime() - now.getTime()
    const { value, unit: u } = unit(diffMs)
    return diffMs < 0 ? `${value}${u} ago` : `in ${value}${u}`
}

export function formatWallClock(ts: Date): string {
    return ts.toISOString().slice(11, 19)
}

export function formatDuration(ms: number): string {
    const totalSec = Math.round(ms / 1000)
    const hours = Math.floor(totalSec / 3600)
    const minutes = Math.floor((totalSec % 3600) / 60)
    const seconds = totalSec % 60
    if (hours > 0) return `${hours}h ${String(minutes).padStart(2, "0")}m`
    if (minutes > 0) return `${minutes}m ${String(seconds).padStart(2, "0")}s`
    return `${seconds}s`
}
