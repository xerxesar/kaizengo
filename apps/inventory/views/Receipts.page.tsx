import { OperationsBoard } from '../lib/OperationsBoard'
import { t } from '@kaizengo/sdk-solid/ui'

export default function Receipts() {
  return (
    <OperationsBoard
      title={t('inventory.receipts.title')}
      subtitle={t('inventory.receipts.subtitle')}
      pickingType="incoming"
    />
  )
}
