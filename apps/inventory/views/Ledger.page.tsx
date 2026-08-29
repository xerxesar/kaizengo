import { ModelWorkspace } from '../lib/ModelWorkspace'
import { t } from '@kaizengo/sdk-solid/ui'

export default function Ledger() {
  return (
    <ModelWorkspace
      model="inventory.stock_ledger"
      title={t('inventory.ledger.title')}
      subtitle={t('inventory.ledger.subtitle')}
      emptyKey="inventory.empty.ledger"
      deletable={false}
      showForm={false}
    />
  )
}
