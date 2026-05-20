import { create } from 'zustand'
import {
  StepStatus,
  SettlementStep,
  type StepConfig,
  type UserInfo,
  type FlowProgress,
  type FlowNodeItem,
  type SettlementChatMessage,
  type LuohuDetail,
  type BaoMingDetail,
} from './types'
import { getAIToDeal, getLuohuDetailsByOrderCode, getPcsByOrderCode, getLhBaoMingByOrderCode } from './http'

const INITIAL_STEPS: StepConfig[] = [
  { step: SettlementStep.Login, title: '系统登录', description: '我先获取你的工号，帮你登录人力云系统', status: StepStatus.NotStarted, progress: 0, actor: 'system' },
  { step: SettlementStep.Registration, title: '落户报名', description: '你满足在职人才引进-学历落户条件，正在为你自动提交信息', status: StepStatus.NotStarted, progress: 0, actor: 'user' },
  { step: SettlementStep.HrReview, title: 'HR审核（公议）', description: 'HR将在3个工作日内完成公议，我需为你实时同步公议结果', status: StepStatus.NotStarted, progress: 0, actor: 'admin' },
  { step: SettlementStep.TalentPermission, title: '人才网权限开通', description: '共享将集中开通【青岛人才网】权限，大约需要3个工作日', status: StepStatus.NotStarted, progress: 0, actor: 'admin' },
  { step: SettlementStep.Application, title: '落户申请', description: '本环节办理流程分为两步', status: StepStatus.NotStarted, progress: 0, actor: 'user' },
  { step: SettlementStep.PreparationCert, title: '准迁证办理', description: '共享正在代办准迁证：预计7天内办理完毕', status: StepStatus.NotStarted, progress: 0, actor: 'admin' },
  { step: SettlementStep.HukouMigration, title: '户口迁移', description: '请根据指引办理户口迁移', status: StepStatus.NotStarted, progress: 0, actor: 'user' },
  { step: SettlementStep.HukouUpload, title: '户口单页上传', description: '请按示例图上传照片/扫描件', status: StepStatus.NotStarted, progress: 0, actor: 'user' },
  { step: SettlementStep.Download, title: '集体户首页下载', description: '恭喜你落户到海尔集体户！', status: StepStatus.NotStarted, progress: 0, actor: 'system' },
]

interface SettlementState {
  steps: StepConfig[]
  currentStep: SettlementStep | null
  userInfo: UserInfo | null
  flowProgress: FlowProgress | null
  flowNodes: FlowNodeItem[]
  messages: SettlementChatMessage[]
  orderCode: string | null
  aorderCode: string | null
  baomingId: string | null
  baoMingDetail: BaoMingDetail | null
  flowStarted: boolean
  loading: boolean
  luohuDetail: LuohuDetail | null
  policeStation: string | null

  setUserInfo: (info: UserInfo) => void
  setCurrentStep: (step: SettlementStep) => void
  updateStepStatus: (step: SettlementStep, status: StepStatus, progress?: number) => void
  addMessage: (message: Omit<SettlementChatMessage, 'id' | 'timestamp'>) => void
  setOrderCode: (code: string) => void
  setAorderCode: (code: string) => void
  setBaomingId: (id: string) => void
  setFlowStarted: (started: boolean) => void
  setLoading: (loading: boolean) => void
  fetchPoliceStation: (orderCode: string) => Promise<void>
  fetchBaoMingDetail: (orderCode: string) => Promise<void>
  fetchLuohuDetail: (orderCode: string) => Promise<void>
  resetFlow: () => void
  syncFromNodeList: (nodes: FlowNodeItem[]) => void
  refreshFlowStatus: () => Promise<void>
}

let messageIdCounter = 0

