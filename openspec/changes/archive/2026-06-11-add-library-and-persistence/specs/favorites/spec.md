# Favorites Specification

## Purpose

Marcar y desmarcar pistas como favoritas y listarlas. El estado de favorito persiste
entre sesiones.

## Requirements

### Requirement: Toggle Favorite

The system MUST allow marking a track as favorite and unmarking it. Marking a track
already favorited MUST be idempotent (no duplicate); unmarking a non-favorite MUST be a
no-op without error.

#### Scenario: Mark a track as favorite

- GIVEN una pista no favorita
- WHEN el usuario la marca como favorita
- THEN queda registrada como favorita

#### Scenario: Mark again is idempotent

- GIVEN una pista ya favorita
- WHEN el usuario la marca de nuevo
- THEN sigue siendo favorita una sola vez (sin duplicado)

#### Scenario: Unmark a favorite

- GIVEN una pista favorita
- WHEN el usuario la desmarca
- THEN deja de ser favorita

#### Scenario: Unmark a non-favorite

- GIVEN una pista que no es favorita
- WHEN el usuario la desmarca
- THEN no hay cambios y no se produce error

### Requirement: List Favorites

The system MUST provide the list of favorited tracks for display.

#### Scenario: List with favorites

- GIVEN existen pistas favoritas
- WHEN el usuario solicita la lista de favoritos
- THEN se devuelven todas las pistas marcadas como favoritas

#### Scenario: List with no favorites

- GIVEN no hay pistas favoritas
- WHEN el usuario solicita la lista
- THEN se devuelve una lista vacía sin error

### Requirement: Persist Favorites

Favorite state MUST persist across restarts via the library storage layer.

#### Scenario: Favorites survive restart

- GIVEN el usuario marcó pistas como favoritas
- WHEN reinicia la aplicación
- THEN esas pistas siguen apareciendo como favoritas
