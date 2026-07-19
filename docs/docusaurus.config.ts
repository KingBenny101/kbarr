import { themes as prismThemes } from 'prism-react-renderer';
import type { Config } from '@docusaurus/types';
import type * as Preset from '@docusaurus/preset-classic';
import type * as OpenApiPlugin from 'docusaurus-plugin-openapi-docs';

const config: Config = {
  title: 'kbarr',
  tagline: 'Self-hosted automatic anime torrent downloader.',
  favicon: 'img/logo.svg',

  url: 'https://kingbenny101.github.io',
  baseUrl: '/kbarr/',

  organizationName: 'KingBenny101',
  projectName: 'kbarr',

  onBrokenLinks: 'throw',
  markdown: {
    mermaid: true,
    hooks: {
      onBrokenMarkdownLinks: 'warn',
    },
  },

  i18n: {
    defaultLocale: 'en',
    locales: ['en'],
  },

  presets: [
    [
      'classic',
      {
        docs: {
          sidebarPath: './sidebars.ts',
          docItemComponent: '@theme/ApiItem',
        },
        blog: {
          showReadingTime: false,
        },
        theme: {
          customCss: './src/css/custom.css',
        },
      } satisfies Preset.Options,
    ],
  ],

  plugins: [
    [
      'docusaurus-plugin-openapi-docs',
      {
        id: 'api',
        docsPluginId: 'classic',
        config: {
          kbarr: {
            specPath: './openapi.json',
            outputDir: 'docs/api',
            sidebarOptions: {
              groupPathsBy: 'tag',
              categoryLinkSource: 'tag',
            },
          } satisfies OpenApiPlugin.Options,
        },
      },
    ],
    [
      '@cmfcmf/docusaurus-search-local',
      {
        indexDocs: true,
        indexBlog: false,
        indexPages: true,
      },
    ],
  ],

  themes: ['docusaurus-theme-openapi-docs', '@docusaurus/theme-mermaid'],

  themeConfig: {
    navbar: {
      title: 'kbarr',
      logo: {
        alt: 'kbarr',
        src: 'img/logo.svg',
      },
      items: [
        {
          type: 'docSidebar',
          sidebarId: 'docs',
          position: 'left',
          label: 'Docs',
        },
        {
          type: 'docSidebar',
          sidebarId: 'api',
          position: 'left',
          label: 'API',
        },
        { to: '/blog', label: 'Blog', position: 'left' },
        {
          href: 'https://github.com/KingBenny101/kbarr',
          label: 'GitHub',
          position: 'right',
        },
      ],
    },
    footer: {
      style: 'dark',
      links: [
        {
          title: 'Docs',
          items: [
            { label: 'Installation', to: '/docs/installation' },
            { label: 'Development', to: '/docs/development' },
            { label: 'API Reference', to: '/docs/api/kbarr-api' },
          ],
        },
        {
          title: 'More',
          items: [
            { label: 'Blog', to: '/blog' },
            { label: 'GitHub', href: 'https://github.com/KingBenny101/kbarr' },
          ],
        },
      ],
      copyright: `Copyright © ${new Date().getFullYear()} KingBenny101. Released under the MIT License.`,
    },
    colorMode: {
      defaultMode: 'dark',
      disableSwitch: false,
      respectPrefersColorScheme: true,
    },
    prism: {
      theme: prismThemes.github,
      darkTheme: prismThemes.dracula,
      additionalLanguages: ['bash', 'json'],
    },
  } satisfies Preset.ThemeConfig,
};

export default config;
