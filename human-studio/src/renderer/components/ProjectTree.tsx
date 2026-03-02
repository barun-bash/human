import React from 'react'
import { FolderPlus, Link, ExternalLink } from 'lucide-react'
import { useProjectStore } from '../stores/project'
import { useBuildStore } from '../stores/build'
import { FileTree } from './FileTree'
import { Badge } from './ui/Badge'
import { Button } from './ui/Button'
import { api } from '../lib/ipc'

interface ProjectTreeProps {
  onPopOut: () => void
}

const statusVariant: Record<string, 'default' | 'accent' | 'success' | 'error' | 'warning' | 'info'> = {
  idle: 'default',
  checking: 'info',
  building: 'accent',
  running: 'success',
  deploying: 'accent',
  success: 'success',
  error: 'error',
}

const statusLabel: Record<string, string> = {
  idle: 'Idle',
  checking: 'Checking...',
  building: 'Building...',
  running: 'Running',
  deploying: 'Deploying...',
  success: 'Build OK',
  error: 'Error',
}

export function ProjectTree({ onPopOut }: ProjectTreeProps) {
  const { projectDir, files, activeFile, openFile, toggleFolder } = useProjectStore()
  const buildStatus = useBuildStore((s) => s.status)

  const handleLinkFolder = async () => {
    const dir = await api.project.openDialog()
    if (dir) {
      const projectFiles = await api.project.open(dir)
      const name = dir.split('/').pop() || dir.split('\\').pop() || 'project'
      useProjectStore.getState().setProject(dir, name)
      useProjectStore.getState().setFiles(projectFiles)
      api.project.watch(dir)
    }
  }

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
        <span
          style={{
            fontSize: 10,
            fontWeight: 600,
            letterSpacing: '0.05em',
            color: 'var(--text-muted)',
            textTransform: 'uppercase',
          }}
        >
          Project
        </span>
        <div style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
          <IconBtn onClick={handleLinkFolder} title="Link folder">
            <Link size={12} />
          </IconBtn>
          <IconBtn title="New project">
            <FolderPlus size={12} />
          </IconBtn>
          <IconBtn onClick={onPopOut} title="Pop out">
            <ExternalLink size={12} />
          </IconBtn>
        </div>
      </div>

      {/* Tree */}
      <div style={{ flex: 1, overflowY: 'auto', padding: '4px 0' }}>
        {projectDir ? (
          <FileTree
            files={files}
            activeFile={activeFile}
            onSelect={openFile}
            onToggle={toggleFolder}
          />
        ) : (
          <div
            style={{
              display: 'flex',
              flexDirection: 'column',
              alignItems: 'center',
              justifyContent: 'center',
              height: '100%',
              gap: 12,
              padding: '0 16px',
              textAlign: 'center',
            }}
          >
            <p style={{ fontSize: 12, color: 'var(--text-muted)' }}>
              Open a project or create a new one
            </p>
            <Button variant="primary" size="sm" onClick={handleLinkFolder}>
              <FolderPlus size={13} />
              Open folder...
            </Button>
          </div>
        )}
      </div>

      {/* Footer: build status */}
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          padding: '6px 12px',
          borderTop: '1px solid var(--border)',
        }}
      >
        <Badge variant={statusVariant[buildStatus] || 'default'}>
          {statusLabel[buildStatus] || 'Idle'}
        </Badge>
      </div>
    </div>
  )
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
