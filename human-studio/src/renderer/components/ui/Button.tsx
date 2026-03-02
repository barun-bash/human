import React from 'react'

interface ButtonProps extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: 'primary' | 'secondary' | 'ghost' | 'danger' | 'success' | 'info'
  size?: 'sm' | 'md' | 'lg'
  children: React.ReactNode
}

const variantStyles: Record<string, React.CSSProperties> = {
  primary: { background: 'var(--accent)', color: '#fff' },
  secondary: { background: 'var(--bg-surface)', color: 'var(--text)', border: '1px solid var(--border)' },
  ghost: { background: 'transparent', color: 'var(--text-muted)' },
  danger: { background: 'var(--error)', color: '#fff' },
  success: { background: 'var(--success)', color: '#fff' },
  info: { background: 'var(--info)', color: '#fff' },
}

const sizeStyles: Record<string, React.CSSProperties> = {
  sm: { padding: '4px 10px', fontSize: '12px', gap: '5px' },
  md: { padding: '6px 14px', fontSize: '13px', gap: '6px' },
  lg: { padding: '8px 18px', fontSize: '13px', gap: '8px' },
}

export function Button({
  variant = 'secondary',
  size = 'md',
  style,
  children,
  disabled,
  ...props
}: ButtonProps) {
  return (
    <button
      style={{
        display: 'inline-flex',
        alignItems: 'center',
        justifyContent: 'center',
        fontWeight: 600,
        borderRadius: 'var(--radius-sm)',
        border: 'none',
        cursor: disabled ? 'default' : 'pointer',
        opacity: disabled ? 0.5 : 1,
        transition: 'opacity 150ms, filter 150ms',
        fontFamily: 'var(--font-body)',
        lineHeight: 1,
        whiteSpace: 'nowrap',
        ...variantStyles[variant],
        ...sizeStyles[size],
        ...style,
      }}
      disabled={disabled}
      onMouseEnter={(e) => {
        if (!disabled) (e.currentTarget.style.filter = 'brightness(1.1)')
      }}
      onMouseLeave={(e) => {
        (e.currentTarget.style.filter = '')
      }}
      {...props}
    >
      {children}
    </button>
  )
}
