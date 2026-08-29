import { createSignal, Show, type JSX } from 'solid-js'
import {
  Card,
  KAppStatus,
  KForm,
  KTable,
  PageHeader,
  parseNamespace,
  t,
} from '@kaizengo/sdk-solid/ui'
import { renderInventoryField } from './form-field'

type Props = {
  model: string
  title: string
  subtitle?: string
  emptyKey?: string
  formTitle?: string
  deletable?: boolean
  showForm?: boolean
}

export function ModelWorkspace(props: Props): JSX.Element {
  const [refreshToken, setRefreshToken] = createSignal(0)
  const ns = () => parseNamespace(props.model)
  const showForm = () => props.showForm ?? true
  const deletable = () => props.deletable ?? true

  return (
    <>
      <PageHeader title={props.title} subtitle={props.subtitle} />
      <div class="flex flex-col gap-5 [&_.composer]:flex-col [&_.composer]:items-stretch">
        <Show when={showForm()}>
          <Card title={props.formTitle || t('inventory.form.new')}>
            <KForm
              model={props.model}
              onsuccess={() => setRefreshToken((n) => n + 1)}
              field={(ctx) => renderInventoryField(ctx, ns().name, ns().app)}
            />
          </Card>
        </Show>
        <KTable
          model={props.model}
          emptyMessage={t(props.emptyKey ?? 'inventory.empty')}
          deletable={deletable()}
          refreshToken={refreshToken()}
        />
      </div>
      <KAppStatus />
    </>
  )
}
