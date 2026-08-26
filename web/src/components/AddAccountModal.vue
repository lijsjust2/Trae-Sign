<script setup lang="ts">
import { ref } from 'vue'
import { api } from '../api'
import { useStore } from '../store'

const emit = defineEmits<{ close: [], done: [] }>()
const store = useStore()

const remark = ref('')
const checkinTime = ref('')
const pushplusToken = ref('')

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
    <div class="modal neu">
      <div class="modal-head">
        <span class="title">添加账号</span>
        <button class="neu-btn neu-btn-sm" @click="emit('close')">关闭</button>
      </div>

      <div class="fields">
        <label class="field">
          <span class="neu-label">备注名（可选）</span>
          <input class="neu-input" v-model="remark" placeholder="给账号起个好记的名字" />
        </label>
        <label class="field">
          <span class="neu-label">签到时间（HH:mm）</span>
          <input class="neu-input" type="time" v-model="checkinTime" />
        </label>
        <label class="field">
          <span class="neu-label">PushPlus Token（可选）</span>
          <input class="neu-input" v-model="pushplusToken" placeholder="留空则用设置里的默认" />
        </label>
      </div>

      <div class="pane">
        <button class="neu-btn neu-btn-primary" :disabled="busy" @click="genUrl">生成登录链接</button>
        <div v-if="loginUrl" class="login-box neu-inset">
          <p class="hint">1. 点击下方链接，用浏览器登录 TRAE 账号</p>
          <a :href="loginUrl" target="_blank" class="login-link">{{ loginUrl.slice(0, 60) }}…</a>
          <p class="hint">2. 登录后浏览器会跳转到 <code>127.0.0.1:18080/authorize?…</code>（页面打不开没关系）</p>
          <p class="hint">3. 复制浏览器地址栏的完整链接，粘贴到下方</p>
          <label class="field">
            <span class="neu-label">回调链接</span>
            <textarea class="neu-textarea" v-model="callbackUrl" rows="3"
              placeholder="http://127.0.0.1:18080/authorize?refreshToken=…"></textarea>
          </label>
          <button class="neu-btn neu-btn-primary" :disabled="busy" @click="submitOAuth">完成登录</button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.mask { position: fixed; inset: 0; background: rgba(45, 55, 72, .35); display: flex; align-items: center; justify-content: center; z-index: 50; padding: 20px; }
.modal { width: 100%; max-width: 560px; max-height: 88vh; overflow-y: auto; padding: 20px 22px; }
.modal-head { display: flex; justify-content: space-between; align-items: center; margin-bottom: 16px; }
.title { font-size: 16px; font-weight: 600; }
.fields { display: flex; flex-direction: column; gap: 12px; margin-bottom: 16px; }
.field { display: block; }
.pane { display: flex; flex-direction: column; gap: 12px; }
.login-box { padding: 14px; display: flex; flex-direction: column; gap: 8px; }
.hint { font-size: 12px; color: var(--muted); }
.login-link { word-break: break-all; font-size: 12px; }
code { font-family: ui-monospace, monospace; }
</style>
