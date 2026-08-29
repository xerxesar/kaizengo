import { ModelWorkspace } from '../lib/ModelWorkspace'
import { t } from '@kaizengo/sdk-solid/ui'

export default function Returns() {
  return (
    <ModelWorkspace
      model="inventory.rma"
      title={t('inventory.returns.title')}
      subtitle={t('inventory.returns.subtitle')}
      emptyKey="inventory.empty.returns"
    />
  )
}
