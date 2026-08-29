import { createSignal } from 'solid-js'
import { KAppStatus, KForm, KFormField, KTable, t } from '@kaizengo/sdk-solid/ui'

export default function GreetingForm() {
  const [refreshToken, setRefreshToken] = createSignal(0)

  function onFormSuccess() {
    setRefreshToken((n) => n + 1)
  }

  return (
    <>
      <p class="mb-4 max-w-2xl text-sm opacity-85">{t('hellospec.acl_hint')}</p>

      <KForm model="hellospec.greeting" onsuccess={onFormSuccess}>
        <KFormField field="message" label={t('hellospec.create')} placeholder={t('hellospec.new_placeholder')} />
        <KFormField field="mood" label={t('hellospec.mood')} />
        <KFormField field="internalNote" label={t('hellospec.internal_note')} />
      </KForm>

      <KTable
        model="hellospec.greeting"
        emptyMessage={t('hellospec.empty')}
        class="mt-4"
        refreshToken={refreshToken()}
      />

      <KAppStatus />
    </>
  )
}
