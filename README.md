# Plataforma de procesamiento de matrices

Monorepositorio basado en contratos para un sistema síncrono de procesamiento de matrices.

## Arquitectura

- La **API de Matrices** está implementada en Go. Autentica a quienes realizan las solicitudes, rota matrices en sentido horario, calcula una descomposición QR reducida y orquesta las solicitudes de estadísticas.
- La **API de Estadísticas** está implementada en Node.js. Calcula resúmenes globales y por matriz, y determina si alguna matriz proporcionada es diagonal.
- El procesamiento de matrices invoca la API de Estadísticas de forma síncrona mediante HTTP. Kafka, Redis y las bases de datos de la aplicación se excluyen deliberadamente porque el flujo actual de solicitud y respuesta no requiere estado persistente ni entrega asíncrona.
- Los puntos de conexión protegidos usan tokens de portador. Las sondas de salud permanecen públicas.
- Los errores de las API usan el formato de detalles del problema de RFC 9457 con `application/problem+json`.

El flujo del proceso es el siguiente:

1. Un cliente obtiene un token de `POST /api/v1/auth/token`.
2. El cliente envía una matriz a `POST /api/v1/matrices/process`.
3. La API de Matrices rota de forma independiente la entrada original y calcula la QR reducida a partir de esa misma entrada mediante reflexiones de Householder.
4. La API de Matrices invoca `POST /api/v1/statistics` con un arreglo `matrices` que contiene las salidas identificadas como `rotated`, `Q` y `R`.
5. La API de Matrices devuelve `requestId` junto con los objetos anidados `rotation`, `qr` y `statistics`.

## Estructura del repositorio

```text
contracts/
  openapi/       Contratos de las API de los servicios
  schemas/       Definiciones compartidas de JSON Schema
docs/
  adr/           Registros de decisiones de arquitectura
  architecture/  Diseño del sistema
  deployment/    Guía de entrega y ejecución
  qa/            Controles de calidad
  security/      Línea base de seguridad
  testing/       Estrategia de pruebas
services/
  matrix-api/    Servicio de matrices y orquestación en Go
  statistics-api/ Servicio de estadísticas en Node.js
web/
  vue-app/       Cliente Vue
deploy/
  keygen/        Contenedor de inicialización de claves RSA locales
  dokploy/       Guía productiva para VPS con Dokploy
  terraform/     Infraestructura de Google Cloud Run
tests/
  e2e/           Pruebas de aceptación de la pila completa
```

## Contratos

- [`contracts/openapi/matrix-api.yaml`](contracts/openapi/matrix-api.yaml)
- [`contracts/openapi/statistics-api.yaml`](contracts/openapi/statistics-api.yaml)

Los documentos OpenAPI usan OpenAPI 3.1 y hacen referencia a esquemas JSON Schema 2020-12. Los cambios de contrato deben actualizar la documentación y las pruebas pertinentes antes de integrar cambios en la implementación.

Las decisiones principales sobre el formato de intercambio son explícitas:

- Las respuestas de tokens usan `accessToken`, `tokenType` y `expiresIn`.
- Las respuestas de procesamiento contienen `rotation`, `qr` y `statistics` de forma anidada.
- Las solicitudes a la API de Estadísticas usan `{ "matrices": [{ "name": "rotated", "values": [[...]] }] }`.
- Las respuestas exitosas de disponibilidad y preparación son `{ "status": "ok" }`.
- Los errores usan `application/problem+json` conforme a RFC 9457.

## Hoja de ruta de ejecución

1. Mantener OpenAPI, los esquemas compartidos y los ADR alineados con el comportamiento observable de los servicios.
2. Verificar de forma independiente los puntos de conexión de la API de Matrices: autenticación, rotación en sentido horario y QR reducida de Householder a partir de la entrada original.
3. Verificar el flujo síncrono de procesamiento, incluida la solicitud exacta a la API de Estadísticas con `rotated`, `Q` y `R`, y la respuesta anidada.
4. Ejecutar pruebas de contrato, numéricas, de integración, de seguridad y de rutas de error antes de una versión.
5. Configurar por entorno las claves JWT, los orígenes, los tiempos de espera y los límites de recursos de producción; los valores predeterminados son puntos de partida operativos, no garantías de la API.
6. Agregar manifiestos de despliegue y una política de observabilidad únicamente para el entorno de destino, preservando el escalado horizontal sin estado.

Introducir Kafka, Redis o una base de datos requiere un nuevo ADR respaldado por una necesidad concreta, como trabajos persistentes, reproducción, caché, auditoría o conservación del historial. No son componentes obligatorios postergados.

## Ejecución local

Docker Compose crea un par de claves RSA local en un volumen con nombre, inicia ambas API y sirve la aplicación Vue detrás de un servidor intermediario Nginx del mismo origen:

```sh
docker compose up --build
```

Abrí `http://localhost:8080` y usá las credenciales exclusivas para el entorno local:

```text
username: demo
password: matrix-demo-change-me
```

La API de Matrices en Go también se expone en `http://localhost:8081`; la API de Estadísticas se expone en `http://localhost:3000` para inspeccionar el contrato. Todos los puertos de Compose se vinculan únicamente a `127.0.0.1`. Sobrescribí las credenciales de demostración y las declaraciones JWT mediante `DEMO_USERNAME`, `DEMO_PASSWORD`, `JWT_ISSUER` y `JWT_AUDIENCE`. Nunca expongas esta pila local, que solo usa HTTP, a una red ni reutilices sus credenciales predeterminadas fuera del desarrollo local.

Cuando la pila esté en buen estado, ejecutá el flujo completo de aceptación HTTP con Node.js 22 o una versión posterior:

```sh
node --test tests/e2e/process-flow.test.mjs
```

Detené los servicios con `docker compose down`. Agregá `--volumes` únicamente si también querés eliminar el par de claves RSA local.

## Controles de calidad

```sh
# Pruebas unitarias, de adaptadores y de invariantes numéricas en Go
cd services/matrix-api && go test ./...

# Pruebas unitarias y de integración, y análisis estático en Node
cd services/statistics-api && npm test && npm run lint && npx tsc --noEmit

# Pruebas unitarias y de componentes, y comprobación de tipos en Vue
cd web/vue-app && npm test && npx vue-tsc --noEmit

# Sintaxis de contratos y despliegue
npx --yes @redocly/cli@2.5.0 lint contracts/openapi/*.yaml
docker compose config --quiet
terraform -chdir=deploy/terraform fmt -recursive -check
```

Además, GitHub Actions construye la topología completa de Compose y ejecuta el flujo integral de aceptación de autenticación, rotación, QR y estadísticas.

## Documentación en PDF

No se versiona documentación binaria. Consultá [`docs/pdf-generation.md`](docs/pdf-generation.md) para conocer el procedimiento reproducible y con versiones fijadas para convertir Markdown a PDF.

## Opciones de despliegue

- Google Cloud Run mediante [`deploy/terraform`](deploy/terraform/README.md).
- VPS autogestionada con Dokploy mediante [`compose.dokploy.yaml`](compose.dokploy.yaml) y [`deploy/dokploy`](deploy/dokploy/README.md).

El Compose de Dokploy expone únicamente Vue/Nginx a Traefik. Las API permanecen en una red interna y el par de claves JWT se conserva en un volumen respaldable.
