import { useEffect, useRef, type DependencyList } from "react"

interface PollingOptions {
    /** Base interval between successful polls, in ms. */
    interval: number
    /** Upper bound for the exponential backoff after failures. Defaults to interval * 8. */
    maxInterval?: number
    /** Called when a poll fails, with the number of consecutive failures so far.
     *  Use `failures === 1` to surface a single toast per outage rather than one per poll. */
    onError?: (consecutiveFailures: number) => void
}

/**
 * usePolling runs an async function on an interval with three robustness properties
 * the old setInterval(fetch, ms) calls lacked:
 *   - it waits for each call to finish before scheduling the next (no overlap/pile-up),
 *   - it backs off exponentially while the backend is failing (no hammering a down API),
 *   - it pauses while the tab is hidden and refetches immediately when it becomes visible.
 *
 * The polled function signals failure by returning false or throwing.
 */
export function usePolling(
    fn: () => Promise<boolean | void>,
    { interval, maxInterval, onError }: PollingOptions,
    deps: DependencyList = [],
) {
    const fnRef = useRef(fn)
    const onErrorRef = useRef(onError)
    useEffect(() => {
        fnRef.current = fn
        onErrorRef.current = onError
    })

    useEffect(() => {
        let cancelled = false
        let timer: ReturnType<typeof setTimeout> | undefined
        let failures = 0
        const cap = maxInterval ?? interval * 8

        const schedule = (ms: number) => {
            if (cancelled) return
            timer = setTimeout(tick, ms)
        }

        const tick = async () => {
            if (cancelled) return
            // Skip work while the tab is hidden; visibilitychange wakes us back up.
            if (document.hidden) {
                schedule(interval)
                return
            }
            try {
                const ok = await fnRef.current()
                if (cancelled) return
                if (ok === false) throw new Error("poll failed")
                failures = 0
                schedule(interval)
            } catch {
                if (cancelled) return
                failures += 1
                onErrorRef.current?.(failures)
                schedule(Math.min(interval * 2 ** (failures - 1), cap))
            }
        }

        const onVisible = () => {
            if (document.hidden || cancelled) return
            if (timer) clearTimeout(timer)
            failures = 0
            tick()
        }

        tick()
        document.addEventListener("visibilitychange", onVisible)

        return () => {
            cancelled = true
            if (timer) clearTimeout(timer)
            document.removeEventListener("visibilitychange", onVisible)
        }
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, deps)
}
