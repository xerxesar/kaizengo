import { ModelWorkspace } from '../lib/ModelWorkspace'
import { t } from '@kaizengo/sdk-solid/ui'

export default function CostLayers() {
  return (
    <ModelWorkspace
      model="inventory.cost_layer"
      title={t('inventory.cost_layers.title')}
      subtitle={t('inventory.cost_layers.subtitle')}
      emptyKey="inventory.empty.cost_layers"
      deletable={false}
      showForm={false}
    />
  )
}
