# Leases para Guardian

Estado: motor interno. Todavía no hay IPC, así que nada fuera de la aplicación puede crear un lease.

## Qué es

Un lease es un bloqueo temporal pedido por un propietario local de confianza, por ejemplo Guardian. Sirve para que
Guardian pueda impedir un sitio de ocio mientras hay un repaso pendiente, sin editar `config.json`.

## Garantías

- Solo en memoria. No se guarda en `config.json` y no sobrevive a un reinicio.
- Solo sobre sitios que el usuario ya tiene como regla activa. No puede ser más amplio que la regla.
- Si el usuario desactiva la regla o pausa FocoLocal, el lease deja de bloquear.
- Propietario, motivo y expiración obligatorios. Máximo 60 minutos y 20 leases por propietario.
- Idempotente: repetir la misma clave devuelve el mismo lease y no lo alarga. Reusar la clave para otro sitio es un error.
- Liberación de emergencia: el propietario puede quitar todos sus leases.
- No consume el presupuesto del usuario ni cambia el conteo de sus reglas.

## API Go (`internal/limit`)

`ApplyLease(LeaseRequest, now)`, `ReleaseLease(owner, id)`, `ReleaseOwnerLeases(owner)` y `ActiveLeases(owner, now)`.

## Lo que falta

El canal IPC (pipe con nombre solo para el usuario actual, con autenticación) y los comandos `capabilities`, `health`,
`apply_lease`, `release_lease` y `emergency_release`. Irán en un PR aparte, con pruebas de contrato y de caída.
