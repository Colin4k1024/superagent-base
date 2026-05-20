import { useEffect, useState } from 'react'
import { Form, Input, Select, Button, message, Spin } from 'antd'
import { useSettlementStore } from '@/lib/settlement/store'
import { baomingByLh, getListDictionaryByIdAndYqId, getJxByEmpCode } from '@/lib/settlement/http'
import { StepStatus, SettlementStep, type SettlementCondition } from '@/lib/settlement/types'

export default function RegistrationForm() {
  const [form] = Form.useForm()
  const [conditions, setConditions] = useState<SettlementCondition[]>([])
  const [submitting, setSubmitting] = useState(false)

  const { userInfo, steps, addMessage, refreshFlowStatus, baoMingDetail, aorderCode, orderCode, flowNodes, fetchBaoMingDetail } =
    useSettlementStore()

  const registrationStep = steps.find((s) => s.step === SettlementStep.Registration)
  const isCompleted = registrationStep?.status === StepStatus.Completed
  const isRejected = registrationStep?.status === StepStatus.Rejected
  const formDisabled = isCompleted && !isRejected

  useEffect(() => {
    if (!isCompleted || baoMingDetail) return
    const regNode = flowNodes.find((n) => Number(n.nodeNum) === 1)
    const code = aorderCode || orderCode || regNode?.orderCode
    if (code) fetchBaoMingDetail(code)
  }, [isCompleted, baoMingDetail, aorderCode, orderCode, flowNodes])

  useEffect(() => {
    if (baoMingDetail) {
      form.setFieldsValue({
        settleCondition: baoMingDetail.detailType,
        currentAddress: baoMingDetail.hkLocation,
        currentPolice: baoMingDetail.hkpcs,
        hasBoughtHouse: baoMingDetail.qdaoHouses === '1' ? '1' : '0',
        hasPropertyCert: baoMingDetail.property === '1' ? '1' : '0',
      })
    }
  }, [baoMingDetail, form])

  const useFallbackConditions = () => {
    const fallback = [
      { id: '1', name: '高层次人才' },
      { id: '2', name: '学历落户' },
      { id: '3', name: '技术技能落户' },
    ]
    setConditions(fallback)
    form.setFieldValue('settleCondition', '2')
  }

  useEffect(() => {
    if (userInfo) {
      form.setFieldsValue({
        usName: userInfo.usName,
        empNo: userInfo.empNo,
        phone: userInfo.userInfoDTO?.usTel || userInfo.phone,
        companyName: userInfo.companyName,
        department: userInfo.userInfoDTO?.department,
        jobname: userInfo.userInfoDTO?.jobname,
        highestdegree: userInfo.userInfoDTO?.highestdegree,
        currentAddress: userInfo.userInfoDTO?.hukouLocation || '',
        idCard: userInfo.empIdentify,
      })
    }

    getListDictionaryByIdAndYqId({ id: 'M2-02-01' })
      .then(async (res: any) => {
        const list = res.result || res.obj
        if (res.success && Array.isArray(list) && list.length > 0) {
          const mapped = list.map((item: any) => ({
            id: item.keyCode || item.id,
            name: item.keyValue || item.name,
          }))
          setConditions(mapped)
          let defaultName = '高层次人才落户'
          try {
            const jxRes = await getJxByEmpCode()
            if (jxRes.success) defaultName = '学历落户'
          } catch { /* ignore */ }
          const defaultItem = mapped.find((item: any) => item.name === defaultName) || mapped[0]
          form.setFieldValue('settleCondition', defaultItem.id)
        } else {
          useFallbackConditions()
        }
      })
      .catch(() => useFallbackConditions())
  }, [userInfo, form])

  const handleSubmit = async (values: Record<string, unknown>) => {
    if (values.hasBoughtHouse === '1' || values.hasPropertyCert === '1') {
      message.error('已在青岛购房或已取得房产证的人员不适用集体户落户，请咨询HR。')
      return
    }

    setSubmitting(true)
    try {
      const params = {
        detailType: values.settleCondition as string,
        type: 2,
        performance: 1,
        empCode: userInfo?.empNo,
        empTel: userInfo?.userInfoDTO?.usTel || userInfo?.phone,
        usId: userInfo?.empNo,
        usTel: userInfo?.userInfoDTO?.usTel || userInfo?.phone,
        hkLocation: values.currentAddress as string,
        hkpcs: values.currentPolice as string,
        qdaoHouses: Number(values.hasBoughtHouse) || 0,
        property: Number(values.hasPropertyCert) || 0,
      }
      const res: any = await baomingByLh(params)
      if (res.msg || res.message) message.warning(res.msg || res.message)
      if (res.success) {
        addMessage({ role: 'assistant', content: '报名提交成功！已进入HR审核环节，预计3个工作日内完成。', type: 'status_update' })
        await refreshFlowStatus()
      }
    } catch {
      message.error('提交失败，请稍后重试')
    } finally {
      setSubmitting(false)
    }
  }

  if (isCompleted && !baoMingDetail) return <Spin size="small" />

  return (
    <Form form={form} layout="vertical" onFinish={handleSubmit} disabled={formDisabled} size="small">
      <Form.Item name="usName" label="姓名"><Input disabled /></Form.Item>
      <Form.Item name="empNo" label="员工号"><Input disabled /></Form.Item>
      <Form.Item name="phone" label="手机号"><Input disabled /></Form.Item>
      <Form.Item name="companyName" label="所属公司"><Input disabled /></Form.Item>
      <Form.Item name="department" label="部门"><Input disabled /></Form.Item>
      <Form.Item name="jobname" label="岗位"><Input disabled /></Form.Item>
      <Form.Item name="highestdegree" label="最高学历"><Input disabled /></Form.Item>
      <Form.Item name="settleCondition" label="落户条件" rules={[{ required: true }]}>
        <Select placeholder="请选择落户条件">
          {conditions.map((c) => (
            <Select.Option key={c.id} value={c.id}>{c.name}</Select.Option>
          ))}
        </Select>
      </Form.Item>
      <Form.Item name="currentAddress" label="现户籍所在地" rules={[{ required: true, message: '请输入现户籍所在地' }]}>
        <Input placeholder="请输入现户籍所在地" />
      </Form.Item>
      <Form.Item name="idCard" hidden><Input /></Form.Item>
      <Form.Item name="currentPolice" label="现户籍派出所" rules={[{ required: true, message: '请输入现户籍派出所' }]}>
        <Input placeholder="请输入现户籍派出所" />
      </Form.Item>
      <Form.Item name="hasBoughtHouse" label="本人或配偶已在青岛购房" rules={[{ required: true }]} initialValue="0">
        <Select>
          <Select.Option value="0">否</Select.Option>
          <Select.Option value="1">是</Select.Option>
        </Select>
      </Form.Item>
      <Form.Item name="hasPropertyCert" label="已取得房产证" rules={[{ required: true }]} initialValue="0">
        <Select>
          <Select.Option value="0">否</Select.Option>
          <Select.Option value="1">是</Select.Option>
        </Select>
      </Form.Item>
      {!formDisabled && (
        <Form.Item>
          <Button type="primary" htmlType="submit" loading={submitting} block>提交报名</Button>
        </Form.Item>
      )}
    </Form>
  )
}
