import { createSignal, type JSX } from 'solid-js'
import {
  Card,
  FormField,
  KAppStatus,
  KForm,
  KTable,
  PageHeader,
  Select,
  t,
  type KFormFieldContext,
} from '@kaizengo/sdk-solid/ui'
import { enumOptions, renderInventoryField } from './form-field'
import { SyncValue } from './SyncValue'

type Props = {
  pickingType?: string
  title: string
  subtitle?: string
}

export function OperationsBoard(props: Props): JSX.Element {
  const [pickingRefresh, setPickingRefresh] = createSignal(0)
  const [moveRefresh, setMoveRefresh] = createSignal(0)

  function pickingField(ctx: KFormFieldContext) {
    if (ctx.field.key === 'pickingType' && props.pickingType) {
      return (
        <>
          <SyncValue value={props.pickingType} current={ctx.draft[ctx.field.key]} setValue={ctx.setValue} />
          <FormField label={ctx.label} required={ctx.field.required}>
            <Select value={props.pickingType} options={enumOptions('picking', 'pickingType')} disabled />
          </FormField>
        </>
      )
    }
    return renderInventoryField(ctx, 'picking', 'inventory')
  }

  function moveField(ctx: KFormFieldContext) {
    return renderInventoryField(ctx, 'stock_move', 'inventory')
  }

  return (
    <>
      <PageHeader title={props.title} subtitle={props.subtitle} />
      <div class="flex flex-col gap-5 [&_.composer]:flex-col [&_.composer]:items-stretch">
        <Card title={t('inventory.operations.header')}>
          <p class="mb-4 text-sm leading-relaxed text-[var(--kg-text-secondary)]">
            {t('inventory.operations.hint')}
          </p>
          <KForm model="inventory.picking" onsuccess={() => setPickingRefresh((n) => n + 1)} field={pickingField} />
          <KTable
            model="inventory.picking"
            emptyMessage={t('inventory.empty.operations')}
            refreshToken={pickingRefresh()}
          />
        </Card>

        <Card title={t('inventory.moves.title')}>
          <p class="mb-4 text-sm leading-relaxed text-[var(--kg-text-secondary)]">{t('inventory.moves.hint')}</p>
          <KForm model="inventory.stock_move" onsuccess={() => setMoveRefresh((n) => n + 1)} field={moveField} />
          <KTable
            model="inventory.stock_move"
            emptyMessage={t('inventory.empty.moves')}
            refreshToken={moveRefresh()}
          />
        </Card>
      </div>
      <KAppStatus />
    </>
  )
}
