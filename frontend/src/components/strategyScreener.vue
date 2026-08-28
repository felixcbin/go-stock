<script setup lang="ts">
import {h, onBeforeMount, ref, computed} from 'vue'
import {
  RunWaterStrategyScreen,
  Follow,
  GetConfig,
  GetGroupList,
  AddStockGroup,
  AddGroup,
  OpenURL,
} from "../../wailsjs/go/main/App"
import {useMessage, NText, NTag, NButton, NDropdown, NIcon} from 'naive-ui'
import {Environment} from "../../wailsjs/runtime"
import {AddOutline, FolderOpenOutline} from "@vicons/ionicons5"
import StockLightweightKlineChart from "./StockLightweightKlineChart.vue"

const message = useMessage()
const columns = ref<any[]>([])
const dataList = ref<any[]>([])
const traceInfo = ref('')
const tableScrollX = ref(2800)
const darkTheme = ref(false)
const loading = ref(false)
const strategyMeta = ref<any>(null)
const klineModalShow = ref(false)
const klineStockCode = ref('')
const klineStockName = ref('')

const criteria = [
  {label: '涨幅', value: '3% – 5%'},
  {label: '涨停基因', value: '20 天内有过涨停'},
  {label: '量比', value: '> 1.4'},
  {label: '市值', value: '< 200 亿'},
  {label: '换手率', value: '5% – 10%'},
  {label: '分时形态', value: '全天在均价线上方（水上）'},
]

const groupList = ref<any[]>([])
const showFollowGroupModal = ref(false)
const newGroupName = ref('')
const pendingFollowRow = ref<any>(null)

const followGroupOptions = computed(() => {
  const opts = [{label: '默认（不分组）', key: 0}]
  groupList.value.forEach(g => opts.push({label: g.name, key: g.ID}))
  opts.push({type: 'divider', key: 'divider'})
  opts.push({label: '新建分组', key: 'new', icon: () => h(NIcon, null, {default: () => h(AddOutline)})})
  return opts
})

const paginationProps = computed(() => ({
  pageSize: 10,
  prefix: ({itemCount}: { itemCount: number }) => h('span', {style: 'margin-right: 8px'}, [
    '共找到 ',
    h(NTag, {type: 'info', bordered: false, size: 'small'}, {default: () => itemCount}),
    ' 只股',
  ]),
}))

function loadGroupList() {
  GetGroupList().then(res => {
    groupList.value = res || []
  }).catch(() => {})
}

function handleFollowSelect(key: string | number, row: any) {
  if (key === 'new') {
    pendingFollowRow.value = row
    newGroupName.value = ''
    showFollowGroupModal.value = true
    return
  }
  doFollow(row, Number(key))
}

function doFollow(row: any, groupId: number) {
  const code = row.MARKET_SHORT_NAME.toLowerCase() + row.SECURITY_CODE
  Follow(code).then(result => {
    if (result === '关注成功') {
      message.success(groupId > 0 ? `已关注，并加入分组「${groupNameById(groupId)}」` : '关注成功')
    } else {
      message.info(result)
    }
    if (groupId > 0) {
      AddStockGroup(groupId, code).then(() => loadGroupList()).catch(() => {})
    }
  }).catch(err => {
    message.error('关注失败: ' + err)
  })
}

function groupNameById(id: number) {
  const g = groupList.value.find(item => item.ID === id)
  return g ? g.name : ''
}

function handleCreateGroupAndFollow() {
  const name = newGroupName.value.trim()
  if (!name) {
    message.warning('请输入分组名称')
    return
  }
  const maxSort = groupList.value.reduce((max, g) => Math.max(max, g.sort || 0), 0)
  AddGroup({name, sort: maxSort + 1}).then(res => {
    if (res === '添加成功') {
      GetGroupList().then(list => {
        groupList.value = list || []
        showFollowGroupModal.value = false
        const created = groupList.value.find(g => g.name === name)
        if (created && pendingFollowRow.value) {
          doFollow(pendingFollowRow.value, created.ID)
        }
        pendingFollowRow.value = null
      })
    } else {
      message.error(res)
    }
  }).catch(err => {
    message.error('新建分组失败: ' + err)
  })
}

function calculateTableWidth(cols: any[]) {
  let totalWidth = 0
  cols.forEach(col => {
    if (col.children?.length > 0) {
      let childrenWidth = 0
      col.children.forEach((child: any) => {
        childrenWidth += child.width || child.minWidth || 100
      })
      totalWidth += Math.max(col.width || col.minWidth || 200, childrenWidth)
    } else {
      totalWidth += col.width || col.minWidth || 120
    }
  })
  return Math.max(totalWidth + 100, 1200)
}

function isNumeric(value: any) {
  return !isNaN(parseFloat(value)) && isFinite(value)
}

function toEastMoneyCode(stockCode: string, marketShortName: string) {
  const m = (marketShortName || '').toUpperCase()
  if (m === 'SH' || m === 'SZ' || m === 'BJ') return stockCode + '.' + m
  if (m === 'HK') return stockCode + '.HK'
  if (m === 'US') return stockCode + '.US'
  if (/^(6|5)/.test(stockCode)) return stockCode + '.SH'
  if (/^(8|4)/.test(stockCode)) return stockCode + '.BJ'
  if (/^[a-zA-Z]+$/.test(stockCode)) return stockCode.toUpperCase() + '.US'
  return stockCode + '.SZ'
}

function showStockKline(row: any) {
  const stockCode = row.SECURITY_CODE
  const stockName = row.SECURITY_SHORT_NAME
  const em = toEastMoneyCode(stockCode, row.MARKET_SHORT_NAME)
  if (!em) {
    message.warning('当前代码暂不支持K线图')
    return
  }
  klineStockCode.value = em
  klineStockName.value = stockName || ''
  klineModalShow.value = true
}

function openCenteredWindow(url: string, width: number, height: number) {
  const left = (window.screen.width - width) / 2
  const top = (window.screen.height - height) / 2
  Environment().then(env => {
    switch (env.platform) {
      case 'windows':
        window.open(
          url,
          'centeredWindow',
          `width=${width},height=${height},left=${left},top=${top},location=no,menubar=no,toolbar=no,display=standalone`
        )
        break
      default:
        OpenURL(url)
    }
  })
}

function buildColumns(rawColumns: any[]) {
  const built = rawColumns
    .filter(item => !item.hiddenNeed && item.title !== '市场码' && item.title !== '市场简称')
    .map(item => {
      if (item.children) {
        return {
          title: item.title + (item.unit ? `[${item.unit}]` : ''),
          key: item.key,
          resizable: true,
          minWidth: 200,
          ellipsis: {tooltip: true},
          children: item.children.filter((child: any) => !child.hiddenNeed).map((child: any) => ({
            title: child.dateMsg,
            key: child.key,
            minWidth: 100,
            resizable: true,
            ellipsis: {tooltip: true},
            sorter: (row1: any, row2: any) => {
              if (isNumeric(row1[child.key]) && isNumeric(row2[child.key])) {
                return row1[child.key] - row2[child.key]
              }
              return 0
            },
          })),
        }
      }
      return {
        title: item.title + (item.unit ? `[${item.unit}]` : ''),
        key: item.key,
        resizable: true,
        minWidth: 120,
        ellipsis: {tooltip: true},
        sorter: (row1: any, row2: any) => {
          if (isNumeric(row1[item.key]) && isNumeric(row2[item.key])) {
            return row1[item.key] - row2[item.key]
          }
          return 0
        },
      }
    })

  built.push({
    title: '操作',
    key: 'actions',
    width: 170,
    fixed: 'right',
    render: (row: any) => h('div', {style: 'display:flex;gap:4px;align-items:center;'}, [
      h(NButton, {
        size: 'tiny',
        type: 'info',
        onClick: () => showStockKline(row),
      }, {default: () => 'K线'}),
      h(NDropdown, {
        trigger: 'click',
        options: followGroupOptions.value,
        placement: 'bottom-end',
        menuProps: () => ({style: 'max-height:300px; overflow-y:auto;'}),
        onSelect: (key: string | number) => handleFollowSelect(key, row),
      }, {
        default: () => h(NButton, {
          strong: true,
          tertiary: true,
          size: 'small',
          type: 'warning',
          style: 'font-size: 14px; padding: 0 10px;',
        }, {
          default: () => '关注',
          icon: () => h(NIcon, null, {default: () => h(FolderOpenOutline, {size: 14})}),
        }),
      }),
    ]),
  })
  return built
}

