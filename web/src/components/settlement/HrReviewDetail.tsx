import { Button } from 'antd'
import { CheckCircleFilled, ClockCircleFilled } from '@ant-design/icons'
import { useSettlementStore } from '@/lib/settlement/store'
import { StepStatus, SettlementStep } from '@/lib/settlement/types'

const STEP_TO_NODE_NUM: Partial<Record<SettlementStep, number>> = {
  [SettlementStep.HrReview]: 2,
  [SettlementStep.TalentPermission]: 3,
  [SettlementStep.PreparationCert]: 5,
}

const STEP_TITLE: Partial<Record<SettlementStep, string>> = {
  [SettlementStep.HrReview]: '审核人',
  [SettlementStep.TalentPermission]: '经办人',
  [SettlementStep.PreparationCert]: '经办人',
}

const STEP_OPINION_LABEL: Partial<Record<SettlementStep, string>> = {
  [SettlementStep.HrReview]: '公议意见：',
  [SettlementStep.TalentPermission]: '经办意见：',
  [SettlementStep.PreparationCert]: '经办意见：',
}

interface Props {
  step?: SettlementStep
}

export default function HrReviewDetail({ step = SettlementStep.HrReview }: Props) {
  const { steps, flowNodes } = useSettlementStore()

  const stepConfig = steps.find((s) => s.step === step)
  const isCompleted = stepConfig?.status === StepStatus.Completed
  const isInProgress = stepConfig?.status === StepStatus.InProgress

  const nodeNum = STEP_TO_NODE_NUM[step]
  const node = nodeNum ? flowNodes.find((n) => Number(n.nodeNum) === nodeNum) : undefined
  const title = STEP_TITLE[step] || '审核流程'

  if (!node) {
    return (
      <div className="p-3">
        <div className="font-medium text-sm mb-2">{title}</div>
        <div className="text-gray-400 text-xs">暂无审核信息</div>
      </div>
    )
  }

  return (
    <div className="p-3">
      <div className="font-medium text-sm mb-3">{title}</div>
      <div className="space-y-2">
        <div className="flex items-center gap-2">
          {isCompleted && <CheckCircleFilled className="text-green-500" />}
          {isInProgress && <ClockCircleFilled className="text-blue-500" />}
          <span className={`text-sm ${isInProgress ? 'text-blue-600 font-medium' : ''}`}>
            {node.operatorName}
          </span>
          <span className="text-xs text-gray-400">{node.operatorCode}</span>
        </div>
        {(isCompleted || node.data2) && (
          <div className="text-xs text-gray-500 pl-6">
            {STEP_OPINION_LABEL[step] || '意见：'}{node.data2 || '通过'}
          </div>
        )}
      </div>
      {isInProgress && (
        <div className="mt-3">
          <Button type="primary" size="small">催办</Button>
        </div>
      )}
    </div>
  )
}
