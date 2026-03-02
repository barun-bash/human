import React, { useState } from 'react'
import {
  ChevronRight,
  ChevronDown,
  File,
  FileCode,
  FileJson,
  FileType,
  Folder,
  FolderOpen,
  Database,
  Settings,
  Palette,
} from 'lucide-react'
import type { FileEntry } from '../stores/project'

interface FileTreeProps {
  files: FileEntry[]
  activeFile: string | null
  onSelect: (path: string) => void
  onToggle: (path: string) => void
  depth?: number
}

const EXT_ICONS: Record<string, { icon: React.ElementType; color: string }> = {
  '.human': { icon: FileCode, color: 'var(--accent)' },
  '.tsx': { icon: FileCode, color: '#3B82F6' },
  '.ts': { icon: FileCode, color: '#3B82F6' },
  '.jsx': { icon: FileCode, color: '#3B82F6' },
  '.js': { icon: FileCode, color: '#FCD34D' },
  '.sql': { icon: Database, color: 'var(--success)' },
  '.json': { icon: FileJson, color: '#FCD34D' },
  '.css': { icon: Palette, color: '#A78BFA' },
  '.scss': { icon: Palette, color: '#A78BFA' },
  '.yml': { icon: Settings, color: '#67E8F9' },
  '.yaml': { icon: Settings, color: '#67E8F9' },
  '.md': { icon: FileType, color: 'var(--text-muted)' },
}

function getFileIcon(name: string) {
  const ext = name.includes('.') ? '.' + name.split('.').pop() : ''
  return EXT_ICONS[ext] || { icon: File, color: 'var(--text-dim)' }
}

export function FileTree({ files, activeFile, onSelect, onToggle, depth = 0 }: FileTreeProps) {
  return (
    <div>
      {files.map((entry) => {
        const { icon: Icon, color } = entry.isDirectory
          ? { icon: entry.expanded ? FolderOpen : Folder, color: 'var(--text-muted)' }
          : getFileIcon(entry.name)

        const isActive = entry.path === activeFile

        return (
          <div key={entry.path}>
            <FileTreeItem
              entry={entry}
              Icon={Icon}
              color={color}
              isActive={isActive}
              depth={depth}
              onSelect={onSelect}
              onToggle={onToggle}
            />
            {entry.isDirectory && entry.expanded && entry.children && (
              <FileTree
                files={entry.children}
                activeFile={activeFile}
                onSelect={onSelect}
                onToggle={onToggle}
                depth={depth + 1}
              />
            )}
          </div>
        )
      })}
    </div>
  )
}

function FileTreeItem({
  entry,
  Icon,
  color,
  isActive,
  depth,
  onSelect,
  onToggle,
}: {
  entry: FileEntry
  Icon: React.ElementType
  color: string
  isActive: boolean
  depth: number
  onSelect: (path: string) => void
  onToggle: (path: string) => void
}) {
  const [hovered, setHovered] = useState(false)

  return (
    <button
      onClick={() =>
        entry.isDirectory ? onToggle(entry.path) : onSelect(entry.path)
      }
      onMouseEnter={() => setHovered(true)}
      onMouseLeave={() => setHovered(false)}
      style={{
        width: '100%',
        display: 'flex',
        alignItems: 'center',
        gap: 6,
        paddingLeft: depth * 16 + 8,
        paddingRight: 8,
        paddingTop: 2,
        paddingBottom: 2,
        textAlign: 'left',
        fontSize: 12,
        border: 'none',
        cursor: 'pointer',
        overflow: 'hidden',
        textOverflow: 'ellipsis',
        whiteSpace: 'nowrap',
        background: isActive
          ? 'var(--accent-dim)'
          : hovered
            ? 'var(--bg-hover)'
            : 'transparent',
        color: isActive ? 'var(--accent)' : 'var(--text)',
        fontFamily: 'var(--font-body)',
      }}
    >
      {entry.isDirectory && (
        <span style={{ color: 'var(--text-dim)', flexShrink: 0, display: 'flex' }}>
          {entry.expanded ? <ChevronDown size={12} /> : <ChevronRight size={12} />}
        </span>
      )}
      <Icon size={13} style={{ color, flexShrink: 0 }} />
      <span style={{ overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
        {entry.name}
      </span>
    </button>
  )
}