function runScreen() {
  loading.value = true
  const loadingMsg = message.loading('正在执行水上策略选股，请稍候...', {duration: 0})
  RunWaterStrategyScreen().then(res => {
    loadingMsg.destroy()
    loading.value = false
    if (res.code === 100) {
      traceInfo.value = res.data?.traceInfo?.showText || ''
      strategyMeta.value = res.data?.strategyMeta || null
      columns.value = buildColumns(res.data.result.columns || [])
      dataList.value = res.data.result.dataList || []
      tableScrollX.value = calculateTableWidth(columns.value)
      const pre = strategyMeta.value?.preFilterCount ?? dataList.value.length
      const post = strategyMeta.value?.postFilterCount ?? dataList.value.length
      if (pre > post) {
        message.info(`指标筛选 ${pre} 只，分时水上过滤后 ${post} 只`)
      }
    } else {
      message.error(res.msg || res.message || '选股失败')
      columns.value = []
      dataList.value = []
    }
  }).catch(err => {
    loadingMsg.destroy()
    loading.value = false
    message.error(String(err))
  })
}

onBeforeMount(() => {
  GetConfig().then(result => {
    if (result.darkTheme) darkTheme.value = true
  })
  loadGroupList()
})
</script>

<template>
  <div style="--wails-draggable:no-drag">
    <n-card size="small" title="水上策略选股" style="margin-bottom: 12px;">
      <n-space vertical size="small">
        <n-text depth="3">
          固定六维条件联合筛选：先通过东财指标选股，再验证分时全天运行在均价线上方。
        </n-text>
        <n-space wrap>
          <n-tag v-for="item in criteria" :key="item.label" type="info" size="small" :bordered="false">
            {{ item.label }}：{{ item.value }}
          </n-tag>
        </n-space>
        <n-flex align="center" :size="12">
          <n-button type="primary" :loading="loading" @click="runScreen">开始选股</n-button>
          <n-text v-if="strategyMeta" depth="3">
            指标筛选 {{ strategyMeta.preFilterCount }} 只 → 分时水上 {{ strategyMeta.postFilterCount }} 只
          </n-text>
        </n-flex>
      </n-space>
    </n-card>

    <div v-if="traceInfo" style="margin-bottom: 8px;">
      <n-ellipsis line-clamp="1" :tooltip="true">
        <n-text type="info">东财解析条件：</n-text>
        <n-text type="warning">{{ traceInfo }}</n-text>
        <template #tooltip>
          <div style="max-width: 580px; text-align: center;">
            <n-text type="warning">{{ traceInfo }}</n-text>
          </div>
        </template>
      </n-ellipsis>
    </div>

    <n-data-table
      :striped="true"
      flex-height
      size="small"
      :columns="columns"
      :data="dataList"
      :pagination="paginationProps"
      :scroll-x="tableScrollX"
      style="height: calc(100vh - 320px)"
      :render-cell="(value, rowData, column) => {
        if (column.key === 'SECURITY_CODE' || column.key === 'SERIAL') {
          return h(NText, {type: 'info', border: false}, {default: () => `${value}`})
        }
        if (isNumeric(value)) {
          let type = 'info'
          if (Number(value) < 0) type = 'success'
          else if (Number(value) >= 0 && Number(value) <= 5) type = 'warning'
          else if (Number(value) > 5) type = 'error'
          return h(NText, {type}, {default: () => `${value}`})
        }
        if (column.key === 'SECURITY_SHORT_NAME') {
          return h(NText, {
            type: 'info',
            bordered: false,
            size: 'small',
            onClick: () => openCenteredWindow(`https://quote.eastmoney.com/${rowData.MARKET_SHORT_NAME}${rowData.SECURITY_CODE}.html#fullScreenChart`, 1240, 700),
          }, {default: () => `${value}`})
        }
        return h(NText, {type: 'info'}, {default: () => `${value}`})
      }"
    />

    <n-modal
      v-model:show="klineModalShow"
      :title="(klineStockName || '') + ' - ' + klineStockCode + ' K线图'"
      preset="card"
      style="width: 1400px; max-width: calc(100vw - 32px);"
      :mask-closable="true"
    >
      <StockLightweightKlineChart
        v-if="klineModalShow && klineStockCode"
        :key="klineStockCode"
        :code="klineStockCode"
        :stock-name="klineStockName"
        :dark-theme="darkTheme"
        :chart-height="460"
      />
    </n-modal>

    <n-modal
      v-model:show="showFollowGroupModal"
      preset="dialog"
      title="新建分组"
      positive-text="创建并关注"
      negative-text="取消"
      @positive-click="handleCreateGroupAndFollow"
      style="width: 420px;"
    >
      <n-form label-placement="left" label-width="80">
        <n-form-item label="分组名称">
          <n-input v-model:value="newGroupName" placeholder="请输入分组名称" @keyup.enter="handleCreateGroupAndFollow"/>
        </n-form-item>
      </n-form>
    </n-modal>
  </div>
</template>
