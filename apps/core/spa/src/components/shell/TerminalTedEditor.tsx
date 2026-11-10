import { useEffect, useRef } from 'react'
import { EditorState } from '@codemirror/state'
import { EditorView, keymap, lineNumbers, highlightActiveLine } from '@codemirror/view'
import { defaultKeymap, history, historyKeymap } from '@codemirror/commands'
import { yaml } from '@codemirror/lang-yaml'
import { vim, Vim } from '@replit/codemirror-vim'
import { Button } from '@/components/ui/button'
import { t } from '@/lib'

export type TedSession = {
  command: string
  seed: string
}

type Props = {
  session: TedSession
  onSave: (body: string) => void
  onCancel: () => void
}

export function TerminalTedEditor({ session, onSave, onCancel }: Props) {
  const hostRef = useRef<HTMLDivElement>(null)
  const viewRef = useRef<EditorView | null>(null)
  const saveRef = useRef(onSave)
  const cancelRef = useRef(onCancel)
  saveRef.current = onSave
  cancelRef.current = onCancel

  useEffect(() => {
    if (!hostRef.current) return

    const writeAndQuit = () => {
      const body = viewRef.current?.state.doc.toString() ?? ''
      saveRef.current(body)
      return true
    }
    const quit = () => {
      cancelRef.current()
      return true
    }

    Vim.defineEx('w', 'w', writeAndQuit)
    Vim.defineEx('wq', 'wq', writeAndQuit)
    Vim.defineEx('x', 'x', writeAndQuit)
    Vim.defineEx('q', 'q', quit)
    Vim.defineEx('quit', 'quit', quit)

    const state = EditorState.create({
      doc: session.seed,
      extensions: [
        vim(),
        lineNumbers(),
        highlightActiveLine(),
        history(),
        yaml(),
        keymap.of([...defaultKeymap, ...historyKeymap]),
        EditorView.theme({
          '&': {
            height: '100%',
            fontSize: '0.8125rem',
            backgroundColor: 'var(--kg-shell-bg)',
            color: 'var(--kg-shell-text)',
          },
          '.cm-content': { fontFamily: 'var(--kg-font-mono)', caretColor: 'var(--kg-primary)' },
          '.cm-gutters': {
            backgroundColor: 'var(--kg-shell-bg)',
            color: 'var(--kg-shell-text-muted)',
            border: 'none',
          },
          '.cm-activeLine': { backgroundColor: 'var(--kg-shell-hover)' },
          '.cm-activeLineGutter': { backgroundColor: 'var(--kg-shell-hover)' },
        }),
        EditorView.lineWrapping,
      ],
    })
    const view = new EditorView({ state, parent: hostRef.current })
    viewRef.current = view
    requestAnimationFrame(() => view.focus())
    return () => {
      view.destroy()
      viewRef.current = null
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [session.command, session.seed])

  return (
    <div className="kg-terminal-ted" role="dialog" aria-label={t('shell.terminal.ted_title')}>
      <header className="kg-terminal-ted-bar">
        <span className="kg-terminal-ted-title">{t('shell.terminal.ted_title')}</span>
        <span className="kg-terminal-ted-cmd" title={session.command}>
          {session.command}
        </span>
        <span className="kg-terminal-ted-hint">{t('shell.terminal.ted_hint')}</span>
        <div className="kg-terminal-ted-actions">
          <Button
            type="button"
            size="sm"
            variant="secondary"
            onClick={() => cancelRef.current()}
          >
            {t('shell.terminal.ted_cancel')}
          </Button>
          <Button
            type="button"
            size="sm"
            onClick={() => saveRef.current(viewRef.current?.state.doc.toString() ?? '')}
          >
            {t('shell.terminal.ted_apply')}
          </Button>
        </div>
      </header>
      <div ref={hostRef} className="kg-terminal-ted-editor" />
    </div>
  )
}
