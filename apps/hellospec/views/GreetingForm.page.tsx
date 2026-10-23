import { useState } from 'react'
import { t } from '@/lib'
import { KAppStatus, KForm, KFormField, KTable } from '@/k'

export default function GreetingForm() {
  const [refreshToken, setRefreshToken] = useState(0)

  function onFormSuccess() {
    setRefreshToken((n) => n + 1)
  }

  return (
    <>
      <p className="mb-4 max-w-2xl text-sm opacity-85">{t('hellospec.acl_hint')}</p>

      <KForm command="hellospec.postGreeting" onsuccess={onFormSuccess}>
        <KFormField
          field="message"
          label={t('hellospec.create')}
          placeholder={t('hellospec.new_placeholder')}
        />
        <KFormField field="mood" label={t('hellospec.mood')} />
        <KFormField field="internalNote" label={t('hellospec.internal_note')} />
      </KForm>

      <KTable
        query="hellospec.greetings"
        emptyMessage={t('hellospec.empty')}
        className="mt-4"
        refreshToken={refreshToken}
      />

      <KAppStatus />
    </>
  )
}
