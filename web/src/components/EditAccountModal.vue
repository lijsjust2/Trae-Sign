<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { api } from '../api'
import { useStore } from '../store'
import type { Account, Settings } from '../types'

const props = defineProps<{ account: Account }>()
const emit = defineEmits<{ close: [], done: [] }>()
const store = useStore()

const remark = ref(props.account.remark)
const pushplusToken = ref(props.account.pushplusToken)
const enabled = ref(props.account.enabled)
const busy = ref(false)
const defaultTime = ref('08:00')

// 从原始 checkinTime 解析出小时和分钟（空则两者都为空 → 使用默认）
const initial = props.account.checkinTime || ''
const hourRef = ref(initial.slice(0, 2))   // '00'-'23' 或 ''
const minRef = ref(initial.slice(3, 5))    // '00'/'15'/'30'/'45' 或 ''

// 同步给后端的 checkinTime：选了小时才有效，分钟未选视为 00
const checkinTime = ref('')
function syncCheckinTime() {
  checkinTime.value = hourRef.value === '' ? '' : hourRef.value + ':' + (minRef.value || '00')
}
watch([hourRef, minRef], syncCheckinTime)
syncCheckinTime()

onMounted(async () => {
  try {
    const s: Settings = await api.getSettings()
    if (s.defaultCheckinTime) defaultTime.value = s.defaultCheckinTime
  } catch { /* ignore */ }
})

// 小时 00-23；分钟 00/15/30/45 四档，足够精细又好选
const hours = Array.from({ length: 24 }, (_, h) => String(h).padStart(2, '0'))
const minutes = ['00', '15', '30', '45']

const effectiveTime = computed(() => checkinTime.value || defaultTime.value)

function useDefault() {
  hourRef.value = ''
  minRef.value = ''
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
            <button v-if="hourRef" type="button" class="clear-btn" @click="useDefault"
              title="清除，使用默认时间">使用默认</button>
          </div>
          <div class="time-pickers">
            <select class="neu-input" v-model="hourRef">
              <option value="">时</option>
              <option v-for="h in hours" :key="h" :value="h">{{ h }}</option>
            </select>
            <span class="sep">:</span>
            <select class="neu-input" v-model="minRef" :disabled="hourRef === ''">
              <option value="">分</option>
              <option v-for="m in minutes" :key="m" :value="m">{{ m }}</option>
            </select>
          </div>
          <div class="hint">当前生效：{{ effectiveTime }}<span v-if="hourRef === ''">（默认）</span></div>
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
.time-pickers { display: flex; align-items: center; gap: 8px; }
.time-pickers .neu-input { flex: 1; min-width: 0; }
.time-pickers .sep { font-weight: 600; color: var(--muted); }
.time-pickers .neu-input:disabled { opacity: .5; cursor: not-allowed; }
</style>
