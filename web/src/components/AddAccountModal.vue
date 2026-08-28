<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { api } from '../api'
import { useStore } from '../store'
import type { Settings } from '../types'

const emit = defineEmits<{ close: [], done: [] }>()
const store = useStore()

const remark = ref('')
const pushplusToken = ref('')

const defaultTime = ref('08:00')
const hourRef = ref('')
const minRef = ref('')
const checkinTime = ref('')
function syncCheckinTime() {
  checkinTime.value = hourRef.value === '' ? '' : hourRef.value + ':' + (minRef.value || '00')
}
watch([hourRef, minRef], syncCheckinTime)
const effectiveTime = computed(() => checkinTime.value || defaultTime.value)
function useDefault() { hourRef.value = ''; minRef.value = '' }

const hours = Array.from({ length: 24 }, (_, h) => String(h).padStart(2, '0'))
const minutes = ['00', '15', '30', '45']

onMounted(async () => {
  try {
    const s: Settings = await api.getSettings()
    if (s.defaultCheckinTime) defaultTime.value = s.defaultCheckinTime
  } catch { /* ignore */ }
})

const loginUrl = ref('')
const machineId = ref('')
const deviceId = ref('')
const privateKeyPem = ref('')
const publicKeyPem = ref('')
const callbackUrl = ref('')
const busy = ref(false)

async function genUrl() {
  busy.value = true
  try {
    const r = await api.loginURL(remark.value, checkinTime.value, pushplusToken.value)
    loginUrl.value = r.loginUrl
    machineId.value = r.machineId
    deviceId.value = r.deviceId
    privateKeyPem.value = r.privateKeyPem
    publicKeyPem.value = r.publicKeyPem
    store.showToast('登录链接已生成')
  } catch (e: any) { store.showToast(e.message) }
  finally { busy.value = false }
}

async function submitOAuth() {
  if (!callbackUrl.value.trim()) { store.showToast('请粘贴回调链接'); return }
  busy.value = true
  try {
    await api.loginCallback(callbackUrl.value, machineId.value, deviceId.value, privateKeyPem.value, publicKeyPem.value, remark.value, checkinTime.value, pushplusToken.value)
    store.showToast('登录成功，账号已添加')
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
          <span class="title">添加账号</span>
          <button class="close-btn" @click="emit('close')" title="关闭">✕</button>
        </div>

        <div class="fields">
          <label class="field">
            <span class="neu-label">备注名（可选）</span>
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
            <span class="neu-label">PushPlus Token（可选）</span>
            <input class="neu-input" v-model="pushplusToken" placeholder="留空 → 使用全局默认" />
          </label>
        </div>

        <button class="neu-btn neu-btn-primary gen-btn" :disabled="busy" @click="genUrl">
          {{ loginUrl ? '重新生成登录链接' : '生成登录链接' }}
        </button>

        <div v-if="loginUrl" class="login-pane neu-inset">
          <div class="step">
            <div class="step-num">1</div>
            <div class="step-body">
              <div class="step-title">浏览器打开登录链接</div>
              <a :href="loginUrl" target="_blank" class="login-link">{{ loginUrl.length > 72 ? loginUrl.slice(0, 72) + '…' : loginUrl }}</a>
            </div>
          </div>
          <div class="step">
            <div class="step-num">2</div>
            <div class="step-body">
              <div class="step-title">登录 TRAE 账号</div>
              <div class="step-desc">浏览器会跳转到 <code>127.0.0.1:18080/authorize?…</code>（页面打不开没关系）</div>
            </div>
          </div>
          <div class="step step-3">
            <div class="step-num">3</div>
            <div class="step-body">
              <div class="step-title">粘贴回调链接并完成</div>
              <label class="field">
                <span class="neu-label">回调链接</span>
                <textarea class="neu-textarea" v-model="callbackUrl" rows="3"
                  placeholder="http://127.0.0.1:18080/authorize?refreshToken=…"></textarea>
              </label>
              <button class="neu-btn neu-btn-primary done-btn" :disabled="busy" @click="submitOAuth">完成登录</button>
            </div>
          </div>
        </div>
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
  width: 100%; max-width: 580px; max-height: 88vh; overflow-y: auto;
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

.gen-btn { width: 100%; padding: 10px 16px; font-size: 14px; font-weight: 600; margin-bottom: 18px; }

.login-pane {
  padding: 16px 18px;
  display: flex; flex-direction: column; gap: 14px;
}
.step { display: flex; gap: 12px; }
.step-num {
  flex-shrink: 0; width: 24px; height: 24px; border-radius: 8px;
  background: linear-gradient(135deg, var(--primary), var(--primary-tint));
  color: #fff; font-size: 12px; font-weight: 700; line-height: 24px; text-align: center;
  box-shadow: 0 2px 6px rgba(94,114,228,.28);
}
.step-body { flex: 1; min-width: 0; display: flex; flex-direction: column; gap: 4px; }
.step-title { font-size: 13.5px; font-weight: 600; color: var(--text); }
.step-desc { font-size: 12px; color: var(--text-soft); line-height: 1.6; }
.login-link {
  word-break: break-all; font-size: 12.5px;
  padding: 6px 10px; background: #fff; border: 1px solid var(--border);
  border-radius: 8px; color: var(--primary); font-weight: 500;
}
.step-3 { margin-top: 2px; padding-top: 14px; border-top: 1px dashed #dbe0fc; }
.step-3 .field { margin-top: 4px; }
.done-btn { margin-top: 12px; padding: 9px 16px; font-weight: 600; }
code {
  font-family: ui-monospace, 'SF Mono', Menlo, Consolas, monospace;
  background: #fff; padding: 1px 6px; border-radius: 5px;
  border: 1px solid var(--border);
  font-size: 11.5px; color: var(--primary-tint);
}
</style>
