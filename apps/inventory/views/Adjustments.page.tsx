import { ModelWorkspace } from '../lib/ModelWorkspace'
import { t } from '@kaizengo/sdk-solid/ui'

export default function Adjustments() {
  return (
    <ModelWorkspace
      model="inventory.adjustment"
      title={t('inventory.adjustments.title')}
      subtitle={t('inventory.adjustments.subtitle')}
      emptyKey="inventory.empty.adjustments"
    />
  )
}
