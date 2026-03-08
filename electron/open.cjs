const path = require('node:path')
const { spawn } = require('node:child_process')

const electronBinary = require('electron')

const rootDir = path.resolve(__dirname, '..')
const env = { ...process.env }

delete env.ELECTRON_RUN_AS_NODE

const child = spawn(electronBinary, ['.'], {
  cwd: rootDir,
  env,
  stdio: 'inherit',
})

child.once('exit', (code) => {
  process.exit(code ?? 0)
})
