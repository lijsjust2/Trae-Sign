<script setup lang="ts">
import { ref } from 'vue'
import { api } from '../api'
import { useStore } from '../store'
import type { Account } from '../types'

const props = defineProps<{ account: Account }>()
const emit = defineEmits<{ close: [], done: [] }>()
const store = useStore()

const remark = ref(props.account.remark)
const checkinTime = ref(props.account.checkinTime)
const pushplusToken = ref(props.account.pushplusToken)
const enabled = ref(props.account.enabled)
const busy = ref(false)

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
        <label class="field">
          <span class="neu-label">签到时间（HH:mm）</span>
          <input class="neu-input" type="time" v-model="checkinTime" />
        </label>
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
</style>
