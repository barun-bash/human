import React from 'react'

interface BadgeProps {
  variant?: 'default' | 'accent' | 'success' | 'error' | 'warning' | 'info'
  children: React.ReactNode
}

const variantStyles: Record<string, React.CSSProperties> = {
  default: { background: 'var(--bg-surface)', color: 'var(--text-muted)' },
  accent: { background: 'var(--accent-dim)', color: 'var(--accent)', border: '1px solid var(--accent-border)' },
  success: { background: 'rgba(45,140,90,0.15)', color: 'var(--success)' },
  error: { background: 'rgba(196,48,48,0.15)', color: 'var(--error)' },
  warning: { background: 'rgba(212,148,10,0.15)', color: 'var(--warning)' },
  info: { background: 'rgba(59,130,246,0.15)', color: 'var(--info)' },
}

export function Badge({ variant = 'default', children }: BadgeProps) {
  return (
    <span
      style={{
        display: 'inline-flex',
        alignItems: 'center',
        padding: '2px 7px',
        fontSize: '10px',
        fontWeight: 600,
        borderRadius: '99px',
        lineHeight: 1.2,
        ...variantStyles[variant],
      }}
    >
      {children}
    </span>
  )
}
