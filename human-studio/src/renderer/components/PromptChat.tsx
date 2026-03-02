import React, { useState, useRef, useEffect } from 'react'
import { Bot, Send, Paperclip, ExternalLink } from 'lucide-react'
import { useChatStore, createMessageId } from '../stores/chat'
import { Button } from './ui/Button'

interface PromptChatProps {
  onPopOut: () => void
}

const QUICK_CHIPS = ['Generate app', 'Add feature', 'Fix error', 'Explain code']

export function PromptChat({ onPopOut }: PromptChatProps) {
  const [input, setInput] = useState('')
  const messagesEndRef = useRef<HTMLDivElement>(null)
  const textareaRef = useRef<HTMLTextAreaElement>(null)
  const { messages, isLoading, pendingAttachments, addMessage, removeAttachment } = useChatStore()

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages])

  const handleSend = () => {
    const text = input.trim()
    if (!text && pendingAttachments.length === 0) return

    addMessage({
      id: createMessageId(),
      role: 'user',
      content: text,
      timestamp: Date.now(),
      attachments: pendingAttachments.length > 0 ? [...pendingAttachments] : undefined,
    })

    setInput('')
    useChatStore.getState().clearAttachments()
  }

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault()
      handleSend()
    }
  }

  const handleChipClick = (chip: string) => {
    setInput(chip + ' ')
    textareaRef.current?.focus()
  }

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
        <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
          <Bot size={13} style={{ color: 'var(--accent)' }} />
          <span
            style={{
              fontSize: 10,
              fontWeight: 600,
              letterSpacing: '0.05em',
              color: 'var(--text-muted)',
              textTransform: 'uppercase',
            }}
          >
            Prompt
          </span>
        </div>
        <IconBtn onClick={onPopOut} title="Pop out">
          <ExternalLink size={12} />
        </IconBtn>
      </div>

      {/* Messages */}
      <div style={{ flex: 1, overflowY: 'auto', padding: 12 }}>
        {messages.length === 0 ? (
          <div
            style={{
              display: 'flex',
              flexDirection: 'column',
              alignItems: 'center',
              justifyContent: 'center',
              height: '100%',
              gap: 12,
              textAlign: 'center',
            }}
          >
            <Bot size={32} style={{ color: 'var(--text-dim)' }} />
            <p style={{ fontSize: 12, color: 'var(--text-muted)' }}>
              Describe what you want to build...
            </p>
          </div>
        ) : (
          <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
            {messages.map((msg) => (
              <div key={msg.id} style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
                <span
                  style={{
                    fontSize: 10,
                    fontWeight: 600,
                    color: msg.role === 'user' ? 'var(--info)' : 'var(--accent)',
                  }}
                >
                  {msg.role === 'user' ? 'You' : 'Human AI'}
                </span>
                <div
                  style={{
                    padding: '8px 12px',
                    borderRadius: 'var(--radius-sm)',
                    fontSize: 12,
                    lineHeight: 1.6,
                    ...(msg.role === 'user'
                      ? {
                          background: 'var(--bg-surface)',
                          border: '1px solid var(--border)',
                          color: 'var(--text)',
                        }
                      : {
                          background: 'var(--accent-dim)',
                          border: '1px solid var(--accent-border)',
                          color: 'var(--text)',
                        }),
                  }}
                >
                  {msg.content}
                </div>
                {msg.attachments && msg.attachments.length > 0 && (
                  <div style={{ display: 'flex', flexWrap: 'wrap', gap: 4, marginTop: 4 }}>
                    {msg.attachments.map((a) => (
                      <span
                        key={a.name}
                        style={{
                          fontSize: 10,
                          color: 'var(--text-dim)',
                          background: 'var(--bg-surface)',
                          padding: '2px 6px',
                          borderRadius: 4,
                        }}
                      >
                        {a.name}
                      </span>
                    ))}
                  </div>
                )}
              </div>
            ))}
            {isLoading && (
              <div style={{ display: 'flex', alignItems: 'center', gap: 6, fontSize: 12, color: 'var(--text-muted)' }}>
                <span>Human AI is thinking...</span>
              </div>
            )}
            <div ref={messagesEndRef} />
          </div>
        )}
      </div>

      {/* Quick chips */}
      <div style={{ display: 'flex', flexWrap: 'wrap', gap: 6, padding: '6px 12px' }}>
        {QUICK_CHIPS.map((chip) => (
          <ChipButton key={chip} onClick={() => handleChipClick(chip)}>
            {chip}
          </ChipButton>
        ))}
      </div>

      {/* Attachments */}
      {pendingAttachments.length > 0 && (
        <div style={{ display: 'flex', flexWrap: 'wrap', gap: 6, padding: '4px 12px' }}>
          {pendingAttachments.map((a) => (
            <span
              key={a.name}
              style={{
                display: 'flex',
                alignItems: 'center',
                gap: 4,
                padding: '2px 8px',
                fontSize: 10,
                background: 'var(--bg-surface)',
                border: '1px solid var(--border)',
                borderRadius: 99,
                color: 'var(--text-muted)',
              }}
            >
              {a.name} ({(a.size / 1024 / 1024).toFixed(1)}MB)
              <button
                onClick={() => removeAttachment(a.name)}
                style={{
                  background: 'none',
                  border: 'none',
                  color: 'var(--text-dim)',
                  cursor: 'pointer',
                  padding: 0,
                  fontSize: 12,
                }}
              >
                &times;
              </button>
            </span>
          ))}
        </div>
      )}

      {/* Input */}
      <div
        style={{
          display: 'flex',
          alignItems: 'flex-end',
          gap: 8,
          padding: '8px 12px',
          borderTop: '1px solid var(--border)',
        }}
      >
        <IconBtn title="Attach file">
          <Paperclip size={14} />
        </IconBtn>
        <textarea
          ref={textareaRef}
          value={input}
          onChange={(e) => setInput(e.target.value)}
          onKeyDown={handleKeyDown}
          placeholder="Describe what you want to build..."
          rows={1}
          style={{
            flex: 1,
            resize: 'none',
            background: 'transparent',
            fontSize: 12,
            color: 'var(--text)',
            border: 'none',
            outline: 'none',
            maxHeight: 120,
            fontFamily: 'var(--font-body)',
            lineHeight: 1.5,
          }}
        />
        <Button
          variant="primary"
          size="sm"
          onClick={handleSend}
          disabled={!input.trim() && pendingAttachments.length === 0}
        >
          <Send size={12} />
        </Button>
      </div>
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
        flexShrink: 0,
      }}
      onMouseEnter={(e) => { e.currentTarget.style.color = 'var(--text)' }}
      onMouseLeave={(e) => { e.currentTarget.style.color = 'var(--text-dim)' }}
    >
      {children}
    </button>
  )
}

function ChipButton({ children, onClick }: { children: React.ReactNode; onClick: () => void }) {
  return (
    <button
      onClick={onClick}
      style={{
        padding: '2px 8px',
        fontSize: 10,
        color: 'var(--text-muted)',
        background: 'var(--bg-surface)',
        border: '1px solid var(--border)',
        borderRadius: 99,
        cursor: 'pointer',
        fontFamily: 'var(--font-body)',
      }}
      onMouseEnter={(e) => {
        e.currentTarget.style.borderColor = 'var(--border-hover)'
        e.currentTarget.style.color = 'var(--text)'
      }}
      onMouseLeave={(e) => {
        e.currentTarget.style.borderColor = 'var(--border)'
        e.currentTarget.style.color = 'var(--text-muted)'
      }}
    >
      {children}
    </button>
  )
}
