import { createSignal } from 'solid-js'
import { KAppStatus, KForm, KFormField, KTable, t } from '@kaizengo/sdk-solid/ui'

export default function Users() {
  const [refreshToken, setRefreshToken] = createSignal(0)

  return (
    <>
      <KForm
        model="identity.user"
        submitLabel={t('identity.users.create')}
        successMessage={t('identity.created')}
        onsuccess={() => setRefreshToken((n) => n + 1)}
      >
        <KFormField field="name" label={t('identity.users.name')} placeholder={t('identity.users.placeholder.name')} />
        <KFormField field="email" label={t('identity.users.email')} placeholder={t('identity.users.placeholder.email')} />
      </KForm>

      <KTable
        model="identity.user"
        emptyMessage={t('identity.users.empty')}
        deletable={false}
        class="mt-4"
        refreshToken={refreshToken()}
      />

      <KAppStatus />
    </>
  )
}
