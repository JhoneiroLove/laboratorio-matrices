# API de Matrices

Servicio Go/Fiber v3 para validar, rotar y calcular descomposiciones QR reducidas de matrices rectangulares `float64`. El código se organiza en dominio, aplicación, puertos y adaptadores dentro de `internal/`.

## API

Endpoints públicos:

- `GET /health/live`
- `GET /health/ready`
- `POST /api/v1/auth/token` con `{"username":"...","password":"..."}`; devuelve `accessToken`, `tokenType` y `expiresIn`

Los siguientes endpoints requieren `Authorization: Bearer <RS256 JWT>`:

- `POST /api/v1/matrices/rotate`
- `POST /api/v1/matrices/qr`
- `POST /api/v1/matrices/process`

Las solicitudes de matrices usan esta estructura:

```json
{
  "matrix": [[1, 2], [3, 4], [5, 6]]
}
```

`/rotate` devuelve `{"direction":"clockwise","matrix":...}`. `/qr` devuelve `q` y `r` en minúsculas; para una entrada `m x n`, `q` tiene dimensiones `m x min(m,n)` y `r` tiene dimensiones `min(m,n) x n`. `/process` devuelve `requestId`, los objetos anidados `rotation` y `qr`, y la respuesta de estadísticas validada.

La solicitud de procesamiento a `POST ${ANALYTICS_BASE_URL}/api/v1/statistics` es `{"matrices":[{"name":"rotated","values":...},{"name":"Q","values":...},{"name":"R","values":...}]}`. El bearer token recibido se reenvía sin modificaciones.

Los errores usan `application/problem+json` con los campos de RFC 9457 y `request_id`. Los cuerpos JSON rechazan campos desconocidos y valores adicionales. Todas las respuestas incluyen headers de seguridad de Helmet; no se configura CORS porque el tráfico del navegador es same-origin.

## Configuración

Consultá `configs/example.env` para conocer todas las opciones. `JWT_PRIVATE_KEY_PATH`, `JWT_PUBLIC_KEY_PATH`, `DEMO_USERNAME` y `DEMO_PASSWORD` son obligatorios. Las claves deben ser claves RSA codificadas en PEM y formar el mismo par criptográfico. Las credenciales de demostración se destinan exclusivamente a este ejercicio técnico y deben proporcionarse mediante variables de entorno.

Se pueden configurar las dimensiones y la cantidad total de elementos de las matrices, el tamaño de los cuerpos HTTP, los tiempos de espera del servidor y del servicio de estadísticas, la vigencia y tolerancia temporal del JWT, y el límite de solicitudes de token. Las entradas y los resultados intermedios o finales de QR se rechazan cuando no son finitos.

## Desarrollo

```sh
go test ./...
go run ./cmd/server
```

El proceso gestiona `SIGINT` y `SIGTERM` mediante un apagado controlado con tiempo máximo. No utiliza base de datos, Kafka, Redis ni ninguna otra dependencia con estado.

## Contenedor

La imagen multietapa compila un binario estático y lo ejecuta con un usuario sin privilegios sobre una base distroless. Montá los archivos PEM privado y público en las rutas configuradas.

```sh
docker build -t matrix-api .
```
