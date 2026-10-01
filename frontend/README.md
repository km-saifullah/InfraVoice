# InfraVoice frontend

A React + Vite + Tailwind CSS app for the InfraVoice backend: log in, speak
(or type) an infrastructure command, review the AI-generated specification,
save it, then plan/apply/destroy it against real AWS through Terraform.

## Setup

```bash
npm install
cp .env.example .env   # point VITE_API_BASE_URL at your backend if not localhost:8000
npm run dev
```

Requires the backend (`../backend`) running, with Ollama and a Whisper-compatible
server reachable as configured in the backend's own `.env` (`AI_OLLAMA_URL`,
`SPEECH_WHISPER_URL`). Voice recording requires the browser's microphone
permission and `https://` or `localhost` (the MediaRecorder / getUserMedia
APIs refuse plain `http://` on any other host).

## What's here

| Path | Purpose |
|---|---|
| `src/lib/apiClient.js` | fetch wrapper matching the backend's `{success, data}` / `{success:false, error}` envelope, with automatic access-token refresh on 401 |
| `src/lib/endpoints.js` | one function per backend route |
| `src/context/AuthContext.jsx` | login/register/logout, access token in memory, refresh token in `localStorage` |
| `src/hooks/useAudioRecorder.js` | `MediaRecorder` + live waveform amplitude sampling |
| `src/hooks/useTerraformRun.js` | polls a run until it reaches `succeeded`/`failed` (runs execute asynchronously on the backend) |
| `src/pages/VoiceCommandPage.jsx` | record or type → transcribe/parse → review → save specification |
| `src/pages/SpecificationsPage.jsx` | list/validate/delete saved specs, preview generated Terraform files |
| `src/pages/TerraformRunsPage.jsx` | validate/plan/apply/destroy, with the apply button gated behind the exact plan you just reviewed plus an explicit confirmation checkbox |

## Design notes

Dark "operator console" theme (`tailwind.config.js`: `ink`/`mist`/`signal`
tokens) rather than a generic light SaaS look, since this tool creates real,
billable cloud infrastructure and should read as precise rather than
playful. The one deliberate flourish is the live waveform during voice
recording; everything else is quiet and information-dense by design.
Status colors (amber = running, green = succeeded, red = failed) are the one
place color carries meaning, used consistently across commands, specs, and
Terraform runs.

## Not included yet

- No automated tests (the backend's test suite covers the actual business
  logic; this is a thin UI layer on top of an already-tested API).
- No TypeScript — kept as plain JSX for now; straightforward to migrate later
  if you want stronger typing against the backend's JSON shapes.
- Command history (`/commands`) is logged on every voice submission but has
  no dedicated page yet — add one if you want a "past commands" timeline.
