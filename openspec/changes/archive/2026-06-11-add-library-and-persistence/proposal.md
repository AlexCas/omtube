# Proposal: Library and Persistence (Fase 2)

## Intent

El MVP reproduce música pero no recuerda nada salvo un historial plano en JSON. Los
usuarios quieren guardar playlists, marcar favoritos y conservar su biblioteca de
forma fiable entre sesiones. Esta fase introduce persistencia en SQLite y la capa de
biblioteca personal.

## Scope

### In Scope
- Playlists locales: crear, renombrar, borrar, añadir/quitar pistas, reproducir.
- Favoritos: marcar/desmarcar pistas y listarlas.
- Historial persistente y navegable (migrado desde JSON a SQLite).
- Base de datos SQLite local con esquema versionado (migraciones simples).
- UI: vista/modo de biblioteca (playlists, favoritos, historial) y comandos asociados.

### Out of Scope
- Caché de descargas, letras, portadas, Discord (Fase 3).
- Sincronización en la nube; multi-dispositivo.
- SponsorBlock (descartado por el usuario).

## Capabilities

### New Capabilities
- `playlists`: gestión de listas locales y reproducción como cola.
- `favorites`: marcar y listar pistas favoritas.
- `library-persistence`: capa SQLite (esquema, migraciones, repositorios).

### Modified Capabilities
- `playback-history`: pasa de archivo JSON a almacenamiento SQLite, con vista navegable.
- `tui-shell`: añade modo/panel de biblioteca y atajos para playlists/favoritos.

## Approach

Nuevo paquete `internal/storage` con SQLite vía `modernc.org/sqlite` (pure Go, sin
cgo → mantiene binario único). Repositorios por entidad (tracks, playlists,
favorites, history). `internal/history` se reimplementa sobre el repositorio. La UI
suma un modo "biblioteca" reutilizando `queue.Add` para cargar playlists.

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `internal/storage` | New | DB SQLite, esquema, migraciones, repos |
| `internal/history` | Modified | Persistir vía storage en vez de JSON |
| `internal/playlist`, `internal/favorites` | New | Lógica de dominio |
| `internal/ui` | Modified | Modo biblioteca + atajos |
| `internal/config` | Modified | Ruta del archivo de DB |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| cgo rompe binario único | Med | Usar `modernc.org/sqlite` (pure Go) |
| Migración del historial JSON existente | Low | Importar `history.json` al primer arranque y conservar respaldo |
| Crecimiento de complejidad en UI | Med | Modo biblioteca aislado; reutilizar list de bubbles |

## Rollback Plan

La DB vive en `~/.local/share/terminaltube/library.db`. Revertir = volver al binario
de Fase 1 y conservar el `history.json` (no se borra al migrar).

## Dependencies
- `modernc.org/sqlite` (pure Go).
- Reuso de `search.Result`, `queue.Queue`, `config` del MVP.

## Success Criteria
- [ ] Crear una playlist, añadirle pistas y reproducirla como cola.
- [ ] Marcar/desmarcar favoritos y listarlos.
- [ ] Historial persistente y navegable tras reiniciar.
- [ ] Datos sobreviven reinicios en SQLite; binario sigue siendo único (sin cgo).
- [ ] Tests de repositorios (CRUD) en verde.
