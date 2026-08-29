import { ModelWorkspace } from '../lib/ModelWorkspace'
import { t } from '@kaizengo/sdk-solid/ui'

export default function Products() {
  return (
    <>
      <ModelWorkspace model="inventory.product" title={t('inventory.products.title')} subtitle={t('inventory.products.subtitle')} emptyKey="inventory.empty.products" />
      <ModelWorkspace model="inventory.product_variant" title={t('inventory.variants.title')} subtitle={t('inventory.variants.subtitle')} emptyKey="inventory.empty.variants" />
    </>
  )
}
