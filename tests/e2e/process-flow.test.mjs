import assert from 'node:assert/strict'
import test from 'node:test'

const baseUrl = process.env.E2E_BASE_URL ?? 'http://localhost:8080'
const statisticsUrl = process.env.E2E_STATISTICS_URL ?? 'http://localhost:3000'
const skipDirectStatistics = process.env.E2E_SKIP_DIRECT_STATISTICS === 'true'
const username = process.env.DEMO_USERNAME ?? 'demo'
const password = process.env.DEMO_PASSWORD ?? 'matrix-demo-change-me'

async function rawJsonRequest(url, init = {}) {
  const response = await fetch(url, {
    ...init,
    headers: { 'content-type': 'application/json', ...init.headers },
  })
  return { response, body: await response.json() }
}

async function jsonRequest(path, init) {
  const { response, body } = await rawJsonRequest(`${baseUrl}${path}`, init)
  assert.equal(response.status, 200, JSON.stringify(body))
  return { response, body }
}

function multiply(left, right) {
  return left.map((row) => right[0].map((_, column) =>
    row.reduce((sum, value, index) => sum + value * right[index][column], 0)))
}

function assertClose(actual, expected, tolerance = 1e-9) {
  assert.equal(actual.length, expected.length)
  actual.forEach((row, rowIndex) => row.forEach((value, columnIndex) => {
    assert.ok(Math.abs(value - expected[rowIndex][columnIndex]) <= tolerance,
      `${value} no se aproxima a ${expected[rowIndex][columnIndex]}`)
  }))
}

async function waitUntilReady() {
  const deadline = Date.now() + 30_000
  while (Date.now() < deadline) {
    try {
      const response = await fetch(`${baseUrl}/health/ready`)
      if (response.ok) return
    } catch {
      // La puerta de enlace todavía puede estar iniciando.
    }
    await new Promise((resolve) => setTimeout(resolve, 500))
  }
  assert.fail(`La aplicación no quedó preparada en ${baseUrl}`)
}

test('la aplicación procesa rotación, QR y estadísticas de extremo a extremo', async () => {
  await waitUntilReady()
  const health = await rawJsonRequest(`${baseUrl}/health/ready`)
  assert.equal(health.response.status, 200)
  assert.deepEqual(health.body, { status: 'ok' })

  const page = await fetch(baseUrl)
  assert.equal(page.status, 200)
  assert.match(page.headers.get('content-security-policy') ?? '', /frame-ancestors 'none'/)

  const authResult = await jsonRequest('/api/v1/auth/token', {
    method: 'POST',
    body: JSON.stringify({ username, password }),
  })
  const auth = authResult.body
  assert.equal(auth.tokenType, 'Bearer')
  assert.ok(auth.accessToken)
  assert.equal(authResult.response.headers.get('cache-control'), 'no-store')

  const matrix = [[1, 2], [3, 4], [5, 6]]
  const authorization = `Bearer ${auth.accessToken}`
  const result = (await jsonRequest('/api/v1/matrices/process', {
    method: 'POST',
    headers: { authorization },
    body: JSON.stringify({ matrix }),
  })).body

  assert.match(result.requestId, /^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i)
  assert.deepEqual(result.rotation, {
    direction: 'clockwise',
    matrix: [[5, 3, 1], [6, 4, 2]],
  })
  assertClose(multiply(result.qr.q, result.qr.r), matrix)
  assert.equal(result.statistics.matrices.length, 3)
  assert.equal(result.statistics.global.elements, 16)
  assert.equal(typeof result.statistics.anyDiagonal, 'boolean')

  const rotation = (await jsonRequest('/api/v1/matrices/rotate', {
    method: 'POST',
    headers: { authorization },
    body: JSON.stringify({ matrix }),
  })).body
  assert.deepEqual(rotation.matrix, [[5, 3, 1], [6, 4, 2]])

  const qr = (await jsonRequest('/api/v1/matrices/qr', {
    method: 'POST',
    headers: { authorization },
    body: JSON.stringify({ matrix }),
  })).body
  assertClose(multiply(qr.q, qr.r), matrix)

  const missingToken = await rawJsonRequest(`${baseUrl}/api/v1/matrices/process`, {
    method: 'POST',
    body: JSON.stringify({ matrix: [[1]] }),
  })
  assert.equal(missingToken.response.status, 401)
  assert.match(missingToken.response.headers.get('content-type') ?? '', /application\/problem\+json/)
  assert.equal(missingToken.response.headers.get('www-authenticate'), 'Bearer')

  const ragged = await rawJsonRequest(`${baseUrl}/api/v1/matrices/process`, {
    method: 'POST',
    headers: { authorization },
    body: JSON.stringify({ matrix: [[1, 2], [3]] }),
  })
  assert.equal(ragged.response.status, 422)
  assert.match(ragged.response.headers.get('content-type') ?? '', /application\/problem\+json/)

  if (!skipDirectStatistics) {
    const unknownProperty = await rawJsonRequest(`${statisticsUrl}/api/v1/statistics`, {
      method: 'POST',
      headers: { authorization },
      body: JSON.stringify({ matrices: [{ name: 'A', values: [[1]], extra: true }] }),
    })
    assert.equal(unknownProperty.response.status, 422)
    assert.equal(unknownProperty.body.errors[0].path, '/matrices/0/extra')

    const forbiddenOrigin = await rawJsonRequest(`${statisticsUrl}/api/v1/statistics`, {
      method: 'POST',
      headers: { origin: 'https://sitio-malicioso.example' },
      body: JSON.stringify({ matrices: [{ name: 'A', values: [[1]] }] }),
    })
    assert.equal(forbiddenOrigin.response.status, 403)
  }
})
