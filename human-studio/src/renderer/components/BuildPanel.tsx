import React, { useEffect, useRef } from 'react'
import { ChevronDown, ChevronUp } from 'lucide-react'
import { useBuildStore } from '../stores/build'
import { useSettingsStore } from '../stores/settings'
import { Badge } from './ui/Badge'
import { api } from '../lib/ipc'

const URL_REGEX = /https?:\/\/localhost:\d+[^\s]*/g

function getLineColor(line: string): string {
  if (/✓|complete|passed|ready|success/i.test(line)) return '#34D399'
  if (/error|Error|failed|FAIL/i.test(line)) return '#F87171'
  if (/warning|Warning/i.test(line)) return '#FBBF24'
  if (/^(\s*\.{3}|.*Building|.*Starting|.*Generating)/i.test(line)) return 'var(--text-dim)'
  return 'var(--text-muted)'
}

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
              {output.split('\n').map((line, i) => {
                const urls = line.match(URL_REGEX)
                if (urls) {
                  // Render line with clickable URLs
                  const parts: React.ReactNode[] = []
                  let rest = line
                  urls.forEach((url, j) => {
                    const idx = rest.indexOf(url)
                    if (idx > 0) parts.push(rest.slice(0, idx))
                    parts.push(
                      <a
                        key={j}
                        onClick={(e) => { e.preventDefault(); api?.shell.openExternal(url) }}
                        style={{ color: '#60A5FA', textDecoration: 'underline', cursor: 'pointer' }}
                      >
                        {url}
                      </a>
                    )
                    rest = rest.slice(idx + url.length)
                  })
                  if (rest) parts.push(rest)
                  return <div key={i} style={{ color: getLineColor(line) }}>{parts}</div>
                }
                return <div key={i} style={{ color: getLineColor(line) }}>{line}</div>
              })}
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
