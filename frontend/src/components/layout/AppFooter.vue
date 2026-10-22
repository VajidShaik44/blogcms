<template>
  <footer class="bg-secondary-800 text-white">
    <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-12">
      <div class="grid grid-cols-1 md:grid-cols-4 gap-8">
        <!-- Brand -->
        <div class="col-span-1 md:col-span-2">
          <div class="flex items-center space-x-2 mb-4">
            <div class="h-8 w-8 bg-primary-600 rounded-lg flex items-center justify-center">
              <span class="text-white font-bold text-lg">B</span>
            </div>
            <span class="text-xl font-bold">BlogCMS</span>
          </div>
          <p class="text-secondary-300 mb-4 max-w-md">
            A modern content management system built with Vue 3 and Go. 
            Create, manage, and publish your content with ease.
          </p>
          <div class="flex flex-wrap items-center gap-4">
            <a
              href="https://github.com/VajidShaik44/blogcms"
              target="_blank"
              rel="noopener noreferrer"
              class="text-secondary-400 hover:text-white"
            >
              <span class="sr-only">BlogCMS source code on GitHub</span>
              <svg class="h-6 w-6" fill="currentColor" viewBox="0 0 24 24">
                <path d="M12 0c-6.626 0-12 5.373-12 12 0 5.302 3.438 9.8 8.207 11.387.599.111.793-.261.793-.577v-2.234c-3.338.726-4.033-1.416-4.033-1.416-.546-1.387-1.333-1.756-1.333-1.756-1.089-.745.083-.729.083-.729 1.205.084 1.839 1.237 1.839 1.237 1.07 1.834 2.807 1.304 3.492.997.107-.775.418-1.305.762-1.604-2.665-.305-5.467-1.334-5.467-5.931 0-1.311.469-2.381 1.236-3.221-.124-.303-.535-1.524.117-3.176 0 0 1.008-.322 3.301 1.23.957-.266 1.983-.399 3.003-.404 1.02.005 2.047.138 3.006.404 2.291-1.552 3.297-1.23 3.297-1.23.653 1.653.242 2.874.118 3.176.77.84 1.235 1.911 1.235 3.221 0 4.609-2.807 5.624-5.479 5.921.43.372.823 1.102.823 2.222v3.293c0 .319.192.694.801.576 4.765-1.589 8.199-6.086 8.199-11.386 0-6.627-5.373-12-12-12z"/>
              </svg>
            </a>
            <a
              href="https://linkedin.com/in/vajid-shaik"
              target="_blank"
              rel="noopener noreferrer"
              class="text-secondary-400 hover:text-white"
            >
              <span class="sr-only">LinkedIn profile</span>
              <svg class="h-6 w-6" fill="currentColor" viewBox="0 0 24 24">
                <path d="M20.447 20.452h-3.554v-5.569c0-1.328-.027-3.037-1.852-3.037-1.853 0-2.136 1.445-2.136 2.939v5.667H9.351V9h3.414v1.561h.049c.476-.9 1.637-1.85 3.37-1.85 3.601 0 4.267 2.37 4.267 5.455v6.286zM5.337 7.433a2.062 2.062 0 110-4.124 2.062 2.062 0 010 4.124zM3.555 9h3.564v11.452H3.555V9zM22.225 0H1.771C.792 0 0 .774 0 1.729v20.542C0 23.227.792 24 1.771 24h20.451C23.2 24 24 23.227 24 22.271V1.729C24 .774 23.2 0 22.222 0h.003z"/>
              </svg>
            </a>
            <a
              href="https://vajidshaik44.github.io"
              target="_blank"
              rel="noopener noreferrer"
              class="text-secondary-300 hover:text-white text-sm"
            >
              Portfolio
            </a>
          </div>
        </div>

        <!-- Quick Links -->
        <div>
          <h3 class="text-sm font-semibold uppercase tracking-wider mb-4">Quick Links</h3>
          <ul class="space-y-2">
            <li>
              <RouterLink to="/" class="text-secondary-300 hover:text-white">Home</RouterLink>
            </li>
            <li>
              <RouterLink to="/posts" class="text-secondary-300 hover:text-white">All Posts</RouterLink>
            </li>
            <li>
              <RouterLink to="/login" class="text-secondary-300 hover:text-white">Login</RouterLink>
            </li>
            <li>
              <RouterLink to="/register" class="text-secondary-300 hover:text-white">Register</RouterLink>
            </li>
          </ul>
        </div>

        <!-- Categories -->
        <div>
          <h3 class="text-sm font-semibold uppercase tracking-wider mb-4">Categories</h3>
          <ul class="space-y-2">
            <li v-for="category in categories.slice(0, 5)" :key="category.id">
              <RouterLink 
                :to="`/categories/${category.slug}`" 
                class="text-secondary-300 hover:text-white"
              >
                {{ category.name }}
              </RouterLink>
            </li>
          </ul>
        </div>
      </div>

      <div class="mt-8 pt-8 border-t border-secondary-700">
        <div class="flex flex-col md:flex-row justify-between items-center">
          <p class="text-secondary-400 text-sm">
            &copy; {{ currentYear }} BlogCMS. All rights reserved.
          </p>
          <div class="flex space-x-6 mt-4 md:mt-0">
            <a href="https://github.com/VajidShaik44" target="_blank" rel="noopener noreferrer" class="text-secondary-400 hover:text-white text-sm">GitHub profile</a>
            <a href="https://linkedin.com/in/vajid-shaik" target="_blank" rel="noopener noreferrer" class="text-secondary-400 hover:text-white text-sm">Contact</a>
          </div>
        </div>
      </div>
    </div>
  </footer>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { RouterLink } from 'vue-router'
import { useCategoriesStore } from '@/stores/categories'

const categoriesStore = useCategoriesStore()
const categories = ref([])
const currentYear = new Date().getFullYear()

onMounted(async () => {
  try {
    categories.value = (await categoriesStore.fetchCategories()).data
  } catch (error) {
    console.error('Failed to fetch categories:', error)
  }
})
</script>
