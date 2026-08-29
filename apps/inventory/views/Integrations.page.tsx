import { ModelWorkspace } from '../lib/ModelWorkspace'
import { t } from '@kaizengo/sdk-solid/ui'

export default function Integrations() {
  return (
    <ModelWorkspace
      model="inventory.integration"
      title={t('inventory.integrations.title')}
      subtitle={t('inventory.integrations.subtitle')}
      emptyKey="inventory.empty.integrations"
    />
  )
}
