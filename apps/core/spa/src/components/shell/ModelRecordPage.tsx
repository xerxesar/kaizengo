import { useEffect, useState } from 'react'
import { KAppStatus, KCollection, KForm, KTable } from '@/k'
import { ScreenLayout } from '@/lib/page-view'
import type { KCollectionView } from '@/components/k/KCollection'
import { Progress } from '@/components/ui/progress'
import { getMenuContext } from '@/lib/layout-context'
import { fetchModelViews, type ModelView } from '@/lib/model-client'
import { contentAppForMenu } from '@/lib/spa-mount'

const collectionViews = new Set<KCollectionView>(['table', 'kanban', 'chart', 'pivot'])

/** List/form for a menu that names a model, or a dynamic `views:` entry. */
export function ModelRecordPage() {
  const menu = getMenuContext()
  const app = menu.selected ? contentAppForMenu(menu.selected, menu.app) : ''
  const viewName = menu.selected?.view?.trim()
  const model = menu.selected?.model?.trim()
  if (!app || (!model && !viewName)) return null
  if (viewName) {
    return (
      <>
        <SpecView app={app} name={viewName} />
        <KAppStatus app={app} />
      </>
    )
  }
  return (
    <>
      <KTable model={`${app}.${model}`} paginated searchable />
      <KAppStatus app={app} />
    </>
  )
}

function SpecView(props: { app: string; name: string }) {
  const [view, setView] = useState<ModelView | null>(null)
  const [error, setError] = useState('')

  useEffect(() => {
    let cancelled = false
    setError('')
    setView(null)
    fetchModelViews(props.app)
      .then((views) => {
        if (cancelled) return
        const found = views.find((item) => item.name === props.name) ?? null
        if (!found) setError(`No view ${props.app}.${props.name}`)
        setView(found)
      })
      .catch((e: unknown) => {
        if (!cancelled) setError(e instanceof Error ? e.message : String(e))
      })
    return () => {
      cancelled = true
    }
  }, [props.app, props.name])

  if (error) return <p className="text-sm text-[var(--kg-danger)]">{error}</p>
  if (!view) {
    return (
      <div className="flex items-center justify-center py-8" role="status" aria-label="Loading">
        <Progress indeterminate className="w-48 max-w-full" />
      </div>
    )
  }
  if (view.kind === 'screen' && view.layout) {
    return <ScreenLayout layout={view.layout} />
  }
  const modelRef = `${props.app}.${view.model}`
  if (view.kind === 'form') {
    return <KForm model={modelRef} view={view.name} />
  }
  const views = (view.views ?? []).filter((item): item is KCollectionView =>
    collectionViews.has(item as KCollectionView),
  )
  const preferred = collectionViews.has(view.defaultView as KCollectionView)
    ? (view.defaultView as KCollectionView)
    : undefined
  return (
    <KCollection
      model={modelRef}
      screen={view.name}
      paginated
      searchable
      views={views.length ? views : undefined}
      defaultView={preferred}
    />
  )
}
