import { t } from '@/lib'
import { KAppStatus, KTable } from '@/k'

export default function GreetingList() {
  return (
    <>
      <KTable
        query="hellospec.greetings"
        paginated
        searchable
        emptyMessage={t('hellospec.empty')}
      />
      <KAppStatus />
    </>
  )
}
