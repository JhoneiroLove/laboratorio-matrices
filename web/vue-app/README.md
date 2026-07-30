# Laboratorio de Matrices

SPA en Vue 3 y TypeScript para autenticarse ante la API de matrices, procesar matrices rectangulares y analizar su rotación, descomposición QR, estadísticas y estado diagonal.

## Requisitos

- Node.js 22 o superior
- Una API Go accesible en `http://localhost:8080` de forma predeterminada

## Desarrollo

```sh
npm install
npm run dev
```

Vite redirige `/api` hacia `VITE_MATRIX_API_PROXY` (de forma predeterminada, `http://localhost:8080`). Definí `VITE_MATRIX_API_URL` solo cuando el navegador deba llamar directamente a una API en otro origen; esa API deberá permitir CORS.

Las solicitudes a la API vencen después de 15 segundos de forma predeterminada. Definí `VITE_API_TIMEOUT_MS` antes de iniciar o compilar la SPA para usar otro valor positivo en milisegundos.

## Contrato de la API

- `POST /api/v1/auth/token` con `{ "username": string, "password": string }`; devuelve `{ "accessToken": string, "tokenType": string, "expiresIn": number }`.
- `POST /api/v1/matrices/process` con `{ "matrix": number[][] }` y `Authorization: Bearer <token>`.

La respuesta del procesamiento se consume y valida en tiempo de ejecución exactamente como la define OpenAPI: `requestId`, `rotation`, `qr` y `statistics`, con resúmenes globales y por matriz. El token de acceso se conserva solo en la memoria del módulo y desaparece al recargar, cerrar sesión o vencer según `expiresIn`.

## Pruebas

```sh
npm test
```

## Contenedor

```sh
docker build -t matrix-workbench .
docker run --rm -p 8081:8080 \
  -e MATRIX_API_URL=http://host.docker.internal:8080 \
  matrix-workbench
```

`MATRIX_API_URL` se resuelve desde el interior del contenedor web. El ejemplo apunta a una API ejecutada en el host de Docker y funciona con Docker Desktop. En Linux, agregá `--add-host=host.docker.internal:host-gateway`; también podés conectar ambos contenedores a una misma red definida por el usuario y usar el nombre del contenedor de la API, por ejemplo, `http://matrix-api:8080`. Compose puede seguir usando HTTP en su red privada; en Cloud Run, la variable debe contener la URL HTTPS del servicio de matrices.

La imagen de nginx se ejecuta sin privilegios, sirve la SPA con redirección para rutas del cliente y encabezados de seguridad, y redirige las solicitudes `/api/` y `/health/` del mismo origen hacia `MATRIX_API_URL`. Esto permite que las verificaciones integrales consulten la salud real del servicio de matrices.
