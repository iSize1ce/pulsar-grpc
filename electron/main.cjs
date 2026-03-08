const fs = require('node:fs')
const http = require('node:http')
const path = require('node:path')
const { spawn } = require('node:child_process')
const { app, BrowserWindow, Menu, dialog, shell } = require('electron')

const productName = 'Pulsar GRPC'
const rootDir = path.resolve(__dirname, '..')
const isDev = process.env.GRPC_EXPLORER_DEV === '1'
const backendPort = process.env.GRPC_EXPLORER_PORT || '22333'
const backendUrl = `http://127.0.0.1:${backendPort}`
const rendererUrl = process.env.GRPC_EXPLORER_RENDERER_URL || null
const rendererOrigin = rendererUrl ? new URL(rendererUrl).origin : null
const userDataDir = path.join(app.getPath('appData'), productName)
const backendDataDir = path.join(userDataDir, 'data')
const goCacheDir = isDev
  ? path.join(rootDir, '.cache', 'go-build')
  : path.join(userDataDir, 'cache', 'go-build')

let backendProcess = null
let forceQuit = false
let quitting = false
const windows = new Set()

app.setName(productName)
app.setPath('userData', userDataDir)
app.setAppLogsPath(path.join(userDataDir, 'logs'))
app.setAboutPanelOptions({
  applicationName: productName,
  applicationVersion: app.getVersion(),
  version: app.getVersion(),
})

function getBackendCommand() {
  if (isDev) {
    return {
      command: process.platform === 'win32' ? 'go.exe' : 'go',
      args: ['run', '.'],
      cwd: rootDir,
    }
  }

  const binaryName = process.platform === 'win32'
    ? 'grpc-explorer-backend.exe'
    : 'grpc-explorer-backend'
  const binaryPath = app.isPackaged
    ? path.join(process.resourcesPath, 'backend', binaryName)
    : path.join(rootDir, 'dist', binaryName)

  if (!fs.existsSync(binaryPath)) {
    throw new Error(`Backend binary not found: ${binaryPath}`)
  }

  return {
    command: binaryPath,
    args: [],
    cwd: app.isPackaged ? process.resourcesPath : rootDir,
  }
}

