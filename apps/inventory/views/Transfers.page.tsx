import { OperationsBoard } from '../lib/OperationsBoard'
import { t } from '@kaizengo/sdk-solid/ui'

export default function Transfers() {
  return (
    <OperationsBoard
      title={t('inventory.transfers.title')}
      subtitle={t('inventory.transfers.subtitle')}
      pickingType="internal"
    />
  )
}
