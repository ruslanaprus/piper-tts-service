# Piper TTS Service

A self-hosted HTTP microservice for text-to-speech synthesis, built on top of [Piper](https://github.com/rhasspy/piper) — a fast, local, neural TTS engine. It exposes a minimal JSON API: send text and a language code, receive a WAV audio file. No cloud services, no API keys, no data leaves your infrastructure.

The service can be called from any backend or frontend that can make an HTTP request — a web application, a mobile backend, a content management system, a chatbot, a document processing pipeline, or a shell script.

---

## How it works

The service is a small Go HTTP server. On startup it reads a `voices.json` registry that maps language codes to Piper ONNX voice model files. When a synthesis request arrives it:

1. Looks up the voice model for the requested language.
2. Spawns the `piper` binary, feeds it the text via stdin, and writes the output to a temporary WAV file.
3. Streams the WAV file back to the caller as `audio/wav`.
4. Deletes the temporary file two seconds after serving it.

---

## API

### `POST /tts`

Synthesise text to speech.

**Request**

```
Content-Type: application/json
```

```json
{
  "text": "Hello, world!",
  "lang": "en"
}
```

| Field  | Type   | Required | Description |
|--------|--------|----------|-------------|
| `text` | string | yes      | The text to synthesise. |
| `lang` | string | no       | Language code matching a key in `voices.json`. Defaults to `en` if omitted or unrecognised. |

**Response**

```
Content-Type: audio/wav
```

A raw WAV audio file containing the synthesised speech. The caller decides what to do with it — save it to disk, stream it to a user, store it in a database, pass it through a pipeline.

**Error responses**

| Status | Reason |
|--------|--------|
| `400`  | `text` field is missing or empty, or the request body is not valid JSON. |
| `500`  | The Piper binary failed to run or produce output. Check service logs for details. |

---

## Quick start

### Run with Docker (standalone)

The service can be run on its own without any other components:

```bash
docker build -t piper-service .
docker run -p 5000:5000 -v ./voices:/app/voices piper-service
```

The API will be available at `http://localhost:5000`.

### Run with Docker Compose

A `docker-compose.yml` is provided in the parent repository for running the service alongside other components. To start only the TTS service:

```bash
docker compose up -d piper-service
```

### Build and run without Docker

**Requirements:** Go 1.22+, the `piper` binary on your `PATH`, and voice model files in `voices/`.

```bash
go build -o piper-service main.go
./piper-service
```

---

## Usage examples

### curl

```bash
# English
curl -X POST http://localhost:5000/tts \
  -H "Content-Type: application/json" \
  -d '{"text": "Hello, world!", "lang": "en"}' \
  --output hello.wav

# Ukrainian
curl -X POST http://localhost:5000/tts \
  -H "Content-Type: application/json" \
  -d '{"text": "Привіт, світ!", "lang": "uk"}' \
  --output hello_uk.wav
```

---

## Adding voices

Piper supports a large number of languages and voices. The full catalogue is available at [huggingface.co/rhasspy/piper-voices](https://huggingface.co/rhasspy/piper-voices).

### 1. Download a voice model

Each voice consists of two files:

- `<name>.onnx` — the model weights
- `<name>.onnx.json` — the model config

Download both and place them in the `voices/` directory.

### 2. Register the voice in `voices.json`

```json
{
  "en": {
    "name": "English (GB, Alan)",
    "model": "/app/voices/en_GB-alan-medium.onnx",
    "config": "/app/voices/en_GB-alan-medium.onnx.json"
  },
  "uk": {
    "name": "Ukrainian (Medium)",
    "model": "/app/voices/uk_UA-ukrainian_tts-medium.onnx",
    "config": "/app/voices/uk_UA-ukrainian_tts-medium.onnx.json"
  },
  "de": {
    "name": "German (Example)",
    "model": "/app/voices/de_DE-example-medium.onnx",
    "config": "/app/voices/de_DE-example-medium.onnx.json"
  }
}
```

The key (`"de"` above) is the language code callers pass in the `lang` field. It can be any string you choose — it only needs to match what your client sends.

### 3. Restart the service

```bash
docker compose restart piper-service
```

The service reads `voices.json` once at startup. Because the `voices/` directory is mounted as a volume, no image rebuild is needed — just a restart.

---

## Docker details

The image uses a two-stage build:

1. **Builder** — compiles the Go server binary using `golang:1.22`.
2. **Runtime** — `debian:bookworm-slim` with the `piper` binary downloaded from the [official GitHub release](https://github.com/rhasspy/piper/releases/tag/2023.11.14-2).

The container runs as a non-root user (`piperuser`) and exposes port `5000`.

> **Architecture note:** The Dockerfile currently downloads the `aarch64` (ARM 64-bit) Piper binary. If you are running on x86-64, update the download URL in the Dockerfile:
>
> ```
> https://github.com/rhasspy/piper/releases/download/2023.11.14-2/piper_linux_x86_64.tar.gz
> ```

---

## File structure

```
piper-service/
├── main.go          — HTTP server: /tts and /health handlers, voice registry loader
├── go.mod           — Go module definition (no external dependencies)
├── Dockerfile       — Two-stage build: Go compiler → Debian slim runtime
└── voices/
    ├── voices.json                          — language code → model file mapping
    ├── en_GB-alan-medium.onnx               — English voice model weights
    ├── en_GB-alan-medium.onnx.json          — English voice model config
    ├── uk_UA-ukrainian_tts-medium.onnx      — Ukrainian voice model weights
    └── uk_UA-ukrainian_tts-medium.onnx.json — Ukrainian voice model config
```

---

## Notes

- **No authentication.** The service has no built-in auth. It should run inside a private network (e.g. a Docker bridge network) and not be exposed directly to the public internet. If you need to expose it, put an authenticating reverse proxy in front of it.
- **Concurrency.** Each request spawns a separate `piper` process. Piper itself is single-threaded, so under heavy concurrent load requests will queue at the OS process level. For high-traffic use, consider running multiple instances behind a load balancer or putting a job queue in front of the service.
- **Temporary files.** Each synthesis writes a temporary WAV to the OS temp directory and removes it two seconds after the response is sent. The service itself stores nothing persistently — if you need caching or storage, handle that in the calling service.
- **Output format.** The service always returns WAV. If you need a different format (MP3, OGG, etc.), convert the response in your calling service using a tool such as `ffmpeg`.
