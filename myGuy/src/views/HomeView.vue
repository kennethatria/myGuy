<script setup lang="ts">
import { useRouter } from 'vue-router'

const router = useRouter()

const navigateToLogin = () => {
  router.push({ name: 'login' })
}
</script>

<template>
  <div class="h-full flex flex-col justify-center p-4">
    <div class="container mx-auto max-width-md">
      <div class="flex justify-center items-center">
        <img class="h-12 w-auto" src="../assets/myguy-icon.svg" alt="MyGuy" />
        <span class="ml-3 text-xl font-bold text-primary">MyGuy</span>
      </div>
      <h1 class="mt-4 text-center">
        Welcome to MyGuy
      </h1>
      <p class="text-center text-sm">
        Your platform for connecting with skilled professionals and getting things done
      </p>
    </div>

    <div class="container mx-auto mt-4" style="max-width: 480px;">
      <div class="card p-4">
        <div class="my-4">
          <div class="relative py-2">
            <div class="w-full" style="border-top: 1px solid var(--color-border);"></div>
            <div class="flex justify-center" style="margin-top: -12px;">
              <span class="px-2 bg-white text-sm">
                Ready to get started?
              </span>
            </div>
          </div>

          <div class="mt-4 flex flex-col">
            <button
              @click="navigateToLogin"
              class="btn btn-primary w-full mb-2"
            >
              Sign In
            </button>
            
            <router-link
              :to="{ name: 'register' }"
              class="btn btn-outline w-full"
            >
              Create an Account
            </router-link>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
