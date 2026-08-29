import { ModelWorkspace } from '../lib/ModelWorkspace'
import { t } from '@kaizengo/sdk-solid/ui'

export default function Hardware() {
  return (
    <ModelWorkspace
      model="inventory.iot_device"
      title={t('inventory.hardware.title')}
      subtitle={t('inventory.hardware.subtitle')}
      emptyKey="inventory.empty.hardware"
    />
  )
}
