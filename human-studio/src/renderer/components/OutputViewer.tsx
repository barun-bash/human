import React from 'react'
import { ExternalLink, ArrowLeft } from 'lucide-react'
import { useBuildStore } from '../stores/build'
import { FileTree } from './FileTree'
import { Badge } from './ui/Badge'
import { api } from '../lib/ipc'

const STACK_COLORS: Record<string, string> = {
  react: '#60A5FA',
  vue: '#42B883',
  angular: '#DD0031',
  svelte: '#FF3E00',
  node: '#34D399',
  python: '#FBBF24',
  go: '#00ADD8',
  postgres: '#22D3EE',
  database: '#22D3EE',
  docker: '#A78BFA',
  terraform: '#7B42BC',
  cicd: '#F97316',
  storybook: '#FF4785',
  monitoring: '#E85D3A',
}

interface OutputViewerProps {
  onPopOut: () => void
}

export function OutputViewer({ onPopOut }: OutputViewerProps) {
  const {
    outputFiles,
    selectedOutputFile,
    selectedOutputContent,
    fileCounts,
    setSelectedOutputFile,
    toggleOutputFolder,
  } = useBuildStore()

  const totalFiles = Object.values(fileCounts).reduce((a, b) => a + b, 0)

  return (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
      {/* Header */}
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          padding: '8px 12px',
          borderBottom: '1px solid var(--border)',
        }}
      >
        <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
          <span
            style={{
              fontSize: 10,
              fontWeight: 600,
              letterSpacing: '0.05em',
              color: 'var(--text-muted)',
              textTransform: 'uppercase',
            }}
          >
            Generated Output
          </span>
          {totalFiles > 0 && (
            <Badge variant="default">{totalFiles} files</Badge>
          )}
        </div>
        <IconBtn onClick={onPopOut} title="Pop out">
          <ExternalLink size={12} />
        </IconBtn>
      </div>

      {/* Content */}
      <div style={{ flex: 1, overflowY: 'auto' }}>
        {selectedOutputFile && selectedOutputContent !== null ? (
          <div style={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
            {/* File preview header */}
            <div
              style={{
                display: 'flex',
                alignItems: 'center',
                gap: 8,
                padding: '6px 12px',
                borderBottom: '1px solid var(--border)',
              }}
            >
              <IconBtn onClick={() => setSelectedOutputFile(null)}>
                <ArrowLeft size={12} />
              </IconBtn>
              <span style={{ fontSize: 12, color: 'var(--text)' }}>
                {selectedOutputFile.split('/').pop()}
              </span>
            </div>
            {/* Code preview */}
            <pre
              style={{
                flex: 1,
                overflow: 'auto',
                padding: 12,
                fontSize: 12,
                lineHeight: 1.6,
                fontFamily: 'var(--font-mono)',
                margin: 0,
                color: 'var(--text)',
              }}
            >
              {selectedOutputContent}
            </pre>
          </div>
        ) : outputFiles.length > 0 ? (
          <div style={{ padding: '4px 0' }}>
            <FileTree
              files={outputFiles.map((f) => ({
                ...f,
                isDirectory: f.isDirectory,
                children: f.children?.map((c) => ({ ...c, isDirectory: c.isDirectory })),
              }))}
              activeFile={null}
              onSelect={async (path) => {
                // Don't try to read directories
                if (isDirectory(outputFiles, path)) return
                setSelectedOutputFile(path, '// Loading...')
                try {
                  const content = await api.project.readFile(path)
                  setSelectedOutputFile(path, content)
                } catch {
                  setSelectedOutputFile(path, '// Error: Could not read file')
                }
              }}
              onToggle={(path) => toggleOutputFolder(path)}
            />
          </div>
        ) : (
          <div
            style={{
              display: 'flex',
              flexDirection: 'column',
              alignItems: 'center',
              justifyContent: 'center',
              height: '100%',
              gap: 8,
              textAlign: 'center',
              padding: '0 16px',
            }}
          >
            <p style={{ fontSize: 12, color: 'var(--text-muted)', marginBottom: 4 }}>
              Run a build to see generated code
            </p>
            <p style={{ fontSize: 11, color: 'var(--text-dim)' }}>
              {navigator.platform.includes('Mac') ? '\u2318\u21E7' : 'Ctrl+Shift+'}B to build
            </p>
          </div>
        )}
      </div>

      {/* Footer: stack badges */}
      {Object.keys(fileCounts).length > 0 && (
        <div
          style={{
            display: 'flex',
            flexWrap: 'wrap',
            gap: 6,
            padding: '6px 12px',
            borderTop: '1px solid var(--border)',
          }}
        >
          {Object.entries(fileCounts).map(([stack, count]) => {
            const color = STACK_COLORS[stack.toLowerCase()] || 'var(--text-muted)'
            return (
              <span
                key={stack}
                style={{
                  display: 'inline-flex',
                  alignItems: 'center',
                  gap: 4,
                  padding: '2px 8px',
                  fontSize: 10,
                  fontWeight: 600,
                  borderRadius: 'var(--radius-sm)',
                  background: color + '22',
                  color,
                }}
              >
                {stack} {count}
              </span>
            )
          })}
        </div>
      )}
    </div>
  )
}

function isDirectory(files: { path: string; isDirectory: boolean; children?: any[] }[], targetPath: string): boolean {
  for (const f of files) {
    if (f.path === targetPath && f.isDirectory) return true
    if (f.children && isDirectory(f.children, targetPath)) return true
  }
  return false
}

function IconBtn({ children, onClick, title }: { children: React.ReactNode; onClick?: () => void; title?: string }) {
  return (
    <button
      onClick={onClick}
      title={title}
      style={{
        padding: 4,
        color: 'var(--text-dim)',
        background: 'transparent',
        border: 'none',
        borderRadius: 'var(--radius-sm)',
        cursor: 'pointer',
        display: 'flex',
        alignItems: 'center',
      }}
      onMouseEnter={(e) => { e.currentTarget.style.color = 'var(--text)' }}
      onMouseLeave={(e) => { e.currentTarget.style.color = 'var(--text-dim)' }}
    >
      {children}
    </button>
  )
}
