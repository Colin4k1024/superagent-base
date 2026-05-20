import { useState } from 'react'
import { Form, Select, Button, Upload, message, Typography } from 'antd'
import type { UploadRequestOption } from 'rc-upload/lib/interface'
import { UploadOutlined } from '@ant-design/icons'
import { useSettlementStore } from '@/lib/settlement/store'
import { StepStatus, SettlementStep } from '@/lib/settlement/types'
import type { LuohuDetailFile } from '@/lib/settlement/types'
import { lhsq, saveFile, getAIToDeal } from '@/lib/settlement/http'

const { Text, Paragraph } = Typography

const HUKOU_MATERIAL_OPTIONS = [
  { value: 'hukou_book', label: '户口本' },
  { value: 'migration_cert', label: '户口迁移证' },
  { value: 'hukou_proof', label: '户籍证明' },
]

const FIELD_TYPE_MAP: Record<string, string> = {
  idFront: 'L6',
  idBack: 'L7',
  hukouBookFront: 'L1',
  hukouBookPersonal: 'L2',
  hukouBookIndex: 'L3',
  migrationCert: 'L4',
  hukouProof: 'L5',
  talentApproval: 'L8',
}

const FILE_TYPE_LABELS: Record<string, string> = {
  L1: '户口本首页',
  L2: '户口本个人页',
  L3: '户口索引页',
  L4: '户口迁移证',
  L5: '户籍证明',
  L6: '身份证正面',
  L7: '身份证反面',
  L8: '人才引进审核通过截图',
}

interface UploadedFile {
  fileUuid: string
  fileName: string
}

