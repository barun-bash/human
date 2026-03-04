import React, { useEffect, useRef, useState } from 'react'
import { ExternalLink } from 'lucide-react'
import { EditorView, keymap, lineNumbers, highlightActiveLine, highlightActiveLineGutter, drawSelection } from '@codemirror/view'
import { EditorState, Compartment } from '@codemirror/state'
import { defaultKeymap, history, historyKeymap, indentWithTab } from '@codemirror/commands'
import { bracketMatching, indentOnInput, foldGutter } from '@codemirror/language'
import { searchKeymap, highlightSelectionMatches, openSearchPanel } from '@codemirror/search'
import { closeBrackets, closeBracketsKeymap } from '@codemirror/autocomplete'
import { javascript } from '@codemirror/lang-javascript'
import { json } from '@codemirror/lang-json'
import { css } from '@codemirror/lang-css'
import { yaml } from '@codemirror/lang-yaml'
import { sql } from '@codemirror/lang-sql'
import { markdown } from '@codemirror/lang-markdown'
import { humanLanguage } from '../lib/human-lang'
import { humanEditorTheme, humanSyntaxHighlighting } from '../lib/editor-theme'
import { useProjectStore } from '../stores/project'
import { useEditorStore, EditorTab } from '../stores/editor'
import { api } from '../lib/ipc'

interface HumanEditorProps {
  onPopOut: () => void
}

const TABS: { id: EditorTab; label: string }[] = [
  { id: 'editor', label: 'Editor' },
  { id: 'ir', label: 'IR Preview' },
  { id: 'changes', label: 'Changes' },
]

const languageCompartment = new Compartment()

function getLanguageExtension(filename: string) {
  if (filename.endsWith('.human')) return humanLanguage
  if (filename.endsWith('.tsx') || filename.endsWith('.ts') || filename.endsWith('.jsx') || filename.endsWith('.js'))
    return javascript({ typescript: filename.endsWith('.ts') || filename.endsWith('.tsx'), jsx: filename.endsWith('.tsx') || filename.endsWith('.jsx') })
  if (filename.endsWith('.json')) return json()
  if (filename.endsWith('.css') || filename.endsWith('.scss')) return css()
  if (filename.endsWith('.yml') || filename.endsWith('.yaml')) return yaml()
  if (filename.endsWith('.sql')) return sql()
  if (filename.endsWith('.md')) return markdown()
  return []
}

