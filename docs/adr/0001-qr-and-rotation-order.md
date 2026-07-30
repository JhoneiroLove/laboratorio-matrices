# ADR-0001: Orden de QR y la rotación

- Estado: Aceptado
- Fecha: 2026-07-30

## Contexto

`POST /api/v1/matrices/process` debe devolver tanto una rotación en sentido horario como una descomposición QR. Calcular QR a partir de la matriz original y hacerlo a partir de la matriz rotada produce resultados, dimensiones y estadísticas válidos pero distintos, por lo que la entrada de cada operación debe ser explícita.

La descomposición se define como QR reducida mediante reflexiones de Householder. Para una entrada de `m x n`, sea `k = min(m, n)`; `Q` tiene forma `m x k`, `R` tiene forma `k x n` y `A` es aproximadamente `Q x R`. Se prefieren las reflexiones de Householder frente al método clásico de Gram-Schmidt por su estabilidad numérica.

## Decisión

La rotación y la QR reducida son operaciones hermanas que se calculan de forma independiente a partir de la matriz original de la solicitud:

- `rotation.matrix` es la entrada original rotada 90 grados en sentido horario.
- `qr.q` y `qr.r` son los factores de la QR reducida de Householder de la entrada original, no de la salida rotada.
- La API de Matrices envía exactamente tres salidas con nombre a la API de Estadísticas: `rotated` contiene `rotation.matrix`, `Q` contiene `qr.q` y `R` contiene `qr.r`.
- La API de Estadísticas calcula un resumen global de todos los valores y un resumen por matriz para cada una de `rotated`, `Q` y `R`.

La respuesta pública del procesamiento mantiene estas responsabilidades anidadas en `rotation`, `qr` y `statistics`.

## Consecuencias

- Una entrada no cuadrada de `m x n` produce una matriz rotada de `n x m`, mientras que QR conserva las dimensiones derivadas de la original: `Q` es `m x min(m,n)` y `R` es `min(m,n) x n`.
- Las pruebas de contrato e integración pueden comprobar los nombres exactos enviados a la API de Estadísticas y reconstruir sus salidas de origen.
- Las estadísticas globales son reproducibles porque su alcance es exactamente `rotated`, `Q` y `R`.