export default function ApplicationForm() {
  const [form] = Form.useForm()
  const [submitting, setSubmitting] = useState(false)
  const [currentTab, setCurrentTab] = useState<'guide' | 'form'>('guide')
  const materialType = Form.useWatch('hukouMaterial', form)
  const [uploadedFiles, setUploadedFiles] = useState<Record<string, UploadedFile>>({})

  const { steps, userInfo, flowNodes, addMessage, refreshFlowStatus, setOrderCode, setBaomingId, baomingId, luohuDetail } =
    useSettlementStore()

  const stepConfig = steps.find((s) => s.step === SettlementStep.Application)
  const isCompleted = stepConfig?.status === StepStatus.Completed

  const createCustomRequest = (fieldName: string) => {
    return async (options: UploadRequestOption) => {
      const { file, onSuccess, onError } = options
      const formData = new FormData()
      formData.append('file', file as File)
      try {
        const res: any = await saveFile(formData)
        const fileUUID = res.result?.fileUUID || res.fileUUID
        const serverFileName = res.result?.fileName || res.fileName || (file as File).name
        if (res.success && fileUUID) {
          setUploadedFiles((prev) => ({ ...prev, [fieldName]: { fileUuid: fileUUID, fileName: serverFileName } }))
          onSuccess?.(res)
        } else {
          message.error('图片上传失败，请重试')
          onError?.(new Error('upload failed'))
        }
      } catch (err) {
        message.error('图片上传失败，请重试')
        onError?.(err as Error)
      }
    }
  }

  const handleRemove = (fieldName: string) => {
    setUploadedFiles((prev) => {
      const next = { ...prev }
      delete next[fieldName]
      return next
    })
  }

  const handleSubmit = async (values: Record<string, unknown>) => {
    const fileFields: string[] = ['idFront', 'idBack', 'talentApproval']
    if (values.hukouMaterial === 'hukou_book') {
      fileFields.push('hukouBookFront', 'hukouBookPersonal', 'hukouBookIndex')
    } else if (values.hukouMaterial === 'migration_cert') {
      fileFields.push('migrationCert')
    } else {
      fileFields.push('hukouProof')
    }

    for (const field of fileFields) {
      if (!uploadedFiles[field]) {
        message.error('请等待所有图片上传完成')
        return
      }
    }

    setSubmitting(true)
    try {
      let currentOrderCode: string | null = null
      const node4 = flowNodes.find((n) => n.nodeNum === '4')
      if (node4?.orderCode) {
        currentOrderCode = node4.orderCode
      } else if (userInfo?.empNo) {
        const flowRes = await getAIToDeal({ userCode: userInfo.empNo })
        if (flowRes.success && Array.isArray(flowRes.result) && flowRes.result.length > 0) {
          const fetched4 = flowRes.result.find((n) => n.nodeNum === '4')
          currentOrderCode = fetched4?.orderCode || flowRes.result[0].orderCode
          setOrderCode(currentOrderCode)
          if (flowRes.code1) setBaomingId(flowRes.code1)
        }
      }
      if (!currentOrderCode) {
        message.error('未获取到流程编号，请稍后重试')
        setSubmitting(false)
        return
      }

      const jsonList = fileFields.map((field) => ({
        fileName: uploadedFiles[field].fileName,
        type: FIELD_TYPE_MAP[field],
        fileUuid: uploadedFiles[field].fileUuid,
        baomingId: baomingId || currentOrderCode,
      }))

      const res = await lhsq({
        orderCode: currentOrderCode,
        jsonListStr: JSON.stringify(jsonList),
        registeredOutsideProvince: '1',
        onlineMigration: '1',
      })

      if (res.success) {
        addMessage({ role: 'assistant', content: '落户申请提交成功！已进入准迁证办理环节，预计7天内完成。', type: 'status_update' })
        await refreshFlowStatus()
      }
    } catch {
      message.error('提交失败，请稍后重试')
    } finally {
      setSubmitting(false)
    }
  }

  const renderUpload = (fieldName: string) => (
    <Upload
      maxCount={1}
      accept="image/*"
      listType="picture-card"
      customRequest={createCustomRequest(fieldName)}
      onRemove={() => handleRemove(fieldName)}
    >
      <UploadOutlined /> 上传
    </Upload>
  )

  const renderFileItem = (file: LuohuDetailFile) => {
    const label = FILE_TYPE_LABELS[file.type] || file.type
    return (
      <div key={`${file.type}-${file.fileUuid}`} className="mb-2">
        <Text type="secondary" className="block text-xs mb-1">{label}</Text>
        <img src={file.fileUuid} alt={file.fileName} className="w-24 rounded" />
      </div>
    )
  }

  if (isCompleted) {
    if (luohuDetail?.luohuDetailDTOS?.length) {
      return <div>{luohuDetail.luohuDetailDTOS.map(renderFileItem)}</div>
    }
    return <Paragraph><Text type="success">落户申请已提交完成。</Text></Paragraph>
  }

  if (currentTab === 'guide') {
    return (
      <div className="text-sm">
        <Paragraph><Text strong>方式一：青岛人才网官网</Text></Paragraph>
        <Paragraph>
          1. 访问青岛人才网（rc.qingdao.gov.cn）<br />
          2. 注册/登录个人账号<br />
          3. 进入"人才引进落户"模块<br />
          4. 按提示完成学历认证申请<br />
          5. 等待审核通过后截图
        </Paragraph>
        <Paragraph><Text strong>方式二：微信公众号</Text></Paragraph>
        <Paragraph>
          1. 关注"青岛人才"微信公众号<br />
          2. 进入"人才服务" → "人才引进"<br />
          3. 按提示完成申请<br />
          4. 审核通过后截图保存
        </Paragraph>
        <Button type="primary" block onClick={() => setCurrentTab('form')} className="mt-3">
          已完成，开始上传材料
        </Button>
      </div>
    )
  }

  return (
    <Form form={form} layout="vertical" onFinish={handleSubmit} size="small">
      <Form.Item name="idFront" label="身份证正面" rules={[{ required: true, message: '请上传身份证正面' }]}>
        {renderUpload('idFront')}
      </Form.Item>
      <Form.Item name="idBack" label="身份证反面" rules={[{ required: true, message: '请上传身份证反面' }]}>
        {renderUpload('idBack')}
      </Form.Item>
      <Form.Item name="hukouMaterial" label="户籍资料类型" rules={[{ required: true }]} initialValue="hukou_book">
        <Select options={HUKOU_MATERIAL_OPTIONS} />
      </Form.Item>
      {materialType === 'hukou_book' && (
        <>
          <Form.Item name="hukouBookFront" label="户口本首页" rules={[{ required: true }]}>{renderUpload('hukouBookFront')}</Form.Item>
          <Form.Item name="hukouBookPersonal" label="户口本个人页" rules={[{ required: true }]}>{renderUpload('hukouBookPersonal')}</Form.Item>
          <Form.Item name="hukouBookIndex" label="户口索引页" rules={[{ required: true }]}>{renderUpload('hukouBookIndex')}</Form.Item>
        </>
      )}
      {materialType === 'migration_cert' && (
        <Form.Item name="migrationCert" label="户口迁移证" rules={[{ required: true }]}>{renderUpload('migrationCert')}</Form.Item>
      )}
      {materialType === 'hukou_proof' && (
        <Form.Item name="hukouProof" label="户籍证明" rules={[{ required: true }]}>{renderUpload('hukouProof')}</Form.Item>
      )}
      <Form.Item name="talentApproval" label="人才引进审核通过截图" rules={[{ required: true }]} extra="请使用青岛人才网官网截图">
        {renderUpload('talentApproval')}
      </Form.Item>
      <Form.Item>
        <Button type="primary" htmlType="submit" loading={submitting} block>提交申请</Button>
      </Form.Item>
    </Form>
  )
}
