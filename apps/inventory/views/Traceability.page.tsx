import { ModelWorkspace } from '../lib/ModelWorkspace'
import { t } from '@kaizengo/sdk-solid/ui'

export default function Traceability() {
  return (
    <>
      <ModelWorkspace model="inventory.stock_lot" title={t('inventory.lots.title')} subtitle={t('inventory.lots.subtitle')} emptyKey="inventory.empty.lots" />
      <ModelWorkspace model="inventory.stock_serial" title={t('inventory.serials.title')} subtitle={t('inventory.serials.subtitle')} emptyKey="inventory.empty.serials" />
    </>
  )
}