export const useSettlementStore = create<SettlementState>((set, get) => ({
  steps: INITIAL_STEPS,
  currentStep: null,
  userInfo: null,
  flowProgress: null,
  flowNodes: [],
  messages: [],
  orderCode: null,
  aorderCode: null,
  baomingId: null,
  baoMingDetail: null,
  flowStarted: false,
  loading: false,
  luohuDetail: null,
  policeStation: null,

  setUserInfo: (info) => set({ userInfo: info }),
  setCurrentStep: (step) => set({ currentStep: step }),

  updateStepStatus: (step, status, progress) =>
    set((state) => ({
      steps: state.steps.map((s) =>
        s.step === step
          ? { ...s, status, progress: progress ?? (status === StepStatus.Completed ? 100 : s.progress) }
          : s,
      ),
    })),

  addMessage: (message) => {
    messageIdCounter += 1
    set((state) => ({
      messages: [
        ...state.messages,
        { ...message, id: `msg-${messageIdCounter}`, timestamp: Date.now() },
      ],
    }))
  },

  setOrderCode: (code) => set({ orderCode: code }),
  setAorderCode: (code) => set({ aorderCode: code }),
  setBaomingId: (id) => set({ baomingId: id }),
  setFlowStarted: (started) => set({ flowStarted: started }),
  setLoading: (loading) => set({ loading }),

  fetchPoliceStation: async (orderCode: string) => {
    try {
      const res = await getPcsByOrderCode({ orderCode })
      if (res.success && res.obj) {
        set({ policeStation: res.obj.policeStation || '' })
      }
    } catch { /* ignore */ }
  },

  fetchBaoMingDetail: async (orderCode: string) => {
    try {
      const res: any = await getLhBaoMingByOrderCode({ orderCode })
      const detail = res.result || res.obj
      if (res.success && detail) {
        set({ baoMingDetail: detail })
      }
    } catch { /* ignore */ }
  },

  fetchLuohuDetail: async (orderCode: string) => {
    try {
      const res = await getLuohuDetailsByOrderCode({ orderCode })
      if (res && (res as any).id) {
        set({ luohuDetail: res as LuohuDetail })
      }
    } catch { /* ignore */ }
  },

  resetFlow: () =>
    set({
      steps: INITIAL_STEPS,
      currentStep: null,
      flowProgress: null,
      flowNodes: [],
      messages: [],
      orderCode: null,
      aorderCode: null,
      baomingId: null,
      baoMingDetail: null,
      flowStarted: false,
      luohuDetail: null,
      policeStation: null,
    }),

  syncFromNodeList: (nodes) => {
    const { updateStepStatus, setCurrentStep, setOrderCode, fetchLuohuDetail, fetchPoliceStation, fetchBaoMingDetail, aorderCode } = get()

    set({ flowNodes: nodes })

    if (nodes.length > 0) {
      setOrderCode(nodes[0].orderCode)
    }

    const NODE_NUM_TO_STEP: Record<number, SettlementStep> = {
      1: SettlementStep.Registration,
      2: SettlementStep.HrReview,
      3: SettlementStep.TalentPermission,
      4: SettlementStep.Application,
      5: SettlementStep.PreparationCert,
      6: SettlementStep.HukouMigration,
      7: SettlementStep.HukouUpload,
      8: SettlementStep.Download,
    }

    const mapNodeStatus = (nodeStatus: string): StepStatus => {
      switch (nodeStatus) {
        case '2':
        case '3':
          return StepStatus.Completed
        case '1':
          return StepStatus.InProgress
        case '4':
          return StepStatus.Terminated
        default:
          return StepStatus.NotStarted
      }
    }

    updateStepStatus(SettlementStep.Login, StepStatus.Completed, 100)

    let latestInProgressStep: SettlementStep | null = null
    let applicationOrderCode: string | null = null
    let migrationOrderCode: string | null = null
    let registrationCompleted = false
    let registrationOrderCode: string | null = null

    for (const node of nodes) {
      const nodeNum = Number(node.nodeNum)
      const frontStep = NODE_NUM_TO_STEP[nodeNum]
      if (!frontStep) continue

      const status = mapNodeStatus(node.nodeStatus)
      const progress = status === StepStatus.Completed ? 100 : status === StepStatus.InProgress ? 50 : 0
      updateStepStatus(frontStep, status, progress)

      if (node.nodeStatus === '1') {
        latestInProgressStep = frontStep
      }

      if (nodeNum === 1 && (node.nodeStatus === '2' || node.nodeStatus === '3')) {
        registrationCompleted = true
        registrationOrderCode = node.orderCode
      }

      if (nodeNum === 4 && (node.nodeStatus === '2' || node.nodeStatus === '3')) {
        applicationOrderCode = node.orderCode
      }

      if (nodeNum === 6 && node.nodeStatus === '1') {
        migrationOrderCode = node.orderCode
      }
    }

    if (latestInProgressStep) {
      setCurrentStep(latestInProgressStep)
    }

    if (registrationCompleted) {
      const bmCode = aorderCode || registrationOrderCode
      if (bmCode) fetchBaoMingDetail(bmCode)
    }

    if (applicationOrderCode) {
      fetchLuohuDetail(applicationOrderCode)
    }

    if (migrationOrderCode) {
      fetchPoliceStation(migrationOrderCode)
    }
  },

  refreshFlowStatus: async () => {
    const { userInfo, syncFromNodeList, setBaomingId, setAorderCode } = get()
    if (!userInfo?.empNo) return
    try {
      const res = await getAIToDeal({ userCode: userInfo.empNo })
      if (res.success && res.result && Array.isArray(res.result) && res.result.length > 0) {
        if (res.code1) setBaomingId(res.code1)
        if (res.aorderCode) setAorderCode(res.aorderCode)
        syncFromNodeList(res.result)
      }
    } catch { /* ignore */ }
  },
}))
