<template>
  <div class="login-shell flex min-h-screen items-center justify-center px-4">
    <div class="w-full max-w-sm">
      <div class="text-center mb-8">
        <img :src="logoUrl" alt="" class="mx-auto mb-4 h-16 w-16 rounded-2xl shadow-2xl" />
        <h1 class="text-2xl font-bold text-white">EasyLLM</h1>
        <p class="text-gray-500 text-sm mt-1">{{ isSetup ? '设置访问密码' : '请输入密码' }}</p>
      </div>

      <form @submit.prevent="handleSubmit" class="space-y-4">
        <div>
          <input
            v-model="password"
            :type="showPassword ? 'text' : 'password'"
            :placeholder="isSetup ? `设置密码（至少 ${minPasswordLength} 位）` : '输入密码'"
            class="w-full px-4 py-3 bg-gray-900 border border-gray-700 rounded-lg text-white placeholder-gray-500 focus:outline-none focus:border-blue-500 transition-colors"
            autofocus
          />
        </div>
        <div v-if="isSetup">
          <input
            v-model="confirmPassword"
            :type="showPassword ? 'text' : 'password'"
            placeholder="确认密码"
            class="w-full px-4 py-3 bg-gray-900 border border-gray-700 rounded-lg text-white placeholder-gray-500 focus:outline-none focus:border-blue-500 transition-colors"
          />
        </div>

        <label class="flex items-center gap-2 text-sm text-gray-400 cursor-pointer select-none">
          <input type="checkbox" v-model="showPassword" class="rounded border-gray-600" />
          <span>显示密码</span>
        </label>

        <div v-if="error" class="text-red-400 text-sm bg-red-900/30 rounded-lg px-3 py-2">{{ error }}</div>

        <button
          type="submit"
          :disabled="loading"
          class="w-full py-3 bg-blue-600 hover:bg-blue-500 disabled:bg-blue-800 disabled:cursor-not-allowed text-white font-medium rounded-lg transition-colors"
        >
          {{ loading ? '...' : isSetup ? '设置密码' : '登录' }}
        </button>
      </form>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { authAPI } from '../api'
import logoUrl from '@/assets/brand/easyllm-app-icon.png'
import { defaultHomePath } from '@/lib/runtime'

const router = useRouter()
const password = ref('')
const confirmPassword = ref('')
const showPassword = ref(false)
const loading = ref(false)
const error = ref('')
const isSetup = ref(false)
const minPasswordLength = 8

onMounted(async () => {
  try {
    const data = await authAPI.check()
    isSetup.value = !data.password_set
  } catch {
    isSetup.value = true
  }
})

async function handleSubmit() {
  error.value = ''

  if (passwordLength(password.value) < minPasswordLength) {
    error.value = `密码至少 ${minPasswordLength} 位`
    return
  }

  if (isSetup.value) {
    if (password.value !== confirmPassword.value) {
      error.value = '两次密码不一致'
      return
    }
    loading.value = true
    try {
      const data = await authAPI.setup(password.value)
      localStorage.setItem('easyllm_token', data.token)
      router.replace(defaultHomePath())
    } catch (e) {
      error.value = e.message || '设置失败'
    } finally {
      loading.value = false
    }
  } else {
    loading.value = true
    try {
      const data = await authAPI.login(password.value)
      localStorage.setItem('easyllm_token', data.token)
      router.replace(defaultHomePath())
    } catch (e) {
      error.value = '密码错误'
    } finally {
      loading.value = false
    }
  }
}

function passwordLength(value) {
  return Array.from(value || '').length
}
</script>

<style scoped>
.login-shell {
  background:
    radial-gradient(circle at 50% 0%, var(--app-bg-glow), transparent 30rem),
    var(--app-bg);
  color: var(--app-text);
}
</style>
