# API de Estadísticas

Servicio Express 5 que valida matrices identificadas por nombre y calcula estadísticas por matriz y globales. La implementación separa el dominio y el caso de uso de aplicación de los adaptadores HTTP, JWT, de configuración y de registro estructurado.

## Requisitos

- Node.js 22 o posterior
- Una clave pública RS256 en formato PEM SPKI (`-----BEGIN PUBLIC KEY-----`)

## Configuración

Definí en el entorno de ejecución las variables incluidas en `.env.example`. El servicio no carga archivos `.env` por sí mismo. `CORS_ORIGINS` es una lista de orígenes permitidos separados por comas; no se admiten comodines. Las dimensiones de las matrices y la cantidad total de elementos están limitadas por las variables `MAX_*` correspondientes.

Los JWT deben utilizar RS256 e incluir declaraciones `iss` y `aud` coincidentes, además de declaraciones `exp` y `nbf` válidas. `JWT_CLOCK_TOLERANCE_SECONDS` controla la tolerancia de desfase del reloj y su valor predeterminado es 30 segundos.

Los tiempos de espera HTTP de Node tienen valores predeterminados de 30 segundos para solicitudes completas, 10 segundos para encabezados y 5 segundos para conexiones persistentes. Se pueden modificar mediante `HTTP_REQUEST_TIMEOUT_MS`, `HTTP_HEADERS_TIMEOUT_MS` y `HTTP_KEEP_ALIVE_TIMEOUT_MS`; el tiempo de espera de los encabezados no puede superar el de la solicitud.

## API

### Estado del servicio

```text
GET /health/live
GET /health/ready
```

Ambas rutas de estado son públicas.

### Estadísticas

```text
POST /api/v1/statistics
Authorization: Bearer <JWT RS256>
Content-Type: application/json
```

Solicitud:

```json
{
  "matrices": [
    {
      "name": "ejemplo",
      "values": [[1, 0], [0, 3.5]]
    }
  ]
}
```

Los nombres deben ser únicos dentro de la solicitud, no pueden estar vacíos y admiten hasta 100 caracteres. Cada matriz debe ser rectangular y no vacía, y todos sus elementos deben ser números JSON finitos. Se rechazan las propiedades desconocidas tanto en la solicitud como en cada matriz.

Respuesta:

```json
{
  "matrices": [
    {
      "name": "ejemplo",
      "minimum": 0,
      "maximum": 3.5,
      "sum": 4.5,
      "average": 1.125,
      "elements": 4,
      "diagonal": true
    }
  ],
  "global": {
    "minimum": 0,
    "maximum": 3.5,
    "sum": 4.5,
    "average": 1.125,
    "elements": 4
  },
  "anyDiagonal": true
}
```

Una matriz se considera diagonal únicamente cuando es cuadrada y cada valor fuera de la diagonal cumple `abs(value) <= MATRIX_EPSILON`. Las sumas utilizan un algoritmo compensado, escalado y de dos pasadas para evitar desbordamientos intermedios cuando el resultado final es representable. Una estadística final no finita produce un problema de rango numérico con estado 422.

Los errores utilizan detalles de problema `application/problem+json` e incluyen el ID de la solicitud. Los errores de validación también incluyen ubicaciones similares a JSON Pointer en `errors[].path`.

## Comandos

```sh
npm test
npm run lint
npm run dev
```

Para construir el contenedor desde este directorio:

```sh
docker build -t statistics-api .
```
