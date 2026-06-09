---
layout: doc
---

# Blog

<script setup>
import { withBase } from 'vitepress'
import { data as posts } from './posts.data.mts'

function formatDate(date) {
  return new Date(date).toLocaleDateString('en-US', {
    year: 'numeric',
    month: 'long',
    day: 'numeric',
  })
}
</script>

<div class="blog-list">
  <div v-for="post in posts" :key="post.url" class="blog-post">
    <p class="post-date">{{ formatDate(post.frontmatter.date) }}</p>
    <h2 class="post-title">
      <a :href="withBase(post.url)">{{ post.frontmatter.title }}</a>
    </h2>
    <div class="post-excerpt" v-html="post.excerpt" />
    <a :href="withBase(post.url)" class="read-more">Read more →</a>
  </div>
</div>

<style scoped>
.blog-list {
  margin-top: 2rem;
}

.blog-post {
  padding: 2rem 0;
  border-bottom: 1px solid var(--vp-c-divider);
}

.blog-post:last-child {
  border-bottom: none;
}

.post-date {
  font-size: 0.875rem;
  color: var(--vp-c-text-2);
  margin: 0 0 0.5rem;
}

.post-title {
  font-size: 1.5rem;
  font-weight: 600;
  margin: 0 0 1rem;
  border: none;
  padding: 0;
}

.post-title a {
  color: var(--vp-c-text-1);
  text-decoration: none;
}

.post-title a:hover {
  color: var(--vp-c-brand-1);
}

.post-excerpt {
  color: var(--vp-c-text-2);
  line-height: 1.7;
  margin-bottom: 1rem;
}

.read-more {
  font-size: 0.9rem;
  font-weight: 500;
  color: var(--vp-c-brand-1);
  text-decoration: none;
}

.read-more:hover {
  text-decoration: underline;
}
</style>
