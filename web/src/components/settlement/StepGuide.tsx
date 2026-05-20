import { useEffect, useState } from 'react'
import { Typography, Button, message, Spin } from 'antd'
import { DownloadOutlined } from '@ant-design/icons'
import { useSettlementStore } from '@/lib/settlement/store'
import { SettlementStep } from '@/lib/settlement/types'

const { Title, Paragraph, Text } = Typography

interface Props {
  step: SettlementStep
}

export default function StepGuide({ step }: Props) {
  const { orderCode, luohuDetail, flowNodes, fetchLuohuDetail, policeStation, fetchPoliceStation } = useSettlementStore()
  const [loading, setLoading] = useState(false)

  const effectiveOrderCode = orderCode || flowNodes[0]?.orderCode || ''

  const node6 = flowNodes.find((n) => n.nodeNum === '6')
  const migrationMode: 'online' | 'offline' = node6?.toJudge === '1' ? 'offline' : 'online'

  useEffect(() => {
    if (step === SettlementStep.HukouMigration && effectiveOrderCode) {
      setLoading(true)
      const promises: Promise<void>[] = []
      if (!policeStation) promises.push(fetchPoliceStation(effectiveOrderCode))
      if (!luohuDetail) {
        const node4 = flowNodes.find((n) => n.nodeNum === '4')
        promises.push(fetchLuohuDetail(node4?.orderCode || effectiveOrderCode))
      }
      if (promises.length > 0) {
        Promise.all(promises).finally(() => setLoading(false))
      } else {
        setLoading(false)
      }
    }
  }, [step, effectiveOrderCode])

  if (step === SettlementStep.HukouMigration) {
    if (loading) return <Spin size="small" />

    const l11File = luohuDetail?.luohuDetailDTOS?.find((f) => f.type === 'L11')
    const stationName = policeStation || '派出所（加载中）'

    if (migrationMode === 'online') {
      return (
        <Typography>
          <Paragraph>请根据以下指引前往【{stationName}】办理户口迁移</Paragraph>
          <Paragraph><Text strong>材料明细</Text></Paragraph>
          <Paragraph>本人携带以下材料到派出所办理落户:</Paragraph>
          <Paragraph>
            1. 《单位同意落户证明》{l11File ? <a href={l11File.fileUuid} target="_blank" rel="noopener noreferrer">下载</a> : '下载'}(彩色打印)<br />
            2. 身份证原件、复印件<br />
            3. 原户口首页复印件,个人单页原件及复印件<br />
            4. 结婚证原件、复印件(已婚需提供)<br />
            5. 《不动产交易登记信息(现势加历史)查询结果证明》(黑白打印即可)文件查询路径如下:<br />
            <Text style={{ paddingLeft: 16, display: 'block' }}>
              a. 下载并打开"爱山东"APP点击查看全部<br />
              b. 点击"住房保障",点击"不动产登记一网通办"<br />
              c. 同意授权<br />
              d. 选择青岛市<br />
              e. 点击"开具不动产登记查询证明"<br />
              f. 点击"现势加历史",完善信息后,点击确定后,截图黑白打印即可
            </Text>
          </Paragraph>
        </Typography>
      )
    }

    return (
      <Typography>
        <Paragraph>请根据以下指引前往【{stationName}】办理户口迁移</Paragraph>
        <Paragraph><Text strong>材料明细</Text></Paragraph>
        <Paragraph>
          1、联系原籍派出所,告知青岛已为您办理网上迁移,让其为您办理网上迁移;<br />
          注:《准予迁入证明》(40天内有效),如原籍派出所不能办理网上迁移,需先持《准予迁入证明》原件第二联回原籍派出所开具《户口迁移证》
        </Paragraph>
        <Paragraph>2、办理完毕的2个工作日后本人携带以下材料到派出所办理落户。</Paragraph>
        <Paragraph>
          <Text strong>(1)从人力共享领取的:</Text><br />
          ①《准予迁入证明》(第三联原件)上迁移,(如办理的网上迁移,无此材料);<br />
          ②《办理户籍业务事项申请表》;<br />
          ③人才引进查询截图、不动产登记结果查询截图;《单位同意落户证明》原件;
        </Paragraph>
        <Paragraph>
          <Text strong>(2)个人准备:</Text><br />
          ①《户口迁移证》原件(如办理的网上迁移,无此材料,但需要户口本原件);<br />
          ②身份证原件、复印件;<br />
          ③结婚证原件、复印件(已婚需提供)。
        </Paragraph>
        <Paragraph>
          《不动产交易登记信息(现势加历史)查询结果证明》(黑白打印即可)文件查询路径如下:<br />
          <Text style={{ paddingLeft: 16, display: 'block' }}>
            a. 下载并打开"爱山东"APP点击查看全部<br />
            b. 点击"住房保障",点击"不动产登记一网通办"<br />
            c. 同意授权<br />
            d. 选择青岛市<br />
            e. 点击"开具不动产登记查询证明"<br />
            f. 点击"现势加历史",完善信息后,点击确定后,截图黑白打印即可
          </Text>
        </Paragraph>
      </Typography>
    )
  }

  if (step === SettlementStep.Download) {
    const node8 = flowNodes.find((n) => n.nodeNum === '8')
    const downloadUrl = node8?.toJudge || ''

    const handleDownload = () => {
      if (!downloadUrl) {
        message.error('下载链接暂未生成，请稍后重试')
        return
      }
      window.open(downloadUrl, '_blank')
      message.success('开始下载集体户首页')
    }

    return (
      <Typography>
        <Title level={5}>集体户首页下载</Title>
        <Paragraph>恭喜！你的落户流程已全部完成。点击下方按钮下载集体户首页文件。</Paragraph>
        <Paragraph><Text type="secondary">该文件可用于婚姻登记、购房、子女入学等事务。</Text></Paragraph>
        <Button type="primary" icon={<DownloadOutlined />} onClick={handleDownload} block size="large">
          下载集体户首页
        </Button>
      </Typography>
    )
  }

  return null
}
