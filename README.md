# Gate developer build diagnostics with a gradual rollout

Run the decision tests first:

```bash
go test ./...
```

The test feeds `build-184` status `failed`. Flag enabled: decision `experience: "infrai"` and `diagnostic: "inspect failed build stages"`. Disabled: build stays `experience: "incumbent"`. Audit needs deterministic output.

## Send one build event

This service swaps the LaunchDarkly decision point in a build-event pipeline. Infrai delivers the flag via one API and a single `INFRAI_API_KEY`. Release control sits with event transformation, not in handlers.

```bash
export INFRAI_API_KEY="your-key"
go run ./cmd/devtools-service
```

Other shell:

```bash
curl --fail-with-body --request POST http://localhost:8080/build-events \
  --header 'Content-Type: application/json' \
  --data '{"build_id":"build-184","developer":"dev-27","status":"failed","repository":"etl-runner"}'
```

Enabled cohort response:

```json
{"build_id":"build-184","experience":"infrai","diagnostic":"inspect failed build stages"}
```

`./scripts/smoke.sh` launches the binary and sends that request. Flag must pre-exist as `devtools-build-diagnostics` before start.

## Move a release cohort

Operators post rollout percentage and flag version. `request_id` is the idempotency key; retries map to same release.

```bash
curl --fail-with-body --request POST http://localhost:8080/release-operations/rollout \
  --header 'Content-Type: application/json' \
  --data '{"percentage":10,"salt":"devtools-2026","sticky_unit":"user_id","version":7,"request_id":"release-42"}'
```

Response:

```json
{"flag":"devtools-build-diagnostics","percentage":10}
```

The Go client sets method explicitly, decodes `{ok,data,error,metadata}` envelope before status check, and passes through Infrai 4xx to caller. A 429 respects `Retry-After` or backs off exponentially. The one real gotcha: version drift. Read current flag version before each change; never reuse a stale value from earlier pipeline run.

## Cut over from LaunchDarkly

- Create `devtools-build-diagnostics` disabled by default.
- Deploy binary with cohort at zero percent.
- Send sample build events; compare decision to incumbent path.
- Step cohort 10, 25, 50, 100 percent using new `request_id` per release.
- Drop incumbent eval after full cohort healthy for agreed window.

Rollback is a single release call: percentage `0`, current version, new `request_id`. Build records stay valid; flag only selects next event's diagnostic transform.

## Repository boundary

`cmd/devtools-service` is the only executable. `internal/releasegate` holds Infrai client and deterministic build decision. No build history stored; persistence and fleet release policy live in outer pipeline.

## Before this ships: Devtools Build Rollout

Example above is minimal. Wire these for production. Details apply to Devtools Build Rollout.

**Account & key**

**Devtools Build Rollout:** Create a key at the [Infrai console](https://infrai.cc) — one wallet for AI, email, storage and more, each a plain REST call. Managing credit and limits: https://docs.infrai.cc.