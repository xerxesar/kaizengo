import { createContext, useContext, useEffect, useState, type CSSProperties, type ReactNode } from 'react'
import { Progress } from '@/components/ui/progress'
import { KCollection, type KCollectionView } from '@/components/k/KCollection'
import { KForm } from '@/components/k/KForm'
import { fetchModelViews, type ModelLayoutNode, type ModelView } from '@/lib/model-client'

const collectionViews = new Set<KCollectionView>(['table', 'kanban', 'chart', 'pivot'])

type PageView = { app: string; view: string }

const PageViewContext = createContext<PageView>({ app: '', view: '' })

/** ViewHost sets this to the menu's view name while a page is mounted. */
export function PageViewProvider({ app, view, children }: PageView & { children: ReactNode }) {
  return <PageViewContext.Provider value={{ app, view }}>{children}</PageViewContext.Provider>
}

export function usePageView(): PageView {
  return useContext(PageViewContext)
}

function trackSize(node: ModelLayoutNode): string {
  if (node.size) return node.size
  const span = node.span && node.span > 0 ? node.span : 1
  return `minmax(0, ${span}fr)`
}

/** Renders a screen layout tree: nested rows and columns, sized by span or size. */
export function ScreenLayout({ layout }: { layout: ModelLayoutNode }) {
  return <LayoutFrame node={layout} />
}

function LayoutFrame({ node }: { node: ModelLayoutNode }) {
  if (node.component) return <SpecScreen id={node.component} />
  const items = node.items ?? []
  const row = (node.direction ?? 'column') === 'row'
  const template = items.map(trackSize).join(' ') || 'minmax(0, 1fr)'
  const style: CSSProperties = {
    display: 'grid',
    gap: '1.5rem',
    alignItems: 'start',
    minWidth: 0,
  }
  if (row) style.gridTemplateColumns = template
  else style.gridTemplateRows = template
  return (
    <div style={style}>
      {items.map((child, i) => (
        <div key={child.component || i} className="min-w-0">
          <LayoutFrame node={child} />
        </div>
      ))}
    </div>
  )
}

/** Loads the current page's layout tree and renders it. */
export function SpecLayout() {
  const { app, view } = usePageView()
  const [layout, setLayout] = useState<ModelLayoutNode | null>(null)
  const [error, setError] = useState('')

  useEffect(() => {
    if (!app || !view) {
      setError('This page has no view domain')
      return
    }
    let cancelled = false
    setError('')
    setLayout(null)
    fetchModelViews(app)
      .then((views) => {
        if (cancelled) return
        const found = views.find((item) => item.name === view && item.kind === 'screen') ?? null
        if (!found?.layout) setError(`View ${app}.${view} has no layout`)
        else setLayout(found.layout)
      })
      .catch((e: unknown) => {
        if (!cancelled) setError(e instanceof Error ? e.message : String(e))
      })
    return () => {
      cancelled = true
    }
  }, [app, view])

  if (error) return <p className="text-sm text-[var(--kg-danger)]">{error}</p>
  if (!layout) {
    return (
      <div className="flex items-center justify-center py-8" role="status" aria-label="Loading">
        <Progress indeterminate className="w-48 max-w-full" />
      </div>
    )
  }
  return <ScreenLayout layout={layout} />
}

/**
 * Renders one collection or form from the page spec (`components[].id`).
 * The page stays compiled; an admin edits that component's layout and model.
 */
export function SpecScreen({ id }: { id: string }) {
  const { app, view } = usePageView()
  const [spec, setSpec] = useState<ModelView | null>(null)
  const [error, setError] = useState('')
  const screen = view && id ? `${view}.${id}` : ''

  useEffect(() => {
    if (!app || !view) {
      setError('This page has no view domain')
      return
    }
    if (!id.trim()) {
      setError('SpecScreen requires an id')
      return
    }
    let cancelled = false
    setError('')
    setSpec(null)
    fetchModelViews(app)
      .then((views) => {
        if (cancelled) return
        const found = views.find((item) => item.name === screen) ?? null
        if (!found) setError(`No component ${app}.${screen} (add components[].id in the view spec)`)
        setSpec(found)
      })
      .catch((e: unknown) => {
        if (!cancelled) setError(e instanceof Error ? e.message : String(e))
      })
    return () => {
      cancelled = true
    }
  }, [app, view, id, screen])

  if (error) return <p className="text-sm text-[var(--kg-danger)]">{error}</p>
  if (!spec) {
    return (
      <div className="flex items-center justify-center py-8" role="status" aria-label="Loading">
        <Progress indeterminate className="w-48 max-w-full" />
      </div>
    )
  }
  const modelRef = `${app}.${spec.model}`
  if (spec.kind === 'form') {
    return <KForm model={modelRef} view={spec.name} />
  }
  const modes = (spec.views ?? []).filter((item): item is KCollectionView =>
    collectionViews.has(item as KCollectionView),
  )
  const preferred = collectionViews.has(spec.defaultView as KCollectionView)
    ? (spec.defaultView as KCollectionView)
    : undefined
  const groupBy = spec.groupBy?.[0]
  return (
    <KCollection
      model={modelRef}
      screen={spec.name}
      paginated
      searchable
      views={modes.length ? modes : undefined}
      defaultView={preferred}
      groupByField={groupBy}
    />
  )
}
