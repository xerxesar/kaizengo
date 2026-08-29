import { OperationsBoard } from '../lib/OperationsBoard'
import { t } from '@kaizengo/sdk-solid/ui'

export default function Operations() {
  return (
    <OperationsBoard
      title={t('inventory.operations.title')}
      subtitle={t('inventory.operations.subtitle')}
    />
  )
}
