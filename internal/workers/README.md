# kbarr Workers

This package manages background tasks that run on a schedule (e.g., AniDB title updates, library scanning).

## How to add a new worker:

1.  **Define your worker function**: In a new file (or `worker.go`), create a function with this signature:
    ```go
    func YourWorker(cfg *config.Config, stop <-chan struct{})
    ```
    Ensure it uses a `time.Ticker` and listens to the `stop` channel.

2.  **Register it in `manager.go`**:
    - Update `Start()` to include your new worker.
    - Add a case for it in the `Reload(name string)` switch statement.

3.  **Trigger Reloads**: If the worker depends on a setting, call `WorkerMgr.Reload("your-worker-name")` in the settings API handler.
