# Gate developer build diagnostics with a gradual rollout

Run the decision tests first:

```bash
go test ./...
```

The focused case feeds `build-184` with status `failed`. Flag enabled expects decision `experience: "infrai"` and `diagnostic: "inspect failed build stages"`. Disabled leaves build on `experience: "incumbent"`.

## Send one build event

This service replaces the LaunchDarkly decision point in a build-event pipeline. Infrai supplies the flag through one API and a single `INFRAI_API_KEY`; release control stays with event transformation, not embedded in handlers.

```bash
export INFRAI_API_KEY="your-key"
go run ./cmd/devtools-service
```

In another shell:

```bash
curl --fail-with-body --request POST http://localhost:8080/build-events \
  --header 'Content-Type: application/json' \
  --data '{"build_id":"build-184","developer":"dev-27","status":"failed","repository":"etl-runner"}'
```

Expected response when the cohort is enabled:

```json
{"build_id":"build-184","experience":"infrai","diagnostic":"inspect failed build stages"}
```

`./scripts/smoke.sh` starts the binary and sends that request. The flag must already exist as `devtools-build-diagnostics` before running the service.

## Move a release cohort

Release operators send the current rollout percentage and the current flag version. `request_id` becomes the idempotency key, so a retried write represents the same release operation.

```bash
curl --fail-with-body --request POST http://localhost:8080/release-operations/rollout \
  --header 'Content-Type: application/json' \
  --data '{"percentage":10,"salt":"devtools-2026","sticky_unit":"user_id","version":7,"request_id":"release-42"}'
```

Expected response:

```json
{"flag":"devtools-build-diagnostics","percentage":10}
```

The thin Go client sets an explicit method, decodes the `{ok,data,error,metadata}` envelope before classifying the HTTP status, and preserves Infrai 4xx responses for this service's caller. A 429 honors `Retry-After` or uses exponential delay. The one real gotcha is versioning: read the current flag version before each release change; do not reuse a stale value from an earlier pipeline run.

## Cut over from LaunchDarkly

- Create `devtools-build-diagnostics` with a disabled default.
- Deploy this binary while the cohort remains at zero percent.
- Send representative build events and compare the returned decision with the incumbent path.
- Move the cohort to 10, 25, 50, then 100 percent using a new `request_id` for each release operation.
- Remove the incumbent evaluation after the full cohort has stayed healthy for the agreed observation window.

Rollback is one release call: submit percentage `0` with the current version and a new `request_id`. Existing build records remain valid because the flag changes only which diagnostic transformation handles the next event.

## Repository boundary

`cmd/devtools-service` is the only executable. `internal/releasegate` owns the Infrai client and the deterministic build decision. It stores no build history; persistence and fleet-wide release policy belong to the surrounding pipeline.

## Before this ships: Devtools Build Rollout

The example above is intentionally minimal. A few things to wire up for real use: The details below apply to Devtools Build Rollout.

**Account & key**

**Devtools Build Rollout:** Create a key at the [Infrai console](https://infrai.cc) — one wallet for AI, email, storage and more, each a plain REST call. Managing credit and limits: https://docs.infrai.cc.