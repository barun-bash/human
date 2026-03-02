import React, { useState } from 'react'

interface InputProps extends React.InputHTMLAttributes<HTMLInputElement> {
  label?: string
}

export function Input({ label, style, ...props }: InputProps) {
  const [focused, setFocused] = useState(false)
  const [hovered, setHovered] = useState(false)

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
      {label && (
        <label style={{ fontSize: 12, fontWeight: 500, color: 'var(--text-muted)' }}>
          {label}
        </label>
      )}
      <input
        {...props}
        onFocus={(e) => {
          setFocused(true)
          props.onFocus?.(e)
        }}
        onBlur={(e) => {
          setFocused(false)
          props.onBlur?.(e)
        }}
        onMouseEnter={() => setHovered(true)}
        onMouseLeave={() => setHovered(false)}
        style={{
          width: '100%',
          padding: '8px 12px',
          fontSize: 13,
          background: 'var(--bg-surface)',
          color: 'var(--text)',
          border: `1px solid ${focused ? 'var(--accent)' : hovered ? 'var(--border-hover)' : 'var(--border)'}`,
          borderRadius: 'var(--radius-sm)',
          outline: 'none',
          boxShadow: focused ? '0 0 0 1px var(--accent)' : 'none',
          fontFamily: 'var(--font-body)',
          transition: 'border-color 150ms, box-shadow 150ms',
          ...style,
        }}
      />
    </div>
  )
}
