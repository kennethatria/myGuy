<template>
  <div class="h-full flex flex-col justify-center p-4">
    <div class="container mx-auto" style="max-width: 480px;">
      <div class="text-center mb-4">
        <div class="flex justify-center items-center mb-4">
          <img class="h-12 w-auto" src="../../assets/myguy-icon.svg" alt="MyGuy" />
          <span class="ml-3 text-xl font-bold text-primary">MyGuy</span>
        </div>
        <h1>{{ heading }}</h1>
        <p v-if="step === 'email'" class="text-sm mt-2">
          New here? Enter your email and we'll create your account.
        </p>
      </div>

      <div class="card p-4">
        <!-- Step 1: email -->
        <form v-if="step === 'email'" @submit.prevent="sendCode">
          <div class="form-group">
            <label for="email" class="form-label">Email address</label>
            <input
              id="email"
              v-model="email"
              name="email"
              type="email"
              autocomplete="email"
              required
              autofocus
              class="form-input"
            />
          </div>

          <div v-if="error" class="text-red-500 mb-4" role="alert">{{ error }}</div>

          <button type="submit" class="btn btn-primary w-full" :disabled="loading">
            {{ loading ? 'Sending code...' : 'Email me a sign-in code' }}
          </button>
        </form>

        <!-- Step 2: code -->
        <form v-else-if="step === 'code'" @submit.prevent="submitCode">
          <p class="text-sm mb-4">
            We sent a 6-digit code to <strong>{{ email }}</strong>. It expires in 10 minutes.
          </p>

          <div class="form-group">
            <label for="code" class="form-label">Sign-in code</label>
            <input
              id="code"
              :value="code"
              name="code"
              type="text"
              inputmode="numeric"
              autocomplete="one-time-code"
              pattern="[0-9]{6}"
              maxlength="6"
              required
              autofocus
              class="form-input code-input"
              @input="onCodeInput"
            />
          </div>

          <div v-if="error" class="text-red-500 mb-4" role="alert">{{ error }}</div>

          <button type="submit" class="btn btn-primary w-full mb-4" :disabled="loading || code.length !== 6">
            {{ loading ? 'Checking...' : 'Continue' }}
          </button>

          <div class="flex justify-between text-sm">
            <button type="button" class="link-button text-primary" @click="changeEmail">
              Use a different email
            </button>
            <button
              type="button"
              class="link-button text-primary"
              :disabled="loading || resendIn > 0"
              @click="sendCode"
            >
              {{ resendIn > 0 ? `Resend code in ${resendIn}s` : 'Resend code' }}
            </button>
          </div>
        </form>

        <!-- Step 3: new account details -->
        <form v-else @submit.prevent="submitName">
          <p class="text-sm mb-4">
            Welcome! <strong>{{ email }}</strong> is verified. What should we call you?
          </p>

          <div class="form-group">
            <label for="fullName" class="form-label">Full name</label>
            <input
              id="fullName"
              v-model="fullName"
              name="fullName"
              type="text"
              autocomplete="name"
              required
              autofocus
              class="form-input"
            />
          </div>

          <div v-if="error" class="text-red-500 mb-4" role="alert">{{ error }}</div>

          <button type="submit" class="btn btn-primary w-full" :disabled="loading">
            {{ loading ? 'Creating account...' : 'Create account' }}
          </button>
        </form>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

type Step = 'email' | 'code' | 'name'

const RESEND_COOLDOWN_SECONDS = 60

const router = useRouter()
const authStore = useAuthStore()

const step = ref<Step>('email')
const email = ref('')
const code = ref('')
const fullName = ref('')
const signupToken = ref('')
const error = ref('')
const loading = ref(false)
const resendIn = ref(0)
let resendTimer: ReturnType<typeof setInterval> | undefined

const heading = computed(() => {
  if (step.value === 'code') return 'Check your email'
  if (step.value === 'name') return 'Create your account'
  return 'Sign in or create an account'
})

const startResendCooldown = () => {
  clearInterval(resendTimer)
  resendIn.value = RESEND_COOLDOWN_SECONDS
  resendTimer = setInterval(() => {
    resendIn.value -= 1
    if (resendIn.value <= 0) clearInterval(resendTimer)
  }, 1000)
}

onBeforeUnmount(() => clearInterval(resendTimer))

const run = async (action: () => Promise<void>) => {
  if (loading.value) return
  error.value = ''
  loading.value = true
  try {
    await action()
  } catch (err) {
    error.value = err instanceof Error ? err.message : 'Something went wrong'
  } finally {
    loading.value = false
  }
}

const sendCode = () =>
  run(async () => {
    await authStore.requestCode(email.value.trim())
    code.value = ''
    step.value = 'code'
    startResendCooldown()
  })

const submitCode = () =>
  run(async () => {
    const result = await authStore.verifyCode(email.value.trim(), code.value)
    if (result.signupToken) {
      signupToken.value = result.signupToken
      step.value = 'name'
      return
    }
    await router.push({ name: 'dashboard' })
  })

const submitName = () =>
  run(async () => {
    await authStore.completeSignup(signupToken.value, fullName.value.trim())
    await router.push({ name: 'dashboard' })
  })

// Keep digits only, and submit as soon as all six are in (incl. paste/autofill).
const onCodeInput = (event: Event) => {
  const input = event.target as HTMLInputElement
  code.value = input.value.replace(/\D/g, '').slice(0, 6)
  input.value = code.value
  if (code.value.length === 6) submitCode()
}

const changeEmail = () => {
  step.value = 'email'
  code.value = ''
  error.value = ''
}
</script>

<style scoped>
.code-input {
  letter-spacing: 0.5em;
  font-size: 1.25rem;
  text-align: center;
}

.link-button {
  background: none;
  border: 0;
  padding: 0;
  cursor: pointer;
}

.link-button:disabled {
  opacity: 0.6;
  cursor: default;
}
</style>