export function HumanEditor({ onPopOut }: HumanEditorProps) {
  const editorContainerRef = useRef<HTMLDivElement>(null)
  const editorViewRef = useRef<EditorView | null>(null)
  const currentFileRef = useRef<string | null>(null)

  const { activeFile, openFiles, unsavedFiles, closeFile, setActiveFile } = useProjectStore()
  const { activeTab, setActiveTab, cursorLine, cursorCol, irContent, fileContents, savedContents, setFileContent, setCursor } = useEditorStore()

  // Create/update CodeMirror editor
  useEffect(() => {
    if (!editorContainerRef.current || activeTab !== 'editor') return

    // If we already have an editor for this file, skip
    if (editorViewRef.current && currentFileRef.current === activeFile) return

    // Destroy previous editor
    if (editorViewRef.current) {
      editorViewRef.current.destroy()
      editorViewRef.current = null
    }

    if (!activeFile) return

    // Wait for content to load before creating editor
    const content = fileContents[activeFile]
    if (content === undefined) return

    const filename = activeFile.split('/').pop() || activeFile.split('\\').pop() || ''

    const view = new EditorView({
      state: EditorState.create({
        doc: content,
        extensions: [
          lineNumbers(),
          highlightActiveLineGutter(),
          highlightActiveLine(),
          history(),
          foldGutter(),
          drawSelection(),
          indentOnInput(),
          bracketMatching(),
          closeBrackets(),
          highlightSelectionMatches(),
          keymap.of([
            ...defaultKeymap,
            ...historyKeymap,
            ...searchKeymap,
            ...closeBracketsKeymap,
            indentWithTab,
          ]),
          languageCompartment.of(getLanguageExtension(filename)),
          humanEditorTheme,
          humanSyntaxHighlighting,
          EditorState.tabSize.of(2),
          EditorView.updateListener.of((update) => {
            if (update.docChanged) {
              const newContent = update.state.doc.toString()
              setFileContent(activeFile, newContent)
              useProjectStore.getState().markUnsaved(activeFile)
            }
            // Update cursor position
            const pos = update.state.selection.main.head
            const line = update.state.doc.lineAt(pos)
            setCursor(line.number, pos - line.from + 1)
          }),
        ],
      }),
      parent: editorContainerRef.current,
    })

    editorViewRef.current = view
    currentFileRef.current = activeFile

    return () => {
      // Don't destroy here — we handle it on re-render
    }
  }, [activeFile, activeTab, activeFile ? fileContents[activeFile] !== undefined : false])

  // Sync content when file changes externally
  useEffect(() => {
    if (!editorViewRef.current || !activeFile) return
    const currentDoc = editorViewRef.current.state.doc.toString()
    const storeContent = fileContents[activeFile]
    if (storeContent !== undefined && storeContent !== currentDoc) {
      editorViewRef.current.dispatch({
        changes: { from: 0, to: currentDoc.length, insert: storeContent },
      })
    }
  }, [activeFile]) // Only re-sync when file switches

  // Cleanup on unmount
  useEffect(() => {
    return () => {
      if (editorViewRef.current) {
        editorViewRef.current.destroy()
        editorViewRef.current = null
      }
    }
  }, [])

  // Wire menu:find and menu:replace to CodeMirror search
  useEffect(() => {
    if (!api) return
    const cleanups = [
      api.on('menu:find', () => {
        if (editorViewRef.current) {
          openSearchPanel(editorViewRef.current)
        }
      }),
      api.on('menu:replace', () => {
        if (editorViewRef.current) {
          // openSearchPanel with replace mode — CM6 toggles replace via the panel
          openSearchPanel(editorViewRef.current)
        }
      }),
    ]
    return () => cleanups.forEach((fn) => fn())
  }, [])

  return (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
      {/* Tab bar for open files */}
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          borderBottom: '1px solid var(--border)',
          background: 'var(--bg-raised)',
        }}
      >
        <div style={{ flex: 1, display: 'flex', alignItems: 'center', overflowX: 'auto' }}>
          {openFiles.map((filePath) => {
            const name = filePath.split('/').pop() || filePath.split('\\').pop() || filePath
            const isActive = filePath === activeFile
            const isUnsaved = unsavedFiles.has(filePath)

            return (
              <FileTabButton
                key={filePath}
                name={name}
                isActive={isActive}
                isUnsaved={isUnsaved}
                isHuman={name.endsWith('.human')}
                onClick={() => setActiveFile(filePath)}
                onClose={() => closeFile(filePath)}
              />
            )
          })}
        </div>

        {/* Editor/IR/Changes tabs */}
        <div style={{ display: 'flex', alignItems: 'center', gap: 1, padding: '0 8px', flexShrink: 0 }}>
          {TABS.map((tab) => (
            <EditorTabButton
              key={tab.id}
              label={tab.label}
              isActive={activeTab === tab.id}
              onClick={() => setActiveTab(tab.id)}
            />
          ))}
        </div>

        <button
          onClick={onPopOut}
          title="Pop out"
          style={{
            padding: 6,
            color: 'var(--text-dim)',
            background: 'transparent',
            border: 'none',
            borderRadius: 'var(--radius-sm)',
            cursor: 'pointer',
            display: 'flex',
            marginRight: 8,
          }}
          onMouseEnter={(e) => { e.currentTarget.style.color = 'var(--text)' }}
          onMouseLeave={(e) => { e.currentTarget.style.color = 'var(--text-dim)' }}
        >
          <ExternalLink size={12} />
        </button>
      </div>

      {/* Editor content */}
      <div style={{ flex: 1, overflow: 'hidden' }}>
        {activeTab === 'editor' && (
          <div style={{ height: '100%', width: '100%' }}>
            {activeFile ? (
              <div ref={editorContainerRef} style={{ height: '100%', width: '100%' }} />
            ) : (
              <div
                style={{
                  height: '100%',
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  fontSize: 12,
                  color: 'var(--text-muted)',
                }}
              >
                Open a file to start editing
              </div>
            )}
          </div>
        )}

        {activeTab === 'ir' && (
          <div style={{ height: '100%', overflow: 'auto', padding: 16 }}>
            {irContent ? (
              <pre
                style={{
                  fontSize: 12,
                  color: 'var(--syn-type)',
                  fontFamily: 'var(--font-mono)',
                  margin: 0,
                }}
              >
                {irContent}
              </pre>
            ) : (
              <div
                style={{
                  height: '100%',
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  fontSize: 12,
                  color: 'var(--text-muted)',
                }}
              >
                Check or build your project to see the IR
              </div>
            )}
          </div>
        )}

        {activeTab === 'changes' && (
          <div style={{ height: '100%', overflow: 'auto' }}>
            {activeFile && fileContents[activeFile] !== undefined && savedContents[activeFile] !== undefined && fileContents[activeFile] !== savedContents[activeFile] ? (
              <DiffView
                original={savedContents[activeFile]}
                modified={fileContents[activeFile]}
              />
            ) : (
              <div
                style={{
                  height: '100%',
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  fontSize: 12,
                  color: 'var(--text-muted)',
                }}
              >
                No changes yet
              </div>
            )}
          </div>
        )}
      </div>

      {/* Status bar */}
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'flex-end',
          padding: '2px 12px',
          borderTop: '1px solid var(--border)',
          fontSize: 10,
          color: 'var(--text-dim)',
        }}
      >
        {activeFile && (
          <span>
            Ln {cursorLine}, Col {cursorCol}
          </span>
        )}
      </div>
    </div>
  )
}

