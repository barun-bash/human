import { spawn, ChildProcess } from 'child_process'
import { app } from 'electron'
import { join } from 'path'
import { existsSync, statSync } from 'fs'

export class CompilerService {
  private process: ChildProcess | null = null

  /** Resolve the bundled `human` binary path. */
  private getBinaryPath(): string {
    const isDevMode = !app.isPackaged

    if (isDevMode) {
      // In development, look for the built binary in the repo root's build/ dir
      const repoRoot = join(__dirname, '../../../..')
      const devBinary = join(repoRoot, 'build', 'human')
      if (existsSync(devBinary) && statSync(devBinary).isFile()) return devBinary
      // Fallback to PATH
      return 'human'
    }

    // In production, the binary is in resources/bin/
    const platform = process.platform === 'win32' ? 'human.exe' : 'human'
    const bundledPath = join(process.resourcesPath, 'bin', platform)
    if (existsSync(bundledPath)) return bundledPath

    // Final fallback to PATH
    return 'human'
  }

  private exec(
    args: string[],
    projectDir: string,
    onData: (data: string) => void
  ): Promise<{ code: number; stdout: string; stderr: string }> {
    return new Promise((resolve, reject) => {
      if (this.process) {
        reject(new Error('A process is already running. Stop it first.'))
        return
      }

      const binary = this.getBinaryPath()
      let stdout = ''
      let stderr = ''

      this.process = spawn(binary, args, {
        cwd: projectDir,
        env: { ...process.env },
        stdio: ['ignore', 'pipe', 'pipe'],
      })

      this.process.stdout?.on('data', (data: Buffer) => {
        const text = data.toString()
        stdout += text
        onData(text)
      })

      this.process.stderr?.on('data', (data: Buffer) => {
        const text = data.toString()
        stderr += text
        onData(text)
      })

      this.process.on('error', (err) => {
        this.process = null
        reject(err)
      })

      this.process.on('close', (code) => {
        this.process = null
        resolve({ code: code ?? 1, stdout, stderr })
      })
    })
  }

  async check(projectDir: string, onData: (data: string) => void) {
    return this.exec(['check', '.'], projectDir, onData)
  }

  async build(projectDir: string, onData: (data: string) => void) {
    return this.exec(['build', '.'], projectDir, onData)
  }

  async run(projectDir: string, onData: (data: string) => void) {
    return this.exec(['run'], projectDir, onData)
  }

  async deploy(projectDir: string, onData: (data: string) => void) {
    return this.exec(['deploy', '--to', 'Docker'], projectDir, onData)
  }

  stop() {
    if (this.process) {
      this.process.kill('SIGTERM')
      // Force kill after 5s
      setTimeout(() => {
        if (this.process) {
          this.process.kill('SIGKILL')
          this.process = null
        }
      }, 5000)
    }
  }

  isRunning(): boolean {
    return this.process !== null
  }
}
