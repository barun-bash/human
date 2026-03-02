import { create } from 'zustand'

export type BuildStatus = 'idle' | 'checking' | 'building' | 'running' | 'success' | 'error'

export interface OutputFile {
  name: string
  path: string
  isDirectory: boolean
  children?: OutputFile[]
  expanded?: boolean
}

export interface BuildState {
  status: BuildStatus
  output: string
  outputFiles: OutputFile[]
  selectedOutputFile: string | null
  selectedOutputContent: string | null
  fileCounts: Record<string, number>

  setStatus: (status: BuildStatus) => void
  appendOutput: (text: string) => void
  clearOutput: () => void
  setOutputFiles: (files: OutputFile[]) => void
  setSelectedOutputFile: (path: string | null, content?: string | null) => void
  setFileCounts: (counts: Record<string, number>) => void
  toggleOutputFolder: (path: string) => void
}

export const useBuildStore = create<BuildState>((set) => ({
  status: 'idle',
  output: '',
  outputFiles: [],
  selectedOutputFile: null,
  selectedOutputContent: null,
  fileCounts: {},

  setStatus: (status) => set({ status }),
  appendOutput: (text) => set((s) => ({ output: s.output + text })),
  clearOutput: () => set({ output: '' }),
  setOutputFiles: (files) => set({ outputFiles: files }),
  setSelectedOutputFile: (path, content = null) =>
    set({ selectedOutputFile: path, selectedOutputContent: content }),
  setFileCounts: (counts) => set({ fileCounts: counts }),
  toggleOutputFolder: (path) =>
    set((s) => ({
      outputFiles: toggleExpanded(s.outputFiles, path),
    })),
}))

function toggleExpanded(files: OutputFile[], targetPath: string): OutputFile[] {
  return files.map((f) => {
    if (f.path === targetPath && f.isDirectory) {
      return { ...f, expanded: !f.expanded }
    }
    if (f.children) {
      return { ...f, children: toggleExpanded(f.children, targetPath) }
    }
    return f
  })
}
