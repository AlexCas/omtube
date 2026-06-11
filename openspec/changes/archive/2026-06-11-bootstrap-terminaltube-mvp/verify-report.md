# Verify Report: bootstrap-terminaltube-mvp

## Resultado: PASS (con smoke interactivo manual pendiente)

## Comprobaciones automáticas

| Check | Comando | Resultado |
|-------|---------|-----------|
| Build | `go build ./...` | ✅ sin errores |
| Vet | `go vet ./...` | ✅ limpio |
| Formato | `gofmt -l .` | ✅ sin diffs |
| Tests unitarios | `go test ./...` | ✅ queue, search, history OK |
| IPC mpv (vivo) | `go test -tags live ./internal/player/` | ✅ handshake/volumen/Close contra mpv real |
| Búsqueda yt-dlp | comando real `ytsearch2:` | ✅ JSON con id/title/channel/duration |
| Arranque TUI | binario en pty | ✅ renderiza paneles, sin errores en log |

## Cobertura por requirement

- `youtube-search`: comando exacto validado contra yt-dlp; parseo y descarte de
  entradas sin id cubiertos por test.
- `audio-playback`: arranque mpv idle + socket + comando/respuesta + Close validados
  por el test live; auto-avance vía `end-file` implementado.
- `playback-queue`: Add/Current/Next/Prev y bordes (sin wrap) cubiertos por tests.
- `tui-shell`: render de paneles y atajos verificados; arranque limpio.
- `playback-history`: round-trip JSON y archivo inexistente cubiertos por tests.

## Pendiente de verificación manual (requiere TTY interactivo + audio + red)

El envío de teclas por pseudo-TTY no fue fiable en CI, por lo que el flujo
interactivo completo debe confirmarse a mano:

1. `./terminaltube` → `/` → "Linkin Park Numb" → Enter → suena audio.
2. `Espacio` pausa/reanuda; `+/-` cambia volumen; `n` avanza; `q` sale limpio.
3. Confirmar entradas en `~/.local/share/terminaltube/history.json`.

## Notas

- mpv sin dispositivo de audio (entornos headless) no producirá sonido; en escritorio
  Omarchy funciona con la salida por defecto.
