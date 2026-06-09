import { createContentLoader } from 'vitepress'

export default createContentLoader('blog/posts/*.md', {
  excerpt: true,
  transform(data) {
    return data.sort((a, b) =>
      +new Date(b.frontmatter.date) - +new Date(a.frontmatter.date)
    )
  },
})
