import { useState, useEffect, useRef } from 'react'
import dayjs from 'dayjs'
import { Form, Button, Upload, Select, DatePicker, message, Alert, Descriptions, Spin } from 'antd'
import { UploadOutlined } from '@ant-design/icons'
import { useSettlementStore } from '@/lib/settlement/store'
import { saveFile, hukouTypeApply, checkHuKouTypeApply, getUserHuKouInfo, ocrHouseholdStamp } from '@/lib/settlement/http'
import type { HukouChangeStatus, HuKouInfo } from '@/lib/settlement/types'
import { StepStatus, SettlementStep } from '@/lib/settlement/types'

export default function HukouUploadForm() {
  const [form] = Form.useForm()
  const [submitting, setSubmitting] = useState(false)
  const [hukouInfo, setHukouInfo] = useState<HuKouInfo | null>(null)
  const [infoLoading, setInfoLoading] = useState(true)
  const [uploadedFile, setUploadedFile] = useState<{ fileUUID: string; fileName: string } | null>(null)
  const [changeStatus, setChangeStatus] = useState<HukouChangeStatus | null>(null)
  const prevBlobRef = useRef<string>('')

  const { orderCode, updateStepStatus, setCurrentStep, addMessage, luohuDetail, refreshFlowStatus } =
    useSettlementStore()

  useEffect(() => {
    return () => { if (prevBlobRef.current) URL.revokeObjectURL(prevBlobRef.current) }
  }, [])

  useEffect(() => {
    const promises: Promise<void>[] = []
    promises.push(getUserHuKouInfo().then((res) => { if (res.obj) setHukouInfo(res.obj) }))
    if (orderCode) {
      promises.push(checkHuKouTypeApply({ orderCode }).then((res) => { if (res.obj) setChangeStatus(res.obj) }).catch(() => {}))
    }
    Promise.all(promises).finally(() => setInfoLoading(false))
  }, [])

  const runOcr = (file: File) => {
    ocrHouseholdStamp(file).then((ocrRes) => {
      if (ocrRes.success && ocrRes.obj) {
        const { household, stamp } = ocrRes.obj
        if (household?.item_list) {
          const registrationItem = household.item_list.find((item) => item.key === 'member_registration_date')
          if (registrationItem?.value) {
            const dateMatch = registrationItem.value.match(/(\d{4})年(\d{1,2})月(\d{1,2})日/)
            if (dateMatch) {
              const parsed = dayjs(`${dateMatch[1]}-${dateMatch[2].padStart(2, '0')}-${dateMatch[3].padStart(2, '0')}`)
              if (parsed.isValid()) form.setFieldsValue({ changeTime: parsed })
            }
          }
        }
        if (stamp?.details?.stamp) {
          const hasZhonghan = stamp.details.stamp.some((s) => s.type === '专用章' && s.value.includes('中韩派出所'))
          const address = hasZhonghan
            ? '山东省青岛市崂山区海尔路1号-山东省青岛市崂山区海尔路1号'
            : '山东省青岛市黄岛区前湾港路236号-山东省青岛市崂山区海尔路1号'
          form.setFieldsValue({ hukouAddress: address })
        }
      }
    }).catch(() => {})
  }

  const handleUpload = async (options: any) => {
    const { file, onSuccess, onError } = options
    const formData = new FormData()
    formData.append('file', file)
    formData.append('type', 'L9')
    formData.append('orderCode', orderCode || '')
    try {
      const res = await saveFile(formData) as any
      const fileUUID = res.fileUUID || res.result?.fileUUID
      const fileName = res.fileName || res.result?.fileName
      if (fileUUID) {
        setUploadedFile({ fileUUID, fileName })
        onSuccess({ fileUUID, fileName })
        runOcr(file as File)
      } else {
        onError(new Error('上传失败'))
        message.error('文件上传失败')
      }
    } catch (err) {
      onError(err)
      message.error('文件上传失败')
    }
  }

  const handleSubmit = async () => {
    try { await form.validateFields() } catch { return }
    if (!uploadedFile) { message.error('请先上传户口单页'); return }

    setSubmitting(true)
    try {
      const values = form.getFieldsValue()
      const luohuDetailDTO = { type: 'L9', fileUuid: uploadedFile.fileUUID, fileName: uploadedFile.fileName }
      const applyRes = await hukouTypeApply({
        categoryType: hukouInfo?.empGroupCode || '',
        categoryTypeName: hukouInfo?.empGroupName || '',
        categoryAddress: luohuDetail?.hkLocation || '',
        companyCategoryFlag: '0',
        newCategoryTypeName: '城镇',
        newCategoryAddress: values.hukouAddress || '',
        luohuMoveInDate: values.changeTime?.format('YYYY-MM-DD') || '',
        luohuDetailDTOString: JSON.stringify([luohuDetailDTO]),
      })

      if (applyRes.success) {
        await refreshFlowStatus()
        if (orderCode) {
          try {
            const checkRes = await checkHuKouTypeApply({ orderCode })
            if (checkRes.obj) setChangeStatus(checkRes.obj)
          } catch { /* ignore */ }
        }
        updateStepStatus(SettlementStep.HukouUpload, StepStatus.Completed, 100)
        updateStepStatus(SettlementStep.Download, StepStatus.WaitingUser)
        setCurrentStep(SettlementStep.Download)
        addMessage({ role: 'assistant', content: '户口单页上传成功，户口性质变更已提交！集体户首页已生成，请下载。', type: 'status_update' })
      } else {
        message.error(applyRes.msg || '户口性质变更提交失败，请联系HR')
      }
    } catch {
      message.error('提交失败，请稍后重试')
    } finally {
      setSubmitting(false)
    }
  }

  if (infoLoading) return <Spin size="small" />

  if (changeStatus) {
    return (
      <Form layout="vertical" size="small">
        <Alert type="success" message="户口性质变更已提交" className="mb-3" />
        {hukouInfo && (
          <Descriptions size="small" column={1} className="mb-3">
            <Descriptions.Item label="姓名">{hukouInfo.usName}</Descriptions.Item>
            <Descriptions.Item label="户口所在地">{hukouInfo.hukouLocation}</Descriptions.Item>
          </Descriptions>
        )}
        <Form.Item label="户口地址"><Select value={changeStatus.newCategoryAddress} disabled /></Form.Item>
        <Form.Item label="变更时间"><DatePicker value={null} placeholder={changeStatus.luohuMoveInDate} disabled style={{ width: '100%' }} /></Form.Item>
      </Form>
    )
  }

  return (
    <Form form={form} layout="vertical" size="small">
      <Alert type="info" message="户口迁移办结后，请上传户口单页，提交户口性质变更" className="mb-3" />
      {hukouInfo && (
        <Descriptions size="small" column={1} className="mb-3">
          <Descriptions.Item label="姓名">{hukouInfo.usName}</Descriptions.Item>
          <Descriptions.Item label="户口所在地">{hukouInfo.hukouLocation}</Descriptions.Item>
        </Descriptions>
      )}
      <Form.Item label="户口地址" name="hukouAddress" rules={[{ required: true, message: '请选择户口地址' }]}>
        <Select placeholder="请选择户口地址">
          <Select.Option value="山东省青岛市崂山区海尔路1号-山东省青岛市崂山区海尔路1号">崂山区海尔路1号</Select.Option>
          <Select.Option value="山东省青岛市黄岛区前湾港路236号-山东省青岛市崂山区海尔路1号">黄岛区前湾港路236号</Select.Option>
        </Select>
      </Form.Item>
      <Form.Item label="变更时间" name="changeTime" rules={[{ required: true, message: '请选择变更时间' }]}>
        <DatePicker style={{ width: '100%' }} placeholder="请选择变更时间" />
      </Form.Item>
      <Form.Item label="户口单页" required>
        <Upload maxCount={1} customRequest={handleUpload} listType="picture-card">
          <UploadOutlined /> 上传
        </Upload>
      </Form.Item>
      <Form.Item>
        <Button type="primary" onClick={handleSubmit} loading={submitting} disabled={!uploadedFile} block>提交</Button>
      </Form.Item>
    </Form>
  )
}
