const http = require('node:http')
const net = require('node:net')
const path = require('node:path')
const { spawn } = require('node:child_process')

const electronBinary = require('electron')

const rootDir = path.resolve(__dirname, '..')
const frontendDir = path.join(rootDir, 'frontend')
const npmCommand = process.platform === 'win32' ? 'npm.cmd' : 'npm'
const baseEnv = { ...process.env }

delete baseEnv.ELECTRON_RUN_AS_NODE

let frontendProcess = null
let electronProcess = null
let shuttingDown = false

async function getFreePort() {
  return new Promise((resolve, reject) => {
    const server = net.createServer()

    server.once('error', reject)
    server.listen(0, '127.0.0.1', () => {
      const address = server.address()
      const port = typeof address === 'object' && address ? address.port : null

      server.close((closeErr) => {
        if (closeErr) {
          reject(closeErr)
          return
        }
        resolve(port)
      })
    })
  })
}

async function waitForUrl(url, timeoutMs = 30000) {
  const startedAt = Date.now()

  while (Date.now() - startedAt < timeoutMs) {
    const ready = await new Promise((resolve) => {
      const req = http.get(url, (res) => {
        res.resume()
        resolve(true)
      })

      req.on('error', () => resolve(false))
      req.setTimeout(1000, () => {
        req.destroy()
        resolve(false)
      })
    })

    if (ready) {
      return
    }

    await new Promise((resolve) => setTimeout(resolve, 250))
  }

  throw new Error(`Timed out waiting for ${url}`)
}

function stopChild(child) {
  if (!child || child.killed) {
    return
  }

  child.kill('SIGTERM')
  setTimeout(() => {
    if (!child.killed) {
      child.kill('SIGKILL')
    }
  }, 3000)
}

function shutdown(code = 0) {
  if (shuttingDown) {
    return
  }

  shuttingDown = true
  stopChild(electronProcess)
  stopChild(frontendProcess)

  setTimeout(() => {
    process.exit(code)
  }, 50)
}

async function main() {
  const backendPort = await getFreePort()
  const rendererPort = await getFreePort()
  const rendererUrl = `http://127.0.0.1:${rendererPort}`

  frontendProcess = spawn(
    npmCommand,
    ['run', 'dev', '--', '--host', '127.0.0.1', '--port', String(rendererPort)],
    {
      cwd: frontendDir,
      env: {
        ...baseEnv,
        VITE_BACKEND_PORT: String(backendPort),
      },
      stdio: 'inherit',
    },
  )

  frontendProcess.once('exit', (code) => {
    if (!shuttingDown) {
      shutdown(code ?? 1)
    }
  })

  await waitForUrl(rendererUrl)

  electronProcess = spawn(electronBinary, ['.'], {
    cwd: rootDir,
    env: {
      ...baseEnv,
      GRPC_EXPLORER_DEV: '1',
      GRPC_EXPLORER_PORT: String(backendPort),
      GRPC_EXPLORER_RENDERER_URL: rendererUrl,
    },
    stdio: 'inherit',
  })

  electronProcess.once('exit', (code) => {
    shutdown(code ?? 0)
  })
}

process.on('SIGINT', () => shutdown(130))
process.on('SIGTERM', () => shutdown(143))

main().catch((err) => {
  console.error(err)
  shutdown(1)
})
