# Resultado de QA en Docker Compose

**Fecha:** 2026-07-30  
**Entorno:** Docker Desktop, Docker Compose v2.39.2, Windows 11  
**Topología:** Nginx/Vue → API de Matrices → API de Estadísticas

## Resultado general

La aplicación superó el flujo completo ejecutado contra los contenedores reales. No se observaron respuestas `5xx`, errores fatales, reinicios ni exposición de tokens en los registros.

## Validaciones ejecutadas

- Los contenedores de la API de Matrices, la API de Estadísticas y Vue/Nginx permanecen activos; Estadísticas y web reportan estado saludable.
- `/health/ready` devuelve `200` y `{ "status": "ok" }` a través de la puerta de enlace.
- La autenticación devuelve un JWT RS256 con `Cache-Control: no-store`.
- El flujo `/process` devuelve rotación horaria, matrices `Q` y `R`, estadísticas globales y por matriz.
- La reconstrucción numérica `Q × R` se aproxima a `A` para una matriz rectangular alta.
- Los puntos de conexión individuales `/rotate` y `/qr` producen resultados correctos.
- La API de Matrices rechaza solicitudes sin token con `401`, `WWW-Authenticate` y Problem Details.
- Las matrices con filas de distinta longitud se rechazan con `422` y Problem Details.
- La API de Estadísticas rechaza propiedades desconocidas con `422` y una ruta de validación precisa.
- Un origen CORS no autorizado se rechaza con `403`.
- El `requestId` generado por la API de Matrices se propaga a los registros de la API de Estadísticas.
- El documento HTML y los recursos JavaScript/CSS responden `200`; los recursos con hash usan caché de un año.
- La respuesta web incluye CSP, protección contra marcos y encabezados de tipo de contenido.

## Contenedores observados

| Servicio | Usuario | Tamaño de imagen | Memoria aproximada en reposo |
| --- | --- | ---: | ---: |
| API de Matrices | UID 65532 | 4.73 MB | 3.4 MiB |
| API de Estadísticas | `node` | 57.7 MB | 24 MiB |
| Vue/Nginx | UID 101 | 22.4 MB | 11 MiB |

## Mejora aplicada durante QA

Las sondas de la API de Estadísticas generaban un registro cada cinco segundos. Se excluyeron `/health/live` y `/health/ready` del registro automático y el intervalo local se ajustó a treinta segundos para conservar señales operativas útiles.

También se reforzó `/health/ready` de la API de Matrices para que compruebe la disponibilidad de la API de Estadísticas. Las pruebas unitarias y de adaptador cubren las respuestas `200` y `503`; la simulación de caída sobre la imagen reconstruida queda como comprobación operativa posterior.

## Automatización

`tests/e2e/process-flow.test.mjs` reproduce las validaciones principales. La integración continua levanta Docker Compose y ejecuta esa prueba después de las pruebas unitarias, contractuales y estáticas.