async function waitForBackend(timeoutMs = 30000) {
  const startedAt = Date.now()

  while (Date.now() - startedAt < timeoutMs) {
    const ready = await new Promise((resolve) => {
      const req = http.get(`${backendUrl}/api/servers`, (res) => {
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

  throw new Error(`Timed out waiting for backend at ${backendUrl}`)
}

async function startBackend() {
  const backend = getBackendCommand()
  fs.mkdirSync(backendDataDir, { recursive: true })

  const backendEnv = {
    ...process.env,
    GRPC_EXPLORER_DATA_DIR: backendDataDir,
    GRPC_EXPLORER_NO_BROWSER: '1',
    GRPC_EXPLORER_PORT: String(backendPort),
  }

  if (isDev) {
    fs.mkdirSync(goCacheDir, { recursive: true })
    backendEnv.GOCACHE = goCacheDir
  }

  backendProcess = spawn(backend.command, backend.args, {
    cwd: backend.cwd,
    env: backendEnv,
    stdio: 'inherit',
  })

  backendProcess.once('exit', (code, signal) => {
    backendProcess = null

    if (quitting) {
      return
    }

    dialog.showErrorBox(
      'Backend stopped',
      `The bundled backend exited unexpectedly (${signal || code || 'unknown'}).`,
    )
    app.quit()
  })

  await waitForBackend()
}

function isAlive(child) {
  return Boolean(child) && child.exitCode === null && child.signalCode === null
}

function stopBackend(timeoutMs = 5000) {
  if (!isAlive(backendProcess)) {
    backendProcess = null
    return Promise.resolve()
  }

  const child = backendProcess

  return new Promise((resolve) => {
    let settled = false

    const finish = () => {
      if (settled) {
        return
      }

      settled = true
      clearTimeout(forceKillTimer)
      child.off('exit', onExit)
      if (backendProcess === child) {
        backendProcess = null
      }
      resolve()
    }

    const onExit = () => {
      finish()
    }

    const forceKillTimer = setTimeout(() => {
      if (!isAlive(child)) {
        finish()
        return
      }

      child.kill('SIGKILL')
      setTimeout(finish, 500)
    }, timeoutMs)

    child.on('exit', onExit)

    if (!child.kill('SIGTERM')) {
      finish()
    }
  })
}

function isExternalUrl(url) {
  return /^https?:\/\//.test(url) || url.startsWith('mailto:')
}

function isAllowedNavigation(url) {
  try {
    if (rendererOrigin) return new URL(url).origin === rendererOrigin
    return url.startsWith('file://')
  } catch {
    return false
  }
}

function handleNavigation(event, url) {
  if (isAllowedNavigation(url)) {
    return
  }

  event.preventDefault()

  if (isExternalUrl(url)) {
    shell.openExternal(url).catch((err) => {
      console.error('Failed to open external URL:', err)
    })
  }
}

function configureWindow(window) {
  window.webContents.setWindowOpenHandler(({ url }) => {
    if (isExternalUrl(url)) {
      shell.openExternal(url).catch((err) => {
        console.error('Failed to open external URL:', err)
      })
    }
    return { action: 'deny' }
  })

  window.webContents.on('will-navigate', (event, url) => {
    handleNavigation(event, url)
  })
}

function buildMenu() {
  const template = [
    ...(process.platform === 'darwin'
      ? [{
          label: productName,
          submenu: [
            { role: 'about' },
            { type: 'separator' },
            { role: 'services' },
            { type: 'separator' },
            { role: 'hide' },
            { role: 'hideOthers' },
            { role: 'unhide' },
            { type: 'separator' },
            { role: 'quit' },
          ],
        }]
      : []),
    ...(process.platform !== 'darwin'
      ? [{
          label: 'File',
          submenu: [
            { role: 'quit' },
          ],
        }]
      : []),
    {
      label: 'Edit',
      submenu: [
        { role: 'undo' },
        { role: 'redo' },
        { type: 'separator' },
        { role: 'cut' },
        { role: 'copy' },
        { role: 'paste' },
        { role: 'selectAll' },
      ],
    },
    {
      label: 'View',
      submenu: [
        { role: 'reload' },
        { role: 'toggleDevTools' },
        { type: 'separator' },
        { role: 'resetZoom' },
        { role: 'zoomIn' },
        { role: 'zoomOut' },
        { type: 'separator' },
        { role: 'togglefullscreen' },
        { type: 'separator' },
        {
          label: 'Clear Storage and Reload',
          accelerator: 'CmdOrCtrl+Shift+R',
          click: async () => {
            const win = BrowserWindow.getFocusedWindow()
            if (!win) return
            await win.webContents.session.clearStorageData({ storages: ['localstorage'] })
            win.webContents.reload()
          },
        },
        {
          label: 'Delete Database and Restart',
          click: async () => {
            const { response } = await dialog.showMessageBox({
              type: 'warning',
              buttons: ['Delete and Restart', 'Cancel'],
              defaultId: 1,
              message: 'Delete the backend database?',
              detail: 'This will delete servers.db and restart the backend.',
            })
            if (response !== 0) return
            await stopBackend()
            try { fs.rmSync(path.join(backendDataDir, 'servers.db'), { force: true }) } catch {}
            await startBackend()
            for (const win of windows) win.webContents.reload()
          },
        },
      ],
    },
    {
      label: 'Window',
      submenu: [
        { role: 'minimize' },
        { role: 'zoom' },
        ...(process.platform === 'darwin' ? [{ type: 'separator' }, { role: 'front' }] : []),
      ],
    },
  ]

  Menu.setApplicationMenu(Menu.buildFromTemplate(template))
}

async function createWindow() {
  const window = new BrowserWindow({
    width: 1440,
    height: 960,
    minWidth: 1100,
    minHeight: 720,
    title: productName,
    backgroundColor: '#111827',
    autoHideMenuBar: process.platform !== 'darwin',
    show: false,
    titleBarStyle: process.platform === 'darwin' ? 'hiddenInset' : 'default',
    webPreferences: {
      contextIsolation: true,
      nodeIntegration: false,
      devTools: true,
      webSecurity: false,
    },
  })

  windows.add(window)
  configureWindow(window)

  const readyToShow = new Promise((resolve) => {
    window.once('ready-to-show', resolve)
  })

  const loadPromise = rendererUrl
    ? window.loadURL(rendererUrl)
    : window.loadFile(
        app.isPackaged
          ? path.join(process.resourcesPath, 'static', 'index.html')
          : path.join(rootDir, 'frontend', 'dist', 'index.html'),
        { query: { backendPort } }
      )
  await Promise.all([loadPromise, readyToShow])
  if (!window.isDestroyed()) {
    window.show()
  }

  window.on('closed', () => {
    windows.delete(window)
  })
}

app.on('before-quit', (event) => {
  if (forceQuit || quitting) {
    return
  }

  event.preventDefault()
  quitting = true

  stopBackend().finally(() => {
    forceQuit = true
    app.quit()
  })
})

app.on('window-all-closed', () => {
  app.quit()
})

app.on('activate', () => {
  if (windows.size === 0) {
    createWindow().catch((err) => {
      dialog.showErrorBox('Window error', err.message)
      app.quit()
    })
  }
})

app.whenReady().then(async () => {
  try {
    buildMenu()
    await startBackend()
    await createWindow()
  } catch (err) {
    await stopBackend()
    dialog.showErrorBox('Startup failed', err instanceof Error ? err.message : String(err))
    app.quit()
  }
})
