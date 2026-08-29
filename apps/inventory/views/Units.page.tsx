import { ModelWorkspace } from '../lib/ModelWorkspace'
import { t } from '@kaizengo/sdk-solid/ui'

export default function Units() {
  return (
    <>
      <ModelWorkspace model="inventory.uom_group" title={t('inventory.uom_groups.title')} subtitle={t('inventory.uom_groups.subtitle')} emptyKey="inventory.empty.uom_groups" />
      <ModelWorkspace model="inventory.uom" title={t('inventory.uoms.title')} subtitle={t('inventory.uoms.subtitle')} emptyKey="inventory.empty.uoms" />
    </>
  )
}