function getFileIcon(name: string): { label: string; color: string } {
  const ext = name.split('.').pop()?.toLowerCase() || ''
  const map: Record<string, string> = {
    human: '#E85D3A', tsx: '#60A5FA', ts: '#60A5FA',
    sql: '#2D8C5A', json: '#FBBF24', css: '#A78BFA',
    md: '#7A7A7A', yml: '#22D3EE', yaml: '#22D3EE',
    prisma: '#2D8C5A', html: '#E85D3A', js: '#FBBF24',
    jsx: '#60A5FA', scss: '#A78BFA', svg: '#FBBF24',
    txt: '#7A7A7A', csv: '#2D8C5A', pdf: '#C43030',
  }
  return { label: ext.slice(0, 3).toUpperCase(), color: map[ext] || '#7A7A7A' }
}

function FileTabButton({
  name,
  isActive,
  isUnsaved,
  isHuman,
  onClick,
  onClose,
}: {
  name: string
  isActive: boolean
  isUnsaved: boolean
  isHuman: boolean
  onClick: () => void
  onClose: () => void
}) {
  const [hovered, setHovered] = useState(false)
  const icon = getFileIcon(name)

  return (
    <button
      onClick={onClick}
      onMouseEnter={() => setHovered(true)}
      onMouseLeave={() => setHovered(false)}
      style={{
        display: 'flex',
        alignItems: 'center',
        gap: 6,
        padding: '6px 12px',
        fontSize: 12,
        flexShrink: 0,
        background: isActive ? 'var(--bg)' : hovered ? 'var(--bg-hover)' : 'transparent',
        color: isActive ? 'var(--text-bright)' : 'var(--text-muted)',
        borderTop: 'none',
        borderLeft: 'none',
        borderRight: '1px solid var(--border)',
        borderBottom: isActive ? '2px solid var(--accent)' : '2px solid transparent',
        cursor: 'pointer',
        fontFamily: 'var(--font-body)',
      }}
    >
      <span
        style={{
          display: 'inline-flex',
          alignItems: 'center',
          justifyContent: 'center',
          width: 16,
          height: 16,
          borderRadius: 3,
          fontSize: 7,
          fontWeight: 700,
          letterSpacing: '-0.02em',
          background: icon.color + '22',
          color: icon.color,
          flexShrink: 0,
        }}
      >
        {icon.label}
      </span>
      <span style={{ color: isHuman ? 'var(--accent)' : undefined }}>
        {name}
      </span>
      {isUnsaved && (
        <span
          style={{
            width: 6,
            height: 6,
            borderRadius: '50%',
            background: 'var(--text-muted)',
          }}
        />
      )}
      <button
        onClick={(e) => {
          e.stopPropagation()
          onClose()
        }}
        style={{
          marginLeft: 4,
          color: 'var(--text-dim)',
          fontSize: 12,
          background: 'none',
          border: 'none',
          cursor: 'pointer',
          padding: 0,
          lineHeight: 1,
        }}
      >
        &times;
      </button>
    </button>
  )
}

function DiffView({ original, modified }: { original: string; modified: string }) {
  const origLines = original.split('\n')
  const modLines = modified.split('\n')

  // Simple line-by-line diff using LCS-like approach
  const diff: { type: 'same' | 'add' | 'remove'; text: string }[] = []
  let oi = 0, mi = 0
  while (oi < origLines.length || mi < modLines.length) {
    if (oi < origLines.length && mi < modLines.length && origLines[oi] === modLines[mi]) {
      diff.push({ type: 'same', text: origLines[oi] })
      oi++; mi++
    } else if (mi < modLines.length && (oi >= origLines.length || modLines.indexOf(origLines[oi], mi) > mi)) {
      diff.push({ type: 'add', text: modLines[mi] })
      mi++
    } else if (oi < origLines.length) {
      diff.push({ type: 'remove', text: origLines[oi] })
      oi++
    }
  }

  return (
    <pre style={{ margin: 0, padding: 12, fontSize: 12, lineHeight: 1.6, fontFamily: 'var(--font-mono)' }}>
      {diff.map((line, i) => {
        const color = line.type === 'add' ? '#34D399' : line.type === 'remove' ? '#F87171' : 'var(--text-muted)'
        const bg = line.type === 'add' ? 'rgba(52,211,153,0.1)' : line.type === 'remove' ? 'rgba(248,113,113,0.1)' : 'transparent'
        const prefix = line.type === 'add' ? '+ ' : line.type === 'remove' ? '- ' : '  '
        return (
          <div key={i} style={{ color, background: bg, padding: '0 4px' }}>
            {prefix}{line.text}
          </div>
        )
      })}
    </pre>
  )
}

function EditorTabButton({
  label,
  isActive,
  onClick,
}: {
  label: string
  isActive: boolean
  onClick: () => void
}) {
  const [hovered, setHovered] = useState(false)

  return (
    <button
      onClick={onClick}
      onMouseEnter={() => setHovered(true)}
      onMouseLeave={() => setHovered(false)}
      style={{
        padding: '4px 8px',
        fontSize: 10,
        borderRadius: 'var(--radius-sm)',
        background: isActive ? 'var(--bg-surface)' : 'transparent',
        color: isActive ? 'var(--text-bright)' : hovered ? 'var(--text-muted)' : 'var(--text-dim)',
        border: 'none',
        cursor: 'pointer',
        fontFamily: 'var(--font-body)',
      }}
    >
      {label}
    </button>
  )
}
