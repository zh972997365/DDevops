<template>
  <div class="min-h-screen bg-canvas lg:grid lg:grid-cols-[1.04fr_0.96fr]">
    <!-- 品牌侧：随主题翻转，只承载品牌与一句定位 -->
    <section class="login-hero hidden lg:flex lg:flex-col">
      <div class="stagger flex h-full flex-col px-10 py-9 xl:px-14 xl:py-11">
        <div class="flex items-center gap-2.5">
          <span class="brand-logo" aria-hidden="true"><img src="/brand-mark.svg" alt="" /></span>
          <p class="text-[13.5px] font-semibold leading-4 text-[color:var(--hero-ink)]">DDevOps</p>
        </div>

        <div class="my-auto max-w-[360px]">
          <h1 class="text-[27px] font-semibold leading-[1.35] tracking-[-0.025em] text-[color:var(--hero-ink)]">
            自动化发布平台
          </h1>
          <p class="mt-3 text-[12.5px] leading-6 text-[color:var(--hero-ink-2)]">
            Jenkins 构建、环境规则、定时发布与钉钉通知，收敛在同一条发布链路。
          </p>
        </div>

        <div class="login-pipeline" aria-hidden="true">
          <p class="text-[10.5px] leading-4 text-[color:var(--hero-ink-2)]">发布链路</p>
          <div class="pipeline">
            <span class="pipeline-rail" />
            <ol class="pipeline-steps">
              <li>测试</li>
              <li>预发</li>
              <li>生产</li>
            </ol>
          </div>
        </div>
      </div>
    </section>

    <!-- 表单侧 -->
    <section class="flex min-h-screen flex-col bg-solid">
      <header class="flex items-center justify-between px-6 py-5 lg:px-10">
        <div class="flex items-center gap-2.5 lg:hidden">
          <span class="brand-logo" aria-hidden="true"><img src="/brand-mark.svg" alt="" /></span>
          <span class="text-[13.5px] font-semibold text-ink">DDevOps</span>
        </div>
      </header>

      <main class="flex flex-1 items-center justify-center px-5 pb-14">
        <div class="w-full max-w-[372px]">
          <h2 class="text-[24px] font-semibold tracking-[-0.02em] text-ink">登录控制台</h2>
          <p class="mt-1 text-sm leading-6 text-ink-3">账号由管理员分配，Web 与钉钉共用同一套权限。</p>

          <form class="mt-7 space-y-4" @submit.prevent="onSubmit">
            <BaseInput
              v-model="username"
              label="用户名"
              placeholder="请输入用户名"
              autocomplete="username"
            >
              <template #prefix><User class="h-3.5 w-3.5" /></template>
            </BaseInput>

            <BaseInput
              v-model="password"
              label="密码"
              type="password"
              placeholder="请输入密码"
              autocomplete="current-password"
            >
              <template #prefix><Lock class="h-3.5 w-3.5" /></template>
            </BaseInput>

            <div class="flex items-center justify-between pt-0.5">
              <label class="flex cursor-pointer items-center gap-2 text-xs text-ink-2">
                <input v-model="remember" type="checkbox" class="h-3.5 w-3.5 rounded-[3px] accent-[var(--primary)]" />
                记住用户名
              </label>
              <span class="text-[10.5px] text-ink-3">密码由管理员重置</span>
            </div>

            <Transition name="err">
              <p
                v-if="error"
                class="rounded-field border border-failed bg-failed-soft px-3 py-2 text-xs leading-5 text-failed"
                role="alert"
              >
                {{ error }}
              </p>
            </Transition>

            <BaseButton type="submit" size="lg" block :loading="loading" class="mt-1">
              进入控制台
            </BaseButton>
          </form>

        </div>
      </main>

      <footer class="flex items-center justify-between px-6 pb-6 text-[10.5px] text-ink-3 lg:px-10">
        <span>DDevOps / 自动化发布平台</span>
        <span>© 2026</span>
      </footer>
    </section>
  </div>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { Lock, User } from 'lucide-vue-next'
import BaseInput from '@/components/base/BaseInput.vue'
import BaseButton from '@/components/base/BaseButton.vue'
import { useAuth } from '@/composables/useAuth'

const router = useRouter()
const route = useRoute()
const { login } = useAuth()

const username = ref(localStorage.getItem('devops_last_user') || '')
const password = ref('')
const remember = ref(!!localStorage.getItem('devops_last_user'))
const loading = ref(false)
const error = ref('')

async function onSubmit() {
  if (!username.value.trim() || !password.value) {
    error.value = '请输入用户名和密码'
    return
  }
  loading.value = true
  error.value = ''
  try {
    await login(username.value.trim(), password.value)
    if (remember.value) localStorage.setItem('devops_last_user', username.value.trim())
    else localStorage.removeItem('devops_last_user')
    const redirect = typeof route.query.redirect === 'string' && route.query.redirect.startsWith('/')
      ? route.query.redirect
      : '/'
    router.replace(redirect)
  } catch (e) {
    error.value = e.message || '登录失败，请重试'
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  if (username.value) document.querySelector('input[type="password"]')?.focus()
  else document.querySelector('input[autocomplete="username"]')?.focus()
})
</script>

<style scoped>
.login-hero {
  background: var(--hero);
}
.login-pipeline {
  max-width: 360px;
}
.pipeline {
  position: relative;
  overflow: hidden;
  margin-top: 12px;
  padding-top: 2px;
}
.pipeline-rail {
  position: absolute;
  top: 4px;
  left: 16.666%;
  right: 16.666%;
  height: 3px;
  background-image: linear-gradient(90deg, transparent, var(--hero-sweep), transparent);
  background-repeat: no-repeat;
  background-size: 34% 100%;
  background-position: -40% 0;
  animation: pipeline-sweep 7s linear infinite;
}
.pipeline-rail::before {
  content: '';
  position: absolute;
  top: 1px;
  right: 0;
  left: 0;
  height: 1px;
  background: var(--hero-track);
}
.pipeline-steps {
  position: relative;
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  margin: 0;
  padding: 0;
  list-style: none;
}
.pipeline-steps li {
  position: relative;
  padding-top: 16px;
  color: var(--hero-ink-2);
  font-size: 11px;
  line-height: 1;
  text-align: center;
}
.pipeline-steps li::before {
  content: '';
  position: absolute;
  top: 2px;
  left: 50%;
  width: 7px;
  height: 7px;
  transform: translateX(-50%);
  border: 1px solid var(--hero-node);
  border-radius: 1px;
  background: var(--hero);
  animation: pipeline-node 7s var(--ease) infinite;
}
.pipeline-steps li:nth-child(2)::before {
  animation-delay: 1.2s;
}
.pipeline-steps li:nth-child(3)::before {
  animation-delay: 2.4s;
}
@keyframes pipeline-sweep {
  0%,
  10% {
    background-position: -40% 0;
  }
  75%,
  100% {
    background-position: 140% 0;
  }
}
@keyframes pipeline-node {
  0%,
  88%,
  100% {
    border-color: var(--hero-node);
    background: var(--hero);
  }
  6%,
  20% {
    border-color: var(--accent);
    background: var(--accent);
  }
}
@media (prefers-reduced-motion: reduce) {
  .pipeline-rail {
    background-image: none;
    animation: none;
  }
  .pipeline-steps li::before {
    animation: none;
  }
}
.err-enter-active,
.err-leave-active {
  transition: opacity 140ms var(--ease), transform 140ms var(--ease);
}
.err-enter-from,
.err-leave-to {
  transform: translateY(-3px);
}
</style>