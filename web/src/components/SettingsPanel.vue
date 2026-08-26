<script setup lang="ts">
import { ref } from 'vue'
import { useStore } from '../store'
import { api } from '../api'

const store = useStore()
const defaultCheckinTime = ref(store.settings.defaultCheckinTime)
const defaultPushplusToken = ref(store.settings.defaultPushplusToken)
const autoCheckin = ref(store.settings.autoCheckin)
const busy = ref(false)

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
      <div class="title">全局设置</div>
      <div class="fields">
        <label class="field">
          <span class="neu-label">默认签到时间（HH:mm）</span>
          <input class="neu-input" type="time" v-model="defaultCheckinTime" />
          <span class="desc">账号未单独设置时使用此时间</span>
        </label>
        <label class="field">
          <span class="neu-label">默认 PushPlus Token</span>
          <input class="neu-input" v-model="defaultPushplusToken" placeholder="一键签到汇总推送用此 token" />
        </label>
        <label class="field switch">
          <span class="neu-label">启用定时自动签到</span>
          <input type="checkbox" v-model="autoCheckin" />
        </label>
      </div>
      <button class="neu-btn neu-btn-primary" :disabled="busy" @click="save">保存设置</button>
    </div>

    <div class="settings neu">
      <div class="title">登录账号</div>
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
.wrap { display: flex; flex-direction: column; gap: 20px; max-width: 480px; }
.settings { padding: 22px; }
.title { font-size: 16px; font-weight: 600; margin-bottom: 18px; }
.fields { display: flex; flex-direction: column; gap: 16px; margin-bottom: 20px; }
.field { display: block; }
.switch { display: flex; align-items: center; gap: 10px; }
.switch .neu-label { margin-bottom: 0; }
.desc { display: block; font-size: 11px; color: var(--faint); margin-top: 4px; }
</style>
