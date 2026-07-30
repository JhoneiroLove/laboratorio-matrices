# ADR-0002: HTTP síncrono sin base de datos de la aplicación

- Estado: Aceptado
- Fecha: 2026-07-30

## Contexto

El procesamiento de matrices necesita estadísticas de un servicio Node.js desplegado por separado. Kafka agregaría entrega asíncrona, coordinación de consumidores, almacenamiento de correlación e infraestructura operativa a una solicitud cuyo contrato público es síncrono. Los requisitos actuales no necesitan reproducción, almacenamiento en búfer, finalización eventual ni un historial persistente de matrices.

Ningún dato de la aplicación necesita persistir después de completar una solicitud. Agregar Redis o una base de datos introduciría un ciclo de vida del estado, credenciales, políticas de retención o expulsión y modos de fallo sin responder a un caso de uso actual.

## Decisión

La API de Matrices en Go invoca la API de Estadísticas en Node.js de forma síncrona mediante HTTP y espera su respuesta antes de completar `/api/v1/matrices/process`.

Ninguno de los servicios usa Kafka, Redis ni una base de datos de la aplicación. Esta es una decisión de arquitectura explícita, no un marcador para infraestructura futura obligatoria. Los ID de solicitud se propagan en encabezados HTTP y cuerpos de respuesta para su correlación, pero la aplicación no persiste ni almacena en caché las cargas y los resultados de las solicitudes.

La API de Matrices debe usar tiempos de espera limitados para la conexión y las solicitudes. Los fallos del servicio ascendente se asignan a respuestas de RFC 9457: las respuestas no válidas o fallidas se asignan a `502`, y el agotamiento del plazo, a `504`.

## Consecuencias

- La latencia y disponibilidad integral de `/process` incluyen la API de Estadísticas.
- Si se introducen reintentos, deben estar limitados y ser seguros para el cálculo idempotente de estadísticas.
- El escalado horizontal permanece sin estado.
- La persistencia mediante Kafka, Redis o una base de datos requiere un nuevo ADR justificado por requisitos concretos de durabilidad, auditoría, caché, conservación del historial o procesamiento asíncrono.
