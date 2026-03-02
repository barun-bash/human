import React, { useEffect, useState, useCallback } from 'react'
import { X } from 'lucide-react'

export type ToastType = 'info' | 'success' | 'error' | 'warning'

interface ToastItem {
  id: string
  type: ToastType
  message: string
}

const typeStyles: Record<ToastType, React.CSSProperties> = {
  info: { background: 'var(--accent)', color: '#fff' },
  success: { background: 'var(--success)', color: '#fff' },
  error: { background: 'var(--error)', color: '#fff' },
  warning: { background: 'var(--warning)', color: '#fff' },
}

// Global toast state
let toastListeners: ((toasts: ToastItem[]) => void)[] = []
let toasts: ToastItem[] = []
let toastId = 0

function notifyListeners() {
  toastListeners.forEach((fn) => fn([...toasts]))
}

export function showToast(type: ToastType, message: string, duration = 3000) {
  const id = `toast-${++toastId}`
  toasts = [...toasts, { id, type, message }]
  notifyListeners()
  setTimeout(() => {
    toasts = toasts.filter((t) => t.id !== id)
    notifyListeners()
  }, duration)
}

export function ToastContainer() {
  const [items, setItems] = useState<ToastItem[]>([])

  useEffect(() => {
    toastListeners.push(setItems)
    return () => {
      toastListeners = toastListeners.filter((fn) => fn !== setItems)
    }
  }, [])

  const dismiss = useCallback((id: string) => {
    toasts = toasts.filter((t) => t.id !== id)
    notifyListeners()
  }, [])

  if (items.length === 0) return null

  return (
    <div
      style={{
        position: 'fixed',
        bottom: 80,
        left: '50%',
        transform: 'translateX(-50%)',
        zIndex: 50,
        display: 'flex',
        flexDirection: 'column',
        gap: 8,
      }}
    >
      {items.map((toast) => (
        <div
          key={toast.id}
          style={{
            display: 'flex',
            alignItems: 'center',
            gap: 12,
            padding: '12px 16px',
            borderRadius: 'var(--radius-sm)',
            boxShadow: '0 8px 32px rgba(0,0,0,0.3)',
            minWidth: 300,
            maxWidth: 500,
            ...typeStyles[toast.type],
          }}
        >
          <span style={{ flex: 1, fontSize: 13, fontWeight: 500 }}>{toast.message}</span>
          <button
            onClick={() => dismiss(toast.id)}
            style={{
              background: 'none',
              border: 'none',
              color: 'inherit',
              opacity: 0.7,
              cursor: 'pointer',
              padding: 0,
              display: 'flex',
            }}
          >
            <X size={14} />
          </button>
        </div>
      ))}
    </div>
  )
}
