import { describe, expect, it } from "vitest"
import { formatDuration, formatRelative, formatWallClock } from "./format"

const NOW = new Date("2026-08-01T12:00:00Z")

describe("formatRelative", () => {
    it("returns never for null", () => {
        expect(formatRelative(null, NOW)).toBe("never")
    })

    it("shows 'due now' for past timestamps within 30s", () => {
        expect(formatRelative(new Date("2026-08-01T11:59:37Z"), NOW)).toBe("due now")
    })

    it("shows 'overdue by X' for past timestamps beyond 30s", () => {
        expect(formatRelative(new Date("2026-08-01T11:59:30Z"), NOW)).toBe("overdue by 30s")
        expect(formatRelative(new Date("2026-08-01T11:55:00Z"), NOW)).toBe("overdue by 5m")
        expect(formatRelative(new Date("2026-08-01T09:00:00Z"), NOW)).toBe("overdue by 3h")
        expect(formatRelative(new Date("2026-07-30T12:00:00Z"), NOW)).toBe("overdue by 2d")
    })

    it("formats future seconds", () => {
        expect(formatRelative(new Date("2026-08-01T12:00:12Z"), NOW)).toBe("in 12s")
    })

    it("formats future minutes", () => {
        expect(formatRelative(new Date("2026-08-01T12:05:00Z"), NOW)).toBe("in 5m")
    })

    it("formats future hours", () => {
        expect(formatRelative(new Date("2026-08-01T15:00:00Z"), NOW)).toBe("in 3h")
    })

    it("formats future days", () => {
        expect(formatRelative(new Date("2026-08-03T12:00:00Z"), NOW)).toBe("in 2d")
    })

    it("uses whole units for future", () => {
        expect(formatRelative(new Date("2026-08-01T12:01:00Z"), NOW)).toBe("in 1m")
        expect(formatRelative(new Date("2026-08-01T12:05:00Z"), NOW)).toBe("in 5m")
    })
})

describe("formatWallClock", () => {
    it("formats 24-hour time with seconds", () => {
        expect(formatWallClock(new Date("2026-08-01T14:32:05Z"))).toBe("14:32:05")
    })
})

describe("formatDuration", () => {
    it("formats sub-minute durations", () => {
        expect(formatDuration(0)).toBe("0s")
        expect(formatDuration(5_000)).toBe("5s")
    })

    it("formats minute durations with seconds", () => {
        expect(formatDuration(62_000)).toBe("1m 02s")
    })

    it("formats hour durations", () => {
        expect(formatDuration(3_900_000)).toBe("1h 05m")
    })
})
