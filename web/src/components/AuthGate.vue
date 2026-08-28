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
      <div class="banner"></div>
      <div class="pad">
        <div class="brand-line">
          <div class="logo">T</div>
          <div class="brand">TRAE 签到</div>
        </div>
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
          <button class="neu-btn neu-btn-primary submit" :disabled="busy" @click="submit">
            {{ mode === 'setup' ? '完成设置' : '登录' }}
          </button>
        </template>
      </div>
    </div>
  </div>
</template>

<style scoped>
.gate {
  min-height: 100vh; display: flex; align-items: center; justify-content: center; padding: 20px;
  background:
    radial-gradient(1200px 700px at 0% 0%, rgba(94,114,228,.12), transparent 60%),
    radial-gradient(1000px 600px at 100% 100%, rgba(17,205,239,.1), transparent 55%),
    linear-gradient(180deg, #eef1f7, #f5f7fa);
}
.card {
  width: 100%; max-width: 400px;
  overflow: hidden;
  box-shadow: 0 30px 60px rgba(30,40,90,.10), 0 10px 24px rgba(30,40,90,.06);
}
/* 顶部 Argon 主色装饰条 */
.banner {
  height: 6px;
  background: linear-gradient(90deg, var(--primary), #7b5dff 50%, var(--info));
}
.pad { padding: 28px 28px 32px; display: flex; flex-direction: column; gap: 14px; }
.brand-line { display: flex; align-items: center; gap: 12px; justify-content: center; }
.logo {
  width: 40px; height: 40px; border-radius: 12px;
  background: linear-gradient(135deg, var(--primary), var(--primary-tint));
  color: #fff; display: flex; align-items: center; justify-content: center;
  font-size: 18px; font-weight: 800; letter-spacing: 1px;
  box-shadow: 0 6px 16px rgba(94,114,228,.32);
}
.brand { font-size: 20px; font-weight: 700; color: var(--text); letter-spacing: .2px; }
.subtitle { font-size: 13px; color: var(--muted); text-align: center; margin-bottom: 6px; }
.field { display: block; }
.loading { text-align: center; color: var(--muted); padding: 20px 0; }
.error {
  font-size: 12.5px; color: var(--danger-fg);
  padding: 8px 12px; border-radius: 8px;
  background: var(--danger-bg);
}
.submit { margin-top: 6px; padding: 10px 16px; font-size: 14px; font-weight: 600; }
</style>
