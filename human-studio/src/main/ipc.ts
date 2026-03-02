import { BrowserWindow, ipcMain, dialog, shell } from 'electron'
import { CompilerService } from './services/compiler'
import { ProjectService } from './services/project'
import { GitService } from './services/git'
import { LLMService } from './services/llm'
import { DockerService } from './services/docker'
import { AuthService } from './services/auth'
import { createPopOutWindow } from './window'

export function registerIpcHandlers(mainWindow: BrowserWindow) {
  const compiler = new CompilerService()
  const project = new ProjectService()
  const git = new GitService()
  const llm = new LLMService()
  const docker = new DockerService()
  const authService = new AuthService()
  authService.setMainWindow(mainWindow)

  // ── Project ──
  ipcMain.handle('project:open-dialog', async () => {
    const result = await dialog.showOpenDialog(mainWindow, {
      properties: ['openDirectory'],
      title: 'Open Project Folder',
    })
    if (result.canceled) return null
    return result.filePaths[0]
  })

  ipcMain.handle('project:open', async (_e, dirPath: string) => {
    return project.openProject(dirPath)
  })

  ipcMain.handle('project:create', async (_e, name: string, parentDir: string) => {
    return project.createProject(name, parentDir)
  })

  ipcMain.handle('project:read-file', async (_e, filePath: string) => {
    return project.readFile(filePath)
  })

  ipcMain.handle('project:write-file', async (_e, filePath: string, content: string) => {
    return project.writeFile(filePath, content)
  })

  ipcMain.handle('project:list-files', async (_e, dirPath: string) => {
    return project.listFiles(dirPath)
  })

  ipcMain.handle('project:create-file', async (_e, filePath: string, content?: string) => {
    return project.createFile(filePath, content)
  })

  ipcMain.handle('project:create-dir', async (_e, dirPath: string) => {
    return project.createDir(dirPath)
  })

  ipcMain.handle('project:delete', async (_e, targetPath: string) => {
    return project.deletePath(targetPath)
  })

  ipcMain.handle('project:rename', async (_e, oldPath: string, newPath: string) => {
    return project.rename(oldPath, newPath)
  })

  ipcMain.handle('project:watch', async (_e, dirPath: string) => {
    return project.watch(dirPath, (event, filePath) => {
      if (!mainWindow.isDestroyed()) {
        mainWindow.webContents.send('project:file-changed', event, filePath)
      }
    })
  })

  ipcMain.handle('project:unwatch', async () => {
    return project.unwatch()
  })

  // ── Compiler ──
  ipcMain.handle('compiler:check', async (_e, projectDir: string) => {
    return compiler.check(projectDir, (data) => {
      if (!mainWindow.isDestroyed()) {
        mainWindow.webContents.send('compiler:output', data)
      }
    })
  })

  ipcMain.handle('compiler:build', async (_e, projectDir: string) => {
    return compiler.build(projectDir, (data) => {
      if (!mainWindow.isDestroyed()) {
        mainWindow.webContents.send('compiler:output', data)
      }
    })
  })

  ipcMain.handle('compiler:run', async (_e, projectDir: string) => {
    return compiler.run(projectDir, (data) => {
      if (!mainWindow.isDestroyed()) {
        mainWindow.webContents.send('compiler:output', data)
      }
    })
  })

  ipcMain.handle('compiler:deploy', async (_e, projectDir: string) => {
    return compiler.deploy(projectDir, (data) => {
      if (!mainWindow.isDestroyed()) {
        mainWindow.webContents.send('compiler:output', data)
      }
    })
  })

  ipcMain.handle('compiler:stop', async () => {
    return compiler.stop()
  })

  ipcMain.handle('compiler:is-running', async () => {
    return compiler.isRunning()
  })

  // ── Git ──
  ipcMain.handle('git:status', async (_e, projectDir: string) => {
    return git.status(projectDir)
  })

  ipcMain.handle('git:branch', async (_e, projectDir: string) => {
    return git.currentBranch(projectDir)
  })

  ipcMain.handle('git:push', async (_e, projectDir: string) => {
    return git.push(projectDir)
  })

  ipcMain.handle('git:pull', async (_e, projectDir: string) => {
    return git.pull(projectDir)
  })

  ipcMain.handle('git:create-branch', async (_e, projectDir: string, name: string) => {
    return git.createBranch(projectDir, name)
  })

  // ── LLM ──
  ipcMain.handle(
    'llm:send',
    async (_e, provider: string, apiKey: string, messages: any[], context: any) => {
      return llm.send(provider, apiKey, messages, context)
    }
  )

  ipcMain.handle(
    'llm:stream',
    async (_e, provider: string, apiKey: string, messages: any[], context: any) => {
      return llm.stream(provider, apiKey, messages, context, (chunk) => {
        if (!mainWindow.isDestroyed()) {
          mainWindow.webContents.send('llm:chunk', chunk)
        }
      })
    }
  )

  // ── Docker ──
  ipcMain.handle('docker:available', async () => {
    return docker.isAvailable()
  })

  // ── Window ──
  ipcMain.handle('window:pop-out', async (_e, panelId: string) => {
    createPopOutWindow(panelId, mainWindow)
  })

  // ── Shell ──
  ipcMain.handle('shell:open-external', async (_e, url: string) => {
    return shell.openExternal(url)
  })

  ipcMain.handle('shell:open-path', async (_e, path: string) => {
    return shell.openPath(path)
  })

  // ── Auth ──
  ipcMain.handle('auth:get-stored', async () => {
    return authService.getStoredAuth()
  })

  ipcMain.handle('auth:oauth', async (_e, provider: string) => {
    return authService.startOAuth(provider)
  })

  ipcMain.handle('auth:validate', async () => {
    return authService.validateSession()
  })

  ipcMain.handle('auth:refresh', async () => {
    return authService.refreshTokens()
  })

  ipcMain.handle('auth:get-profile', async () => {
    return authService.getProfile()
  })

  ipcMain.handle('auth:get-subscription', async () => {
    return authService.getSubscription()
  })

  ipcMain.handle('auth:select-plan', async (_e, plan: string) => {
    return authService.selectPlan(plan)
  })

  ipcMain.handle('auth:logout', async () => {
    return authService.logout()
  })
}
