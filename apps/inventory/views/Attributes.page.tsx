import { ModelWorkspace } from '../lib/ModelWorkspace'
import { t } from '@kaizengo/sdk-solid/ui'

export default function Attributes() {
  return (
    <>
      <ModelWorkspace model="inventory.attribute" title={t('inventory.attributes.title')} subtitle={t('inventory.attributes.subtitle')} emptyKey="inventory.empty.attributes" />
      <ModelWorkspace model="inventory.attribute_value" title={t('inventory.attribute_values.title')} subtitle={t('inventory.attribute_values.subtitle')} emptyKey="inventory.empty.attribute_values" />
    </>
  )
}
