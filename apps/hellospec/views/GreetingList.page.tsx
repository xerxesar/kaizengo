import { t } from '@/lib'
import { KAppStatus, KCollection, KTable } from '@/k'
import type { HellospecGreeting } from '../lib/graphql'

export default function GreetingList() {
  return (
    <KCollection<HellospecGreeting>
      query="hellospec.greetings"
      paginated
      searchable
      views={['table', 'kanban', 'chart', 'pivot']}
      defaultView="kanban"
      keyOf={(greeting) => greeting.id}
      emptyMessage={t('appman.empty')}
      card={(model) => {
        return (
          <div className="flex min-h-0 flex-col justify-between gap-2 h-full overflow-hidden max-w-32">
            <div className="flex flex-col w-full gap-4 items-start justify-between gap-2">
              <div className="min-w-0 w-full">
                <h4 className="truncate overflow-hidden w-full text-sm font-semibold text-[var(--kg-text)]">
                  #{model.id}
                </h4>
              </div>
              <div className="truncate font-mono text-xs text-[var(--kg-text-muted)]">
                {model.message}
              </div>
            </div>
            <div
              className="line-clamp-2 text-sm leading-relaxed text-[var(--kg-warning)]"
              title={"sumtitle"}
            >
              {model.internalNote}
              <p className="m-0 truncate text-xs text-[var(--kg-text-muted)]">
                {model.authorId}
              </p>
            </div>
            
          </div>
        )
      }}
    />
  )
}
