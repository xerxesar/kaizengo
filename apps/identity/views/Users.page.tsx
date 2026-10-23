import { useState } from 'react'
import { t } from '@/lib'
import { KAppStatus, KForm, KFormField, KTable } from '@/k'

export default function Users() {
  const [refreshToken, setRefreshToken] = useState(0)

  return (
    <>
      <KForm
        command="identity.postUser"
        submitLabel={t('identity.users.create')}
        successMessage={t('identity.created')}
        onsuccess={() => setRefreshToken((n) => n + 1)}
      >
        <KFormField field="name" label={t('identity.users.name')} placeholder={t('identity.users.placeholder.name')} />
        <KFormField field="email" label={t('identity.users.email')} placeholder={t('identity.users.placeholder.email')} />
      </KForm>

      <KTable
        query="identity.users"
        emptyMessage={t('identity.users.empty')}
        deletable={false}
        className="mt-4"
        refreshToken={refreshToken}
      />

      <KAppStatus />
    </>
  )
}
