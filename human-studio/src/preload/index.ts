import { contextBridge, ipcRenderer } from 'electron'

const api = {
  // ── Project ──
  project: {
    openDialog: () => ipcRenderer.invoke('project:open-dialog'),
    open: (dir: string) => ipcRenderer.invoke('project:open', dir),
    create: (name: string, parentDir: string) =>
      ipcRenderer.invoke('project:create', name, parentDir),
    readFile: (path: string) => ipcRenderer.invoke('project:read-file', path),
    writeFile: (path: string, content: string) =>
      ipcRenderer.invoke('project:write-file', path, content),
    listFiles: (dir: string) => ipcRenderer.invoke('project:list-files', dir),
    createFile: (path: string, content?: string) =>
      ipcRenderer.invoke('project:create-file', path, content),
    createDir: (dir: string) => ipcRenderer.invoke('project:create-dir', dir),
    delete: (path: string) => ipcRenderer.invoke('project:delete', path),
    rename: (oldPath: string, newPath: string) =>
      ipcRenderer.invoke('project:rename', oldPath, newPath),
    watch: (dir: string) => ipcRenderer.invoke('project:watch', dir),
    unwatch: () => ipcRenderer.invoke('project:unwatch'),
  },

  // ── Compiler ──
  compiler: {
    check: (dir: string) => ipcRenderer.invoke('compiler:check', dir),
    build: (dir: string) => ipcRenderer.invoke('compiler:build', dir),
    run: (dir: string) => ipcRenderer.invoke('compiler:run', dir),
    deploy: (dir: string) => ipcRenderer.invoke('compiler:deploy', dir),
    stop: () => ipcRenderer.invoke('compiler:stop'),
    isRunning: () => ipcRenderer.invoke('compiler:is-running'),
  },

  // ── Git ──
  git: {
    status: (dir: string) => ipcRenderer.invoke('git:status', dir),
    branch: (dir: string) => ipcRenderer.invoke('git:branch', dir),
    push: (dir: string) => ipcRenderer.invoke('git:push', dir),
    pull: (dir: string) => ipcRenderer.invoke('git:pull', dir),
    createBranch: (dir: string, name: string) =>
      ipcRenderer.invoke('git:create-branch', dir, name),
  },

  // ── LLM ──
  llm: {
    send: (provider: string, apiKey: string, messages: any[], context: any) =>
      ipcRenderer.invoke('llm:send', provider, apiKey, messages, context),
    stream: (provider: string, apiKey: string, messages: any[], context: any) =>
      ipcRenderer.invoke('llm:stream', provider, apiKey, messages, context),
  },

  // ── Docker ──
  docker: {
    available: () => ipcRenderer.invoke('docker:available'),
  },

  // ── Window ──
  window: {
    popOut: (panelId: string) => ipcRenderer.invoke('window:pop-out', panelId),
  },

  // ── Shell ──
  shell: {
    openExternal: (url: string) => ipcRenderer.invoke('shell:open-external', url),
    openPath: (path: string) => ipcRenderer.invoke('shell:open-path', path),
  },

  // ── Auth ──
  auth: {
    getStored: () => ipcRenderer.invoke('auth:get-stored'),
    oauth: (provider: string) => ipcRenderer.invoke('auth:oauth', provider),
    validate: () => ipcRenderer.invoke('auth:validate'),
    refresh: () => ipcRenderer.invoke('auth:refresh'),
    getProfile: () => ipcRenderer.invoke('auth:get-profile'),
    getSubscription: () => ipcRenderer.invoke('auth:get-subscription'),
    selectPlan: (plan: string) => ipcRenderer.invoke('auth:select-plan', plan),
    logout: () => ipcRenderer.invoke('auth:logout'),
  },

  // ── Event Listeners ──
  on: (channel: string, callback: (...args: any[]) => void) => {
    const validChannels = [
      'compiler:output',
      'llm:chunk',
      'project:file-changed',
      'popout:closed',
      'theme:system-changed',
      'auth:session-expired',
      'menu:new-project',
      'menu:open-project',
      'menu:link-folder',
      'menu:save',
      'menu:save-all',
      'menu:settings',
      'menu:find',
      'menu:replace',
      'menu:toggle-sidebar',
      'menu:toggle-build-panel',
      'menu:toggle-theme',
      'menu:focus-panel',
      'menu:check',
      'menu:build',
      'menu:run',
      'menu:stop',
      'menu:deploy',
      'menu:clean',
      'menu:keyboard-shortcuts',
      'menu:check-updates',
      'menu:about',
    ]
    if (validChannels.includes(channel)) {
      const listener = (_event: any, ...args: any[]) => callback(...args)
      ipcRenderer.on(channel, listener)
      return () => ipcRenderer.removeListener(channel, listener)
    }
    return () => {}
  },

  removeAllListeners: (channel: string) => {
    ipcRenderer.removeAllListeners(channel)
  },
}

contextBridge.exposeInMainWorld('electronAPI', api)

export type ElectronAPI = typeof api
