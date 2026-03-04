import React, { useEffect } from 'react'
import { X } from 'lucide-react'

interface KeyboardShortcutsProps {
  open: boolean
  onClose: () => void
}

const isMac = typeof navigator !== 'undefined' && navigator.platform.includes('Mac')
const mod = isMac ? '⌘' : 'Ctrl+'
const shift = isMac ? '⇧' : 'Shift+'

const SECTIONS: { title: string; shortcuts: { keys: string; label: string }[] }[] = [
  {
    title: 'File',
    shortcuts: [
      { keys: `${mod}N`, label: 'New project' },
      { keys: `${mod}O`, label: 'Open project' },
      { keys: `${mod}S`, label: 'Save' },
      { keys: `${mod}${shift}S`, label: 'Save all' },
    ],
  },
  {
    title: 'Build',
    shortcuts: [
      { keys: `${mod}B`, label: 'Build' },
      { keys: `${mod}${shift}B`, label: 'Run' },
      { keys: `${mod}${shift}C`, label: 'Check' },
      { keys: `${mod}.`, label: 'Stop' },
    ],
  },
  {
    title: 'Editor',
    shortcuts: [
      { keys: `${mod}F`, label: 'Find' },
      { keys: `${mod}H`, label: 'Find & replace' },
      { keys: `${mod}Z`, label: 'Undo' },
      { keys: `${mod}${shift}Z`, label: 'Redo' },
    ],
  },
  {
    title: 'View',
    shortcuts: [
      { keys: `${mod}\\`, label: 'Toggle sidebar' },
      { keys: `${mod}J`, label: 'Toggle build panel' },
      { keys: `${mod}${shift}T`, label: 'Toggle theme' },
    ],
  },
]

export function KeyboardShortcuts({ open, onClose }: KeyboardShortcutsProps) {
  useEffect(() => {
    if (!open) return
    const handleKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose()
    }
    window.addEventListener('keydown', handleKey)
    return () => window.removeEventListener('keydown', handleKey)
  }, [open, onClose])

  if (!open) return null

  return (
    <div
      onClick={onClose}
      style={{
        position: 'fixed',
        inset: 0,
        zIndex: 1000,
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        background: 'rgba(0,0,0,0.5)',
        backdropFilter: 'blur(4px)',
      }}
    >
      <div
        onClick={(e) => e.stopPropagation()}
        style={{
          width: 420,
          maxHeight: '80vh',
          overflow: 'auto',
          background: 'var(--bg-raised)',
          borderRadius: 'var(--radius-lg)',
          border: '1px solid var(--border)',
          boxShadow: '0 20px 60px rgba(0,0,0,0.4)',
        }}
      >
        {/* Header */}
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            padding: '16px 20px 12px',
            borderBottom: '1px solid var(--border)',
          }}
        >
          <span style={{ fontSize: 14, fontWeight: 600, color: 'var(--text-bright)' }}>
            Keyboard Shortcuts
          </span>
          <button
            onClick={onClose}
            style={{
              padding: 4,
              color: 'var(--text-dim)',
              background: 'transparent',
              border: 'none',
              borderRadius: 'var(--radius-sm)',
              cursor: 'pointer',
              display: 'flex',
            }}
          >
            <X size={14} />
          </button>
        </div>

        {/* Sections */}
        <div style={{ padding: '12px 20px 20px' }}>
          {SECTIONS.map((section) => (
            <div key={section.title} style={{ marginBottom: 16 }}>
              <div
                style={{
                  fontSize: 10,
                  fontWeight: 600,
                  letterSpacing: '0.05em',
                  textTransform: 'uppercase',
                  color: 'var(--text-dim)',
                  marginBottom: 8,
                }}
              >
                {section.title}
              </div>
              {section.shortcuts.map((s) => (
                <div
                  key={s.keys}
                  style={{
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'space-between',
                    padding: '4px 0',
                  }}
                >
                  <span style={{ fontSize: 12, color: 'var(--text-muted)' }}>{s.label}</span>
                  <kbd
                    style={{
                      fontSize: 11,
                      fontFamily: 'var(--font-mono)',
                      padding: '2px 6px',
                      borderRadius: 4,
                      background: 'var(--bg-surface)',
                      color: 'var(--text)',
                      border: '1px solid var(--border)',
                    }}
                  >
                    {s.keys}
                  </kbd>
                </div>
              ))}
            </div>
          ))}
        </div>
      </div>
    </div>
  )
}
