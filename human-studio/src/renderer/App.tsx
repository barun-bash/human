import React, { useState, useCallback, useEffect } from 'react'
import { TopBar } from './components/TopBar'
import { ProjectTree } from './components/ProjectTree'
import { PromptChat } from './components/PromptChat'
import { HumanEditor } from './components/HumanEditor'
import { OutputViewer } from './components/OutputViewer'
import { BuildPanel } from './components/BuildPanel'
import { ProfilePanel } from './components/ProfilePanel'
import { AuthScreen } from './components/AuthScreen'
import { PlanSelector } from './components/PlanSelector'
import { ResizeHandle } from './components/ResizeHandle'
import { ToastContainer, showToast } from './components/ui/Toast'
import { useResize } from './hooks/useResize'
import { useTheme } from './hooks/useTheme'
import { useSettingsStore } from './stores/settings'
import { useProjectStore } from './stores/project'
import { useBuildStore } from './stores/build'
import { useEditorStore } from './stores/editor'
import { useAuthStore } from './stores/auth'
import { api } from './lib/ipc'
import type { OutputFile } from './stores/build'
import type { FileEntry } from './stores/project'

/** Scan output dir after build, populate output tree + stack badges + refresh project tree */
async function loadOutputFiles(projectDir: string) {
  try {
    const outputDir = projectDir + '/output'
    const entries: FileEntry[] = await api.project.listFiles(outputDir)

    // Convert FileEntry[] → OutputFile[] with auto-expand for top 2 levels
    function toOutputFiles(files: FileEntry[], depth: number): OutputFile[] {
      return files.map((f) => ({
        name: f.name,
        path: f.path,
        isDirectory: f.isDirectory,
        expanded: depth < 2,
        children: f.children ? toOutputFiles(f.children, depth + 1) : undefined,
      }))
    }
    const outputFiles = toOutputFiles(entries, 0)

    // Count files by top-level directory
    const counts: Record<string, number> = {}
    for (const entry of entries) {
      if (entry.isDirectory && entry.children) {
        counts[entry.name] = countFiles(entry.children)
      }
    }

    useBuildStore.getState().setOutputFiles(outputFiles)
    useBuildStore.getState().setFileCounts(counts)
  } catch {
    // output/ dir might not exist yet — ignore
  }

  // Refresh project tree
  try {
    const files = await api.project.open(projectDir)
    useProjectStore.getState().setFiles(files)
  } catch {
    // ignore
  }
}

function countFiles(entries: FileEntry[]): number {
  let count = 0
  for (const e of entries) {
    if (e.isDirectory && e.children) count += countFiles(e.children)
    else count++
  }
  return count
}

