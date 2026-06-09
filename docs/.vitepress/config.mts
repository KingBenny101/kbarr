import { defineConfig } from 'vitepress'

export default defineConfig({
  title: 'kbarr',
  description: 'Take lite',
  base: '/kbarr/',
  ignoreDeadLinks: [/^http:\/\/localhost/],

  themeConfig: {
    nav: [
      { text: 'About', link: '/' },
      { text: 'Installation', link: '/installation' },
      { text: 'Documentation', link: '/api' },
      { text: 'Blog', link: '/blog/' },
    ],

socialLinks: [
      { icon: 'github', link: 'https://github.com/KingBenny101/kbarr' },
    ],

    footer: {
      message: 'Released under the MIT License.',
      copyright: 'Copyright © 2026 KingBenny101',
    },
  },
})
