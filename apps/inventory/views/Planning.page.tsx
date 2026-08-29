import { ModelWorkspace } from '../lib/ModelWorkspace'
import { t } from '@kaizengo/sdk-solid/ui'

export default function Planning() {
  return (
    <ModelWorkspace
      model="inventory.reorder_rule"
      title={t('inventory.planning.title')}
      subtitle={t('inventory.planning.subtitle')}
      emptyKey="inventory.empty.planning"
    />
  )
}
