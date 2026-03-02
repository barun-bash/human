import React from 'react'

interface AvatarProps {
  name?: string
  size?: number
}

export function Avatar({ name = '', size = 28 }: AvatarProps) {
  const initials = name
    .split(' ')
    .map((w) => w[0])
    .join('')
    .toUpperCase()
    .slice(0, 2) || 'U'

  return (
    <div
      style={{
        display: 'inline-flex',
        alignItems: 'center',
        justifyContent: 'center',
        borderRadius: '50%',
        background: 'var(--accent)',
        color: '#fff',
        fontWeight: 600,
        width: size,
        height: size,
        fontSize: size * 0.4,
        fontFamily: 'var(--font-heading)',
      }}
    >
      {initials}
    </div>
  )
}
