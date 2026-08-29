import { OperationsBoard } from '../lib/OperationsBoard'
import { t } from '@kaizengo/sdk-solid/ui'

export default function Fulfillment() {
  return (
    <OperationsBoard
      title={t('inventory.fulfillment.title')}
      subtitle={t('inventory.fulfillment.subtitle')}
      pickingType="outgoing"
    />
  )
}
