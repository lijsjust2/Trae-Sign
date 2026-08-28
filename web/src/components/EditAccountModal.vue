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
    <div class="modal">
      <div class="banner"></div>
      <div class="pad">
        <div class="modal-head">
          <span class="title">编辑账号</span>
          <button class="close-btn" @click="emit('close')" title="关闭">✕</button>
        </div>
        <div class="fields">
          <label class="field">
            <span class="neu-label">备注名</span>
            <input class="neu-input" v-model="remark" placeholder="给账号起个好记的名字" />
          </label>
          <div class="field">
            <div class="time-row">
              <span class="neu-label">签到时间</span>
              <button v-if="hourRef" type="button" class="clear-btn" @click="useDefault">使用默认</button>
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
            <div class="hint">当前生效：<b>{{ effectiveTime }}</b><span v-if="hourRef === ''">（默认）</span></div>
          </div>
          <label class="field">
            <span class="neu-label">PushPlus Token</span>
            <input class="neu-input" v-model="pushplusToken" placeholder="留空 → 使用全局默认" />
          </label>
          <label class="field switch">
            <span class="neu-label">启用签到</span>
            <input type="checkbox" v-model="enabled" />
          </label>
        </div>
        <div class="info neu-inset">
          昵称：{{ account.nickname || '—' }}　｜　累计积分：{{ account.totalCredits }}
        </div>
        <button class="neu-btn neu-btn-primary save-btn" :disabled="busy" @click="save">保存修改</button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.mask {
  position: fixed; inset: 0; z-index: 50; padding: 20px;
  display: flex; align-items: center; justify-content: center;
  background: rgba(26,33,64,.35);
  backdrop-filter: blur(6px);
  -webkit-backdrop-filter: blur(6px);
}
.modal {
  width: 100%; max-width: 480px;
  background: #fff;
  border-radius: 14px; overflow: hidden;
  border: 1px solid var(--border);
  box-shadow: 0 30px 60px rgba(26,33,64,.22), 0 8px 20px rgba(26,33,64,.08);
  animation: pop .18s ease-out both;
}
@keyframes pop {
  from { opacity: 0; transform: translateY(8px) scale(.985); }
  to   { opacity: 1; transform: none; }
}
.banner {
  height: 5px;
  background: linear-gradient(90deg, var(--primary), #7b5dff 55%, var(--info));
}
.pad { padding: 22px 24px 24px; }
.modal-head { display: flex; justify-content: space-between; align-items: center; margin-bottom: 20px; }
.title { font-size: 18px; font-weight: 700; color: var(--text); letter-spacing: .1px; }
.close-btn {
  width: 30px; height: 30px; border-radius: 8px;
  background: transparent; color: var(--muted);
  font-size: 14px; line-height: 1;
  transition: all .15s ease;
}
.close-btn:hover { background: var(--danger-bg); color: var(--danger-fg); }

.fields { display: flex; flex-direction: column; gap: 16px; margin-bottom: 18px; }
.field { display: block; }
.switch { display: flex; align-items: center; gap: 10px; }
.switch .neu-label { margin-bottom: 0; }

.info {
  font-size: 12.5px; color: var(--text-soft);
  padding: 10px 14px; margin-bottom: 18px;
  text-align: center;
}
.info b { color: var(--primary); font-weight: 700; }

.time-row { display: flex; justify-content: space-between; align-items: center; }
.clear-btn {
  background: none; border: none; color: var(--primary);
  font-size: 12px; font-weight: 600; cursor: pointer; padding: 0;
}
.clear-btn:hover { text-decoration: underline; }
.hint { font-size: 12px; color: var(--muted); margin-top: 6px; }
.time-pickers { display: flex; align-items: center; gap: 8px; max-width: 260px; }
.time-pickers .neu-input { flex: 1; min-width: 0; }
.time-pickers .sep { font-weight: 700; color: var(--muted); font-size: 16px; margin: 0 2px; }
.time-pickers .neu-input:disabled { opacity: .55; cursor: not-allowed; }

.save-btn { width: 100%; padding: 10px 16px; font-size: 14px; font-weight: 600; }
</style>
