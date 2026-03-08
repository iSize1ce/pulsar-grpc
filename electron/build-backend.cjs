const fs = require('node:fs')
const path = require('node:path')
const { spawnSync } = require('node:child_process')

const rootDir = path.resolve(__dirname, '..')
const outputDir = path.join(rootDir, 'dist')
const goCacheDir = path.join(rootDir, '.cache', 'go-build')
const binaryName = process.platform === 'win32'
  ? 'pulsar-grpc-backend.exe'
  : 'pulsar-grpc-backend'

fs.mkdirSync(outputDir, { recursive: true })
fs.mkdirSync(goCacheDir, { recursive: true })

const result = spawnSync('go', ['build', '-o', path.join(outputDir, binaryName), '.'], {
  cwd: rootDir,
  env: {
    ...process.env,
    GOCACHE: goCacheDir,
  },
  stdio: 'inherit',
})

if (result.status !== 0) {
  process.exit(result.status ?? 1)
}
