import React, { useState, useRef, useEffect } from 'react'
import { ChevronDown } from 'lucide-react'

interface DropdownItem {
  label: string
  value: string
  icon?: React.ReactNode
  divider?: boolean
  disabled?: boolean
}

interface DropdownProps {
  items: DropdownItem[]
  value: string
  onChange: (value: string) => void
  trigger?: React.ReactNode
  align?: 'left' | 'right'
}

export function Dropdown({
  items,
  value,
  onChange,
  trigger,
  align = 'left',
}: DropdownProps) {
  const [open, setOpen] = useState(false)
  const ref = useRef<HTMLDivElement>(null)

  useEffect(() => {
    const handler = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) {
        setOpen(false)
      }
    }
    document.addEventListener('mousedown', handler)
    return () => document.removeEventListener('mousedown', handler)
  }, [])

  const selected = items.find((i) => i.value === value)

  return (
    <div ref={ref} style={{ position: 'relative' }}>
      <button
        onClick={() => setOpen(!open)}
        style={{
          display: 'flex',
          alignItems: 'center',
          gap: 6,
          padding: '4px 8px',
          fontSize: 12,
          color: 'var(--text-muted)',
          borderRadius: 'var(--radius-sm)',
          background: 'transparent',
          border: 'none',
          cursor: 'pointer',
          fontFamily: 'var(--font-body)',
        }}
        onMouseEnter={(e) => {
          e.currentTarget.style.background = 'var(--bg-hover)'
          e.currentTarget.style.color = 'var(--text)'
        }}
        onMouseLeave={(e) => {
          e.currentTarget.style.background = 'transparent'
          e.currentTarget.style.color = 'var(--text-muted)'
        }}
      >
        {trigger || (
          <>
            {selected?.icon}
            <span>{selected?.label || value}</span>
            <ChevronDown size={12} />
          </>
        )}
      </button>

      {open && (
        <div
          style={{
            position: 'absolute',
            top: '100%',
            marginTop: 4,
            zIndex: 50,
            minWidth: 180,
            background: 'var(--bg-raised)',
            border: '1px solid var(--border)',
            borderRadius: 'var(--radius-sm)',
            boxShadow: '0 8px 32px rgba(0,0,0,0.3)',
            padding: '4px 0',
            ...(align === 'right' ? { right: 0 } : { left: 0 }),
          }}
        >
          {items.map((item, i) =>
            item.divider ? (
              <div
                key={i}
                style={{
                  height: 1,
                  background: 'var(--border)',
                  margin: '4px 0',
                }}
              />
            ) : (
              <DropdownItemButton
                key={item.value}
                item={item}
                isSelected={item.value === value}
                onSelect={() => {
                  onChange(item.value)
                  setOpen(false)
                }}
              />
            )
          )}
        </div>
      )}
    </div>
  )
}

function DropdownItemButton({
  item,
  isSelected,
  onSelect,
}: {
  item: DropdownItem
  isSelected: boolean
  onSelect: () => void
}) {
  const [hovered, setHovered] = useState(false)

  return (
    <button
      disabled={item.disabled}
      onClick={onSelect}
      onMouseEnter={() => setHovered(true)}
      onMouseLeave={() => setHovered(false)}
      style={{
        width: '100%',
        display: 'flex',
        alignItems: 'center',
        gap: 8,
        padding: '6px 12px',
        fontSize: 12,
        textAlign: 'left',
        color: isSelected ? 'var(--accent)' : 'var(--text)',
        background: hovered && !item.disabled ? 'var(--bg-hover)' : 'transparent',
        border: 'none',
        cursor: item.disabled ? 'default' : 'pointer',
        opacity: item.disabled ? 0.4 : 1,
        fontFamily: 'var(--font-body)',
      }}
    >
      {item.icon}
      {item.label}
    </button>
  )
}
