<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { api } from '../api'
import { useStore } from '../store'
import type { Account, Settings } from '../types'

const props = defineProps<{ account: Account }>()
const emit = defineEmits<{ close: [], done: [] }>()
const store = useStore()

const remark = ref(props.account.remark)
const checkinTime = ref(props.account.checkinTime)
const pushplusToken = ref(props.account.pushplusToken)
const enabled = ref(props.account.enabled)
const busy = ref(false)
const defaultTime = ref('08:00')

// 加载默认签到时间，用于"使用默认"提示
onMounted(async () => {
  try {
    const s: Settings = await api.getSettings()
    if (s.defaultCheckinTime) defaultTime.value = s.defaultCheckinTime
  } catch { /* ignore */ }
})

// 生成半小时档位的时间选项，按时段分组
const timeOptions = computed(() => {
  const groups: { label: string; items: { value: string; text: string }[] }[] = [
    { label: '凌晨（00:00 - 05:30）', items: [] },
    { label: '上午（06:00 - 11:30）', items: [] },
    { label: '下午（12:00 - 17:30）', items: [] },
    { label: '晚上（18:00 - 23:30）', items: [] },
  ]
  for (let h = 0; h < 24; h++) {
    for (const m of [0, 30]) {
      const value = `${String(h).padStart(2, '0')}:${String(m).padStart(2, '0')}`
      const text = value
      const groupIdx = Math.floor(h / 6)
      groups[groupIdx].items.push({ value, text })
    }
  }
  return groups
})

const effectiveTime = computed(() => checkinTime.value || defaultTime.value)

function clearCheckinTime() {
  checkinTime.value = ''
}

async function save() {
  busy.value = true
  try {
    await api.updateAccount(props.account.id, {
      remark: remark.value,
      checkinTime: checkinTime.value,
      pushplusToken: pushplusToken.value,
      enabled: enabled.value
    })
    store.showToast('已保存')
    emit('done')
  } catch (e: any) { store.showToast(e.message) }
  finally { busy.value = false }
}
</script>

<template>
  <div class="mask" @click.self="emit('close')">
    <div class="modal neu">
      <div class="modal-head">
        <span class="title">编辑账号</span>
        <button class="neu-btn neu-btn-sm" @click="emit('close')">关闭</button>
      </div>
      <div class="fields">
        <label class="field">
          <span class="neu-label">备注名</span>
          <input class="neu-input" v-model="remark" />
        </label>
        <div class="field">
          <div class="time-row">
            <span class="neu-label">签到时间</span>
            <button v-if="checkinTime" type="button" class="clear-btn" @click="clearCheckinTime"
              title="清除，使用默认时间">使用默认</button>
          </div>
          <select class="neu-input" v-model="checkinTime">
            <option value="">使用默认（{{ defaultTime }}）</option>
            <optgroup v-for="g in timeOptions" :key="g.label" :label="g.label">
              <option v-for="it in g.items" :key="it.value" :value="it.value">{{ it.text }}</option>
            </optgroup>
          </select>
          <div class="hint">当前生效：{{ effectiveTime }}</div>
        </div>
        <label class="field">
          <span class="neu-label">PushPlus Token</span>
          <input class="neu-input" v-model="pushplusToken" placeholder="留空则用设置里的默认" />
        </label>
        <label class="field switch">
          <span class="neu-label">启用签到</span>
          <input type="checkbox" v-model="enabled" />
        </label>
      </div>
      <div class="info">
        昵称：{{ account.nickname || '—' }} ｜ 累计积分：{{ account.totalCredits }}
      </div>
      <button class="neu-btn neu-btn-primary" :disabled="busy" @click="save">保存</button>
    </div>
  </div>
</template>

<style scoped>
.mask { position: fixed; inset: 0; background: rgba(45, 55, 72, .35); display: flex; align-items: center; justify-content: center; z-index: 50; padding: 20px; }
.modal { width: 100%; max-width: 460px; padding: 20px 22px; }
.modal-head { display: flex; justify-content: space-between; align-items: center; margin-bottom: 16px; }
.title { font-size: 16px; font-weight: 600; }
.fields { display: flex; flex-direction: column; gap: 12px; margin-bottom: 16px; }
.field { display: block; }
.switch { display: flex; align-items: center; gap: 10px; }
.switch .neu-label { margin-bottom: 0; }
.info { font-size: 12px; color: var(--muted); margin-bottom: 16px; }
.time-row { display: flex; justify-content: space-between; align-items: center; }
.clear-btn { background: none; border: none; color: var(--accent, #4a90d9); font-size: 12px; cursor: pointer; padding: 0; }
.clear-btn:hover { text-decoration: underline; }
.hint { font-size: 12px; color: var(--muted); margin-top: 4px; }
</style>