export function App() {
  const [profileOpen, setProfileOpen] = useState(false)
  const { screen, setScreen, setUser, setSubscription, logout } = useAuthStore()
  useTheme()

  // Auth: check stored tokens on startup
  useEffect(() => {
    if (!api) return
    ;(async () => {
      try {
        const stored = await api.auth.getStored()
        if (stored?.auth?.accessToken) {
          // Validate tokens
          const valid = await api.auth.validate()
          if (valid) {
            setUser(stored.auth.user)
            if (stored.subscription) {
              setSubscription(stored.subscription)
            } else {
              try {
                const sub = await api.auth.getSubscription()
                setSubscription(sub)
              } catch {
                // Non-fatal
              }
            }
            setScreen('app')
            return
          }
        }
      } catch {
        // No stored auth or validation failed
      }
      setScreen('auth')
    })()
  }, [])

  // Listen for session expiry from main process
  useEffect(() => {
    if (!api) return
    const cleanup = api.on('auth:session-expired', () => {
      logout()
      showToast('warning', 'Session expired. Please log in again.')
    })
    return cleanup
  }, [logout])

  const { columnWidths, setColumnWidth, sidebarVisible } = useSettingsStore()
  const projectDir = useProjectStore((s) => s.projectDir)

  // Resize hooks for each column border
  const projectResize = useResize({
    min: 120,
    max: 400,
    initial: columnWidths.project,
    onResize: (w) => setColumnWidth('project', w),
  })

  const promptResize = useResize({
    min: 200,
    max: 600,
    initial: columnWidths.prompt,
    onResize: (w) => setColumnWidth('prompt', w),
  })

  const outputResize = useResize({
    min: 200,
    max: 600,
    initial: columnWidths.output,
    onResize: (w) => setColumnWidth('output', w),
  })

  // Build actions
  const handleCheck = useCallback(async () => {
    if (!projectDir) {
      showToast('error', 'Open a project first')
      return
    }
    const { status: buildStatus } = useBuildStore.getState()
    if (buildStatus === 'checking' || buildStatus === 'building' || buildStatus === 'running') {
      showToast('warning', 'A process is already running')
      return
    }
    useBuildStore.getState().clearOutput()
    useBuildStore.getState().setStatus('checking')
    try {
      const result = await api.compiler.check(projectDir)
      useBuildStore.getState().setStatus(result.code === 0 ? 'success' : 'error')
      if (result.code === 0) {
        showToast('success', 'Check passed')
        useEditorStore.getState().setIRContent(result.stdout || useBuildStore.getState().output)
      } else {
        showToast('error', 'Check failed')
      }
    } catch (err: any) {
      useBuildStore.getState().setStatus('error')
      showToast('error', err.message || 'Check failed')
    }
  }, [projectDir])

  const handleBuild = useCallback(async () => {
    if (!projectDir) {
      showToast('error', 'Open a project first')
      return
    }
    const { status: buildStatus } = useBuildStore.getState()
    if (buildStatus === 'checking' || buildStatus === 'building' || buildStatus === 'running') {
      showToast('warning', 'A process is already running')
      return
    }
    useBuildStore.getState().clearOutput()
    useBuildStore.getState().setStatus('building')
    useSettingsStore.getState().setBuildPanelOpen(true)
    try {
      const result = await api.compiler.build(projectDir)
      useBuildStore.getState().setStatus(result.code === 0 ? 'success' : 'error')
      if (result.code === 0) {
        showToast('success', 'Build complete')
        useEditorStore.getState().setIRContent(result.stdout || useBuildStore.getState().output)
        await loadOutputFiles(projectDir)
      } else {
        showToast('error', 'Build failed')
      }
    } catch (err: any) {
      useBuildStore.getState().setStatus('error')
      showToast('error', err.message || 'Build failed')
    }
  }, [projectDir])

  const handleRun = useCallback(async () => {
    if (!projectDir) {
      showToast('error', 'Open a project first')
      return
    }
    useBuildStore.getState().clearOutput()
    useBuildStore.getState().setStatus('running')
    useSettingsStore.getState().setBuildPanelOpen(true)
    try {
      const result = await api.compiler.run(projectDir)
      useBuildStore.getState().setStatus(result.code === 0 ? 'success' : 'error')
      if (result.code === 0) {
        await loadOutputFiles(projectDir)
      }
    } catch (err: any) {
      useBuildStore.getState().setStatus('error')
      showToast('error', err.message || 'Run failed')
    }
  }, [projectDir])

  const handleStop = useCallback(async () => {
    try {
      await api.compiler.stop()
      useBuildStore.getState().setStatus('idle')
      showToast('info', 'Process stopped')
    } catch {
      // ignore
    }
  }, [])

  const handlePopOut = useCallback((panel: string) => {
    api?.window.popOut(panel)
  }, [])

  // Listen for compiler output from main process
  useEffect(() => {
    if (!api) return
    const cleanup = api.on('compiler:output', (data: string) => {
      useBuildStore.getState().appendOutput(data)
    })
    return cleanup
  }, [])

  // Listen for menu events
  useEffect(() => {
    if (!api) return
    const cleanups = [
      api.on('menu:check', handleCheck),
      api.on('menu:build', handleBuild),
      api.on('menu:run', handleRun),
      api.on('menu:stop', handleStop),
      api.on('menu:toggle-build-panel', () => useSettingsStore.getState().toggleBuildPanel()),
      api.on('menu:toggle-sidebar', () => useSettingsStore.getState().toggleSidebar()),
      api.on('menu:toggle-theme', () => useSettingsStore.getState().toggleTheme()),
      api.on('menu:settings', () => setProfileOpen(true)),
      api.on('menu:save', async () => {
        const { activeFile } = useProjectStore.getState()
        const { fileContents } = useEditorStore.getState()
        if (activeFile && fileContents[activeFile] !== undefined) {
          await api.project.writeFile(activeFile, fileContents[activeFile])
          useProjectStore.getState().markSaved(activeFile)
          useEditorStore.getState().setSavedContent(activeFile, fileContents[activeFile])
          showToast('success', 'File saved')
        }
      }),
      api.on('menu:open-project', async () => {
        const dir = await api.project.openDialog()
        if (dir) {
          const files = await api.project.open(dir)
          const name = dir.split('/').pop() || 'project'
          useProjectStore.getState().setProject(dir, name)
          useProjectStore.getState().setFiles(files)
          useSettingsStore.getState().addRecentProject(dir)
          api.project.watch(dir)
        }
      }),
      api.on('menu:link-folder', async () => {
        const dir = await api.project.openDialog()
        if (dir) {
          const files = await api.project.open(dir)
          const name = dir.split('/').pop() || 'project'
          useProjectStore.getState().setProject(dir, name)
          useProjectStore.getState().setFiles(files)
          api.project.watch(dir)
        }
      }),
    ]
    return () => cleanups.forEach((fn) => fn())
  }, [handleCheck, handleBuild, handleRun, handleStop])

  // Load file content when active file changes
  const activeFile = useProjectStore((s) => s.activeFile)
  useEffect(() => {
    if (!activeFile || !api) return
    const { fileContents } = useEditorStore.getState()
    if (fileContents[activeFile] !== undefined) return // already loaded

    api.project.readFile(activeFile).then((content: string) => {
      useEditorStore.getState().setSavedContent(activeFile, content)
    }).catch(() => {
      showToast('error', `Failed to read ${activeFile.split('/').pop()}`)
    })
  }, [activeFile])

  // Auth gating
  if (screen === 'loading') {
    return (
      <div style={{ height: '100%', display: 'flex', alignItems: 'center', justifyContent: 'center', background: 'var(--bg)' }}>
        <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 16 }}>
          <svg width="48" height="48" viewBox="0 0 120 120">
            <rect width="120" height="120" rx="24" fill="#0D0D0D" />
            <text x="24" y="84" fontFamily="Nunito, sans-serif" fontWeight="700" fontSize="72" letterSpacing="-1">
              <tspan fill="#F5F5F3">h</tspan>
              <tspan fill="#E85D3A">_</tspan>
            </text>
          </svg>
          <div style={{ fontSize: 12, color: 'var(--text-muted)' }}>Loading...</div>
        </div>
      </div>
    )
  }

  if (screen === 'auth') {
    return (
      <>
        <AuthScreen />
        <ToastContainer />
      </>
    )
  }

  if (screen === 'plan-select') {
    return (
      <>
        <PlanSelector />
        <ToastContainer />
      </>
    )
  }

  return (
    <div style={{ height: '100%', display: 'flex', flexDirection: 'column', background: 'var(--bg)' }}>
      {/* Top Bar */}
      <TopBar
        onCheck={handleCheck}
        onBuild={handleBuild}
        onRun={handleRun}
        onStop={handleStop}
        onOpenProfile={() => setProfileOpen(true)}
        onConfigureKeys={() => setProfileOpen(true)}
      />

      {/* Main content area */}
      <div style={{ flex: 1, display: 'flex', overflow: 'hidden' }}>
        {/* Column 1: Project Tree */}
        {sidebarVisible && (
          <>
            <div
              style={{
                width: columnWidths.project,
                flexShrink: 0,
                background: 'var(--bg-raised)',
                overflow: 'hidden',
              }}
            >
              <ProjectTree onPopOut={() => handlePopOut('project')} />
            </div>
            <ResizeHandle
              onMouseDown={projectResize.onMouseDown}
              handleRef={projectResize.handleRef}
            />
          </>
        )}

        {/* Column 2: Prompt Chat */}
        <div
          style={{
            width: columnWidths.prompt,
            flexShrink: 0,
            background: 'var(--bg-raised)',
            overflow: 'hidden',
          }}
        >
          <PromptChat onPopOut={() => handlePopOut('prompt')} />
        </div>
        <ResizeHandle
          onMouseDown={promptResize.onMouseDown}
          handleRef={promptResize.handleRef}
        />

        {/* Column 3: Editor (flex) */}
        <div
          style={{
            flex: 1,
            minWidth: 280,
            overflow: 'hidden',
            background: 'var(--bg)',
          }}
        >
          <HumanEditor onPopOut={() => handlePopOut('editor')} />
        </div>
        <ResizeHandle
          onMouseDown={outputResize.onMouseDown}
          handleRef={outputResize.handleRef}
        />

        {/* Column 4: Output */}
        <div
          style={{
            width: columnWidths.output,
            flexShrink: 0,
            background: 'var(--bg-raised)',
            overflow: 'hidden',
          }}
        >
          <OutputViewer onPopOut={() => handlePopOut('output')} />
        </div>
      </div>

      {/* Bottom: Build Panel */}
      <BuildPanel />

      {/* Profile slide-in */}
      <ProfilePanel open={profileOpen} onClose={() => setProfileOpen(false)} />

      {/* Toast notifications */}
      <ToastContainer />
    </div>
  )
}
