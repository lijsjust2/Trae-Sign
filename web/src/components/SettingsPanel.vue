<script setup lang="ts">
import { ref, watch } from 'vue'
import { useStore } from '../store'
import { api } from '../api'

const store = useStore()
const defaultPushplusToken = ref(store.settings.defaultPushplusToken)
const autoCheckin = ref(store.settings.autoCheckin)
const busy = ref(false)

// 默认签到时间拆成小时+分钟两个下拉，与编辑账号弹窗交互一致
const initial = store.settings.defaultCheckinTime || '08:00'
const defHour = ref(initial.slice(0, 2))
const defMin = ref(initial.slice(3, 5))
const defaultCheckinTime = ref('')
function syncDefault() {
  defaultCheckinTime.value = defHour.value + ':' + (defMin.value || '00')
}
watch([defHour, defMin], syncDefault)
syncDefault()

const hours = Array.from({ length: 24 }, (_, h) => String(h).padStart(2, '0'))
const minutes = ['00', '15', '30', '45']

async function save() {
  busy.value = true
  try {
    await api.saveSettings({
      defaultCheckinTime: defaultCheckinTime.value,
      defaultPushplusToken: defaultPushplusToken.value,
      autoCheckin: autoCheckin.value
    })
    await store.loadSettings()
    store.showToast('设置已保存')
  } catch (e: any) { store.showToast(e.message) }
  finally { busy.value = false }
}

// 测试推送：发送一条预览消息到默认 token（用输入框当前值，未保存也能测）
const testBusy = ref(false)
async function testPush() {
  if (!defaultPushplusToken.value.trim()) { store.showToast('请先填写默认 PushPlus Token'); return }
  testBusy.value = true
  try {
    await api.testPush(defaultPushplusToken.value.trim())
    store.showToast('测试推送已发送，请查看微信')
  } catch (e: any) { store.showToast(e.message) }
  finally { testBusy.value = false }
}

// ===== 登录账号密码 =====
const authUsername = ref('')
const oldPassword = ref('')
const newPassword = ref('')
const authBusy = ref(false)

async function changeAuth() {
  if (!authUsername.value.trim()) { store.showToast('请输入新账号名'); return }
  if (!newPassword.value) { store.showToast('请输入新密码'); return }
  authBusy.value = true
  try {
    await api.authChange(oldPassword.value, authUsername.value.trim(), newPassword.value)
    store.showToast('登录账号密码已修改')
    oldPassword.value = ''
    newPassword.value = ''
  } catch (e: any) { store.showToast(e.message) }
  finally { authBusy.value = false }
}
</script>

<template>
  <div class="wrap">
    <div class="settings neu">
      <div class="head">
        <div class="title">全局设置</div>
        <span class="tag tag-primary">通用</span>
      </div>
      <div class="fields">
        <div class="field">
          <span class="neu-label">默认签到时间</span>
          <div class="time-pickers">
            <select class="neu-input" v-model="defHour">
              <option v-for="h in hours" :key="h" :value="h">{{ h }}</option>
            </select>
            <span class="sep">:</span>
            <select class="neu-input" v-model="defMin">
              <option v-for="m in minutes" :key="m" :value="m">{{ m }}</option>
            </select>
          </div>
          <span class="desc">账号未单独设置时使用此时间 · 当前生效 {{ defaultCheckinTime }}</span>
        </div>
        <label class="field">
          <span class="neu-label">默认 PushPlus Token</span>
          <input class="neu-input" v-model="defaultPushplusToken" placeholder="一键签到汇总推送用此 token" />
        </label>
        <label class="field switch">
          <span class="neu-label">启用定时自动签到</span>
          <input type="checkbox" v-model="autoCheckin" />
        </label>
      </div>
      <div class="btn-row">
        <button class="neu-btn neu-btn-primary" :disabled="busy" @click="save">保存设置</button>
        <button class="neu-btn" :disabled="testBusy" @click="testPush">{{ testBusy ? '推送中…' : '测试推送' }}</button>
      </div>
    </div>

    <div class="settings neu">
      <div class="head">
        <div class="title">登录账号</div>
        <span class="tag tag-warn">安全</span>
      </div>
      <div class="fields">
        <label class="field">
          <span class="neu-label">新账号名</span>
          <input class="neu-input" v-model="authUsername" placeholder="登录用的账号名" />
        </label>
        <label class="field">
          <span class="neu-label">旧密码</span>
          <input class="neu-input" type="password" v-model="oldPassword" placeholder="验证身份用" />
        </label>
        <label class="field">
          <span class="neu-label">新密码</span>
          <input class="neu-input" type="password" v-model="newPassword" placeholder="至少 6 位" />
        </label>
      </div>
      <button class="neu-btn neu-btn-primary" :disabled="authBusy" @click="changeAuth">修改登录账号密码</button>
    </div>
  </div>
</template>

<style scoped>
.wrap { display: flex; flex-direction: column; gap: 22px; max-width: 560px; }
.settings { padding: 22px 24px; }
.head { display: flex; align-items: center; justify-content: space-between; margin-bottom: 22px; }
.title { font-size: 17px; font-weight: 700; color: var(--text); letter-spacing: .1px; }
.fields { display: flex; flex-direction: column; gap: 18px; margin-bottom: 22px; }
.field { display: block; }
.switch { display: flex; align-items: center; gap: 10px; }
.switch .neu-label { margin-bottom: 0; }
.desc { display: block; font-size: 12px; color: var(--muted); margin-top: 6px; }
.time-pickers { display: flex; align-items: center; gap: 8px; max-width: 260px; }
.time-pickers .neu-input { flex: 1; min-width: 0; }
.time-pickers .sep { font-weight: 700; color: var(--muted); font-size: 16px; margin: 0 4px; }
.btn-row { display: flex; gap: 10px; }
.btn-row .neu-btn { flex: 1; }
</style>
