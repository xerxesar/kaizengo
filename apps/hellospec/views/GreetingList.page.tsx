import { KAppStatus, KTable, t } from '@kaizengo/sdk-solid/ui'

export default function GreetingList() {
  return (
    <>
      <KTable model="hellospec.greeting" emptyMessage={t('hellospec.empty')} />
      <KAppStatus />
    </>
  )
}
