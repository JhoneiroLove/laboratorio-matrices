# Resumen de arquitectura

## Componentes

La API de Matrices en Go se encarga de la autenticación, la rotación en sentido horario, la descomposición QR reducida, la correlación de solicitudes y la orquestación. La API de Estadísticas en Node.js se encarga de calcular los resúmenes y clasificar las matrices diagonales. Ambos son servicios HTTP sin estado.

```text
Cliente
  | HTTP con token de portador
  v
API de Matrices (Go) ---- HTTP con token de portador ----> API de Estadísticas (Node.js)
```

No hay un intermediario de mensajes Kafka, una caché de Redis ni una base de datos de la aplicación. El flujo síncrono es intencionalmente sin estado; consultá ADR-0002.

## Reglas numéricas

- Las entradas son matrices rectangulares no vacías de números reales finitos. JSON Schema valida parcialmente la forma; las implementaciones deben rechazar filas de distinta longitud y valores no finitos una vez decodificados.
- La rotación es exactamente de 90 grados en sentido horario. Una entrada de `m x n` se convierte en `n x m`.
- QR es una QR reducida calculada mediante reflexiones de Householder. Para una entrada `A` con forma `m x n`, `k=min(m,n)`, `Q` es `m x k`, `R` es `k x n` y `A ~= Q R`.
- Las comparaciones numéricas usan un épsilon configurable cuyo valor predeterminado actual es `1e-10`. Los valores que cumplan `abs(value) <= epsilon` pueden normalizarse a cero cuando sea necesario depurar la descomposición.
- Una matriz es diagonal solo si es cuadrada y todos los valores fuera de la diagonal cumplen `abs(value) <= epsilon`. Los elementos de la diagonal pueden ser cero. `anyDiagonal` es verdadero cuando al menos una matriz proporcionada es diagonal.
- Un resumen aplana todos los valores de su alcance. `elements` es la cantidad de elementos aplanados; `average = sum / elements`. Las estadísticas globales aplanan todas las matrices solicitadas.

## Límite del proceso

`/process` es síncrono. La API de Matrices rota de forma independiente la entrada original y calcula la QR reducida a partir de esa misma entrada. Envía `rotated`, `Q` y `R` a la API de Estadísticas, espera el resultado o hasta que se agote el plazo del servicio ascendente, y devuelve un UUID de solicitud junto con los objetos anidados `rotation`, `qr` y `statistics`. Consultá ADR-0001.

La API de Matrices reenvía ese UUID de solicitud como `X-Request-ID`, por lo que los registros y las respuestas de error de la API de Estadísticas conservan el mismo identificador de correlación a través del límite HTTP.

`/health/live` comprueba únicamente el proceso de la API de Matrices. `/health/ready` consulta con un tiempo de espera limitado la preparación de la API de Estadísticas y devuelve `503` cuando la dependencia síncrona no puede atender solicitudes.

## Límites configurables

Los límites protegen la CPU y la memoria, pero forman parte de la configuración del despliegue y no son garantías permanentes de la API. Los valores predeterminados actuales son:

| Servicio | Configuración | Valor predeterminado actual |
| --- | --- | ---: |
| API de Matrices | Cuerpo de la solicitud | 1 MiB |
| API de Matrices | Filas / columnas | 256 / 256 |
| API de Matrices | Elementos por entrada | 65,536 |
| API de Estadísticas | Cuerpo de la solicitud | 256 KiB |
| API de Estadísticas | Matrices por solicitud | 100 |
| API de Estadísticas | Filas / columnas por matriz | 1,000 / 1,000 |
| API de Estadísticas | Elementos en toda la solicitud | 1,000,000 |

Los despliegues pueden elegir valores positivos más estrictos o más amplios según los recursos medidos y los objetivos de latencia. Los clientes deben gestionar las respuestas de validación de RFC 9457 en lugar de asumir que estos valores predeterminados son universales.

## Responsabilidad de los contratos

Los archivos OpenAPI definen el comportamiento HTTP portable. Los esquemas JSON compartidos definen cargas reutilizables. Los límites de recursos específicos de cada entorno se mantienen como valores operativos predeterminados documentados, en lugar de máximos del esquema, para que la configuración pueda cambiar sin modificar de forma incorrecta el contrato portable de intercambio.
