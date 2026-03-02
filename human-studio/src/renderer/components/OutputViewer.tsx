import React from 'react'
import { ExternalLink, ArrowLeft } from 'lucide-react'
import { useBuildStore } from '../stores/build'
import { FileTree } from './FileTree'
import { Badge } from './ui/Badge'

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
                setSelectedOutputFile(path, '// Loading...')
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
            <p style={{ fontSize: 12, color: 'var(--text-muted)' }}>
              Run a build to see generated code
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
          {Object.entries(fileCounts).map(([stack, count]) => (
            <Badge key={stack} variant="default">
              {stack} {count}
            </Badge>
          ))}
        </div>
      )}
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
