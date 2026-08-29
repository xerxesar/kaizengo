import { ModelWorkspace } from '../lib/ModelWorkspace'
import { t } from '@kaizengo/sdk-solid/ui'

export default function Stock() {
  return (
    <ModelWorkspace
      model="inventory.stock_quant"
      title={t('inventory.stock.title')}
      subtitle={t('inventory.stock.subtitle')}
      emptyKey="inventory.empty.stock"
      deletable={false}
      showForm={false}
    />
  )
}
