import React, { useEffect, useRef } from 'react'
import { ChevronDown, ChevronUp } from 'lucide-react'
import { useBuildStore } from '../stores/build'
import { useSettingsStore } from '../stores/settings'
import { Badge } from './ui/Badge'

const statusVariant: Record<string, 'default' | 'accent' | 'success' | 'error' | 'info'> = {
  idle: 'default',
  checking: 'info',
  building: 'accent',
  running: 'success',
  success: 'success',
  error: 'error',
}

const statusLabel: Record<string, string> = {
  idle: 'Ready',
  checking: 'Running...',
  building: 'Running...',
  running: 'Running...',
  success: 'Passed',
  error: 'Failed',
}

export function BuildPanel() {
  const { buildPanelOpen, toggleBuildPanel } = useSettingsStore()
  const { status, output } = useBuildStore()
  const outputRef = useRef<HTMLPreElement>(null)

  useEffect(() => {
    if (outputRef.current) {
      outputRef.current.scrollTop = outputRef.current.scrollHeight
    }
  }, [output])

  return (
    <div style={{ borderTop: '1px solid var(--border)' }}>
      {/* Toggle header */}
      <button
        onClick={toggleBuildPanel}
        style={{
          width: '100%',
          display: 'flex',
          alignItems: 'center',
          gap: 8,
          padding: '6px 16px',
          background: 'transparent',
          border: 'none',
          cursor: 'pointer',
          color: 'var(--text-muted)',
          fontFamily: 'var(--font-body)',
        }}
        onMouseEnter={(e) => { e.currentTarget.style.background = 'var(--bg-hover)' }}
        onMouseLeave={(e) => { e.currentTarget.style.background = 'transparent' }}
      >
        {buildPanelOpen ? <ChevronDown size={12} /> : <ChevronUp size={12} />}
        <span style={{ fontSize: 12, fontWeight: 500, color: 'var(--text-muted)' }}>
          Build Output
        </span>
        <Badge variant={statusVariant[status] || 'default'}>
          {statusLabel[status] || 'Ready'}
        </Badge>
      </button>

      {/* Terminal output */}
      {buildPanelOpen && (
        <div style={{ height: 192, background: 'var(--bg)', overflow: 'hidden' }}>
          {output ? (
            <pre
              ref={outputRef}
              style={{
                height: '100%',
                overflow: 'auto',
                padding: 12,
                fontSize: 12,
                lineHeight: 1.6,
                color: 'var(--text-muted)',
                fontFamily: 'var(--font-mono)',
                margin: 0,
              }}
            >
              {output}
            </pre>
          ) : (
            <div
              style={{
                height: '100%',
                display: 'flex',
                flexDirection: 'column',
                alignItems: 'center',
                justifyContent: 'center',
                textAlign: 'center',
                padding: 24,
                color: 'var(--text-dim)',
                fontSize: 12,
                gap: 4,
              }}
            >
              <code style={{ fontFamily: 'var(--font-mono)', color: 'var(--text-muted)' }}>$ human build</code>
              <span>Press {navigator.platform.includes('Mac') ? '\u2318' : 'Ctrl+'}B to build your project</span>
            </div>
          )}
        </div>
      )}
    </div>
  )
}
