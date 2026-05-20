export enum StepStatus {
  NotStarted = 'not_started',
  InProgress = 'in_progress',
  WaitingUser = 'waiting_user',
  Completed = 'completed',
  Rejected = 'rejected',
  Terminated = 'terminated',
}

export enum SettlementStep {
  Login = 1,
  Registration = 2,
  HrReview = 3,
  TalentPermission = 4,
  Application = 5,
  PreparationCert = 6,
  HukouMigration = 7,
  HukouUpload = 8,
  Download = 9,
}

export interface StepConfig {
  step: SettlementStep
  title: string
  description: string
  runtimeDescription?: string
  status: StepStatus
  progress: number
  actor: 'system' | 'user' | 'admin'
}

export interface UserInfo {
  phone: string
  companyCity: string
  provName: string
  organizationName: string
  cityName: string
  usName: string
  cityCode: string
  empNo: string
  companyName: string
  companyCode: string
  organizationCode: string
  empIdentify: string
  usBirthday: string
  email: string
  inHaierDate: string
  role: string
  provCode: string
  firstWorkingDay: string
  userInfoDTO: {
    usId: string
    usName: string
    usTel: string
    usEmail: string
    usSex: string
    usIdentify: string
    usCompany: string
    companyName: string
    highestdegree: string
    usParea: string
    hukouLocation: string
    company: string
    inHaierDate: string
    firstworkingday: string
    marriagestatus: string
    jobname: string
    firstdegreegraschool: string
    firstdegreeenddate: string
    firstdegreegramajor: string
    industrialCode: string
    empStatusCode: string
    empGroupCode: string
    empGroupName: string
    empSubGroupCode: string
    bandcode: string
    contract: string
    department: string
    insertBy: string
    personelareacode: string
  }
}

export interface SettlementCondition {
  id?: string
  name?: string
  description?: string
  keyCode?: string
  keyValue?: string
  keyType?: string
  richText?: string
}

export interface FlowProgress {
  orderCode: string
  currentStep: number
  status: string
  rejectReason?: string
  terminateReason?: string
  approver?: string
  approverName?: string
}

export interface FlowNodeItem {
  orderCode: string
  nodeName: string
  nodeNum: string
  nodeStatus: string
  nodeStatusName: string
  insertTime: string
  operatorCode: string
  operatorName: string
  operatorTime: string
  businessCode: string
  businessName: string
  displayName: string
  nodeCount: string
  standardNodeNum: string
  empCode: string
  empName: string
  parkName: string
  sceneCode: string
  messageFlag: string
  isEvaluable: string
  forShow: string
  dataFrom: string
  toJudge?: string
  data2?: string
}

export interface SettlementChatMessage {
  id: string
  role: 'user' | 'assistant' | 'system'
  content: string
  timestamp: number
  type: 'text' | 'process_card' | 'form_trigger' | 'status_update'
  metadata?: Record<string, unknown>
}

export interface LuohuDetailFile {
  type: string
  fileUuid: string
  fileName: string
}

export interface LuohuDetail {
  id: string
  type: string
  firstDegree: string
  school: string
  hkLocation: string
  qdaoHouses: string
  property: string
  graduationTime: string
  hkpcs: string
  residType: string
  registeredOutsideProvince: string
  onlineMigration: string
  receiveMode: string
  nationCode: string
  nationName: string
  luohuDetailDTOS: LuohuDetailFile[]
}

export interface BaoMingDetail {
  id: string
  empCode: string
  orderCode: string
  type: string
  firstDegree: string
  performance: string
  presentPost: string
  school: string
  empTel: string
  contract: string
  entryTime: string
  hkLocation: string
  qdaoHouses: string
  property: string
  createDate: string
  isDel: string
  graduationTime: string
  hkpcs: string
  isZxApproval: string
  step: string
  status: string
  detailType: string
  zxApprovalDate: string
  zxApprovalRemark: string
  residType: string
  isBm: string
  empIdentify: string
  department: string
  empname: string
}

export interface HukouChangeStatus {
  intId: number
  empCode: string
  orderCode: string
  type: string
  categoryType: string
  categoryTypeName: string
  categoryAddress: string
  newCategoryType: string
  newCategoryTypeName: string
  newCategoryAddress: string
  companyCategoryFlag: string
  luohuMoveInDate: string
  empname: string
  luohuDetailDTOS?: Array<{ type: string; fileUuid: string; fileName: string }>
}

export interface HuKouInfo {
  usId: string
  usName: string
  usTel: string
  usEmail: string
  usSex: string
  usIdentify: string
  usCompany: string
  companyName: string
  highestdegree: string
  usParea: string
  hukouLocation: string
  company: string
  inHaierDate: string
  contractStartDate: string
  contractEndDate: string
  firstworkingday: string
  marriagestatus: string
  jobname: string
  firstdegreegraschool: string
  firstdegreeenddate: string
  firstdegreegramajor: string
  industrialCode: string
  empStatusCode: string
  empStatusName: string
  empGroupCode: string
  empGroupName: string
  empSubGroupCode: string
  bandcode: string
  contract: string
  department: string
  insertBy: string
  nationCode: string
  nationName: string
  personelareacode: string
}
