<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { api } from '../api'

const emit = defineEmits<{ authed: [] }>()

const mode = ref<'loading' | 'setup' | 'login'>('loading')
const username = ref('')
const password = ref('')
const password2 = ref('')
const busy = ref(false)
const error = ref('')

onMounted(async () => {
  try {
    const s = await api.authStatus()
    if (s.loggedIn) { emit('authed'); return }
    mode.value = s.initialized ? 'login' : 'setup'
  } catch (e: any) {
    error.value = e.message
    mode.value = 'login'
  }
})

async function submit() {
  error.value = ''
  if (mode.value === 'setup' && password.value !== password2.value) {
    error.value = '两次输入的密码不一致'
    return
  }
  busy.value = true
  try {
    if (mode.value === 'setup') {
      await api.authSetup(username.value.trim(), password.value)
    } else {
      await api.authLogin(username.value.trim(), password.value)
    }
    emit('authed')
  } catch (e: any) { error.value = e.message }
  finally { busy.value = false }
}
</script>

<template>
  <div class="gate">
    <div class="card neu">
      <div class="brand">TRAE 签到</div>
      <div class="subtitle">
        {{ mode === 'setup' ? '初次使用，请设置管理账号密码' : '请登录以管理签到账号' }}
      </div>

      <div v-if="mode === 'loading'" class="loading">加载中…</div>
      <template v-else>
        <label class="field">
          <span class="neu-label">账号</span>
          <input class="neu-input" v-model="username" placeholder="管理账号名"
            @keyup.enter="submit" autocomplete="username" />
        </label>
        <label class="field">
          <span class="neu-label">密码</span>
          <input class="neu-input" type="password" v-model="password" placeholder="至少 6 位"
            @keyup.enter="submit" :autocomplete="mode === 'setup' ? 'new-password' : 'current-password'" />
        </label>
        <label v-if="mode === 'setup'" class="field">
          <span class="neu-label">确认密码</span>
          <input class="neu-input" type="password" v-model="password2" placeholder="再输入一次"
            @keyup.enter="submit" autocomplete="new-password" />
        </label>
        <div v-if="error" class="error">{{ error }}</div>
        <button class="neu-btn neu-btn-primary" :disabled="busy" @click="submit">
          {{ mode === 'setup' ? '完成设置' : '登录' }}
        </button>
      </template>
    </div>
  </div>
</template>

<style scoped>
.gate { min-height: 100vh; display: flex; align-items: center; justify-content: center; padding: 20px; }
.card { width: 100%; max-width: 380px; padding: 32px 28px; display: flex; flex-direction: column; gap: 14px; }
.brand { font-size: 20px; font-weight: 700; color: var(--brand); text-align: center; }
.subtitle { font-size: 13px; color: var(--muted); text-align: center; margin-bottom: 6px; }
.field { display: block; }
.loading { text-align: center; color: var(--muted); padding: 20px 0; }
.error { font-size: 12px; color: #e5484d; }
</style>
