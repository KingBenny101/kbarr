import clsx from 'clsx';
import Link from '@docusaurus/Link';
import useDocusaurusContext from '@docusaurus/useDocusaurusContext';
import Layout from '@theme/Layout';
import Heading from '@theme/Heading';
import styles from './index.module.css';

function Hero() {
  return (
    <header className={clsx('hero hero--primary', styles.hero)}>
      <div className="container">
        <Heading as="h1" className="hero__title">kbarr</Heading>
        <p className="hero__subtitle">Take lite</p>
        <p className={styles.heroTagline}>
          A self-hosted anime management application.
        </p>
        <div className={styles.heroActions}>
          <Link className="button button--secondary button--lg" to="/docs/installation">
            Get Started
          </Link>
          <Link
            className="button button--outline button--secondary button--lg"
            href="https://github.com/KingBenny101/kbarr"
          >
            GitHub
          </Link>
        </div>
      </div>
    </header>
  );
}

type Feature = { title: string; description: string };

const features: Feature[] = [
  {
    title: 'AniDB',
    description: 'Uses AniDB as the source for anime metadata.',
  },
  {
    title: 'Docker-first',
    description: 'Runs entirely via Docker Compose. No local dependencies required.',
  },
  {
    title: 'Built in Go',
    description: 'Hobby project to learn Go. Vibe coded while listening to songs.',
  },
];

function Features() {
  return (
    <section className={styles.features}>
      <div className="container">
        <div className="row">
          {features.map(({ title, description }) => (
            <div key={title} className={clsx('col col--4')}>
              <div className={styles.featureCard}>
                <Heading as="h3">{title}</Heading>
                <p>{description}</p>
              </div>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}

function Content() {
  return (
    <main className={styles.content}>
      <div className="container">
        <section>
          <Heading as="h2">What is kbarr?</Heading>
          <p>
            kbarr is a self-hosted application for managing and automatically downloading anime. You
            add a show, it monitors for new episodes and handles the rest — searching, downloading,
            and organising.
          </p>
          <p>It was built as a hobby project to learn Go, using AniDB as the metadata source.</p>
        </section>

        <section>
          <Heading as="h2">How it works</Heading>
          <p>kbarr runs as a set of independent services under Docker Compose:</p>
          <table>
            <thead>
              <tr>
                <th>Service</th>
                <th>Role</th>
              </tr>
            </thead>
            <tbody>
              <tr><td><strong>core</strong></td><td>Main API and frontend</td></tr>
              <tr><td><strong>metadata</strong></td><td>Fetches and caches anime metadata from AniDB</td></tr>
              <tr><td><strong>indexer</strong></td><td>Matches episodes to torrent files</td></tr>
              <tr><td><strong>downloader</strong></td><td>Manages the download queue</td></tr>
            </tbody>
          </table>
          <p>Each service runs independently and communicates over the internal Docker network.</p>
        </section>

        <section>
          <Heading as="h2">Workflow</Heading>
          <ol>
            <li><strong>Search</strong> for an anime via the UI</li>
            <li><strong>Add</strong> it to your library</li>
            <li><strong>Monitor</strong> the seasons and episodes you want</li>
            <li>kbarr searches for matching torrents and <strong>downloads</strong> them automatically</li>
          </ol>
        </section>

        <section>
          <Heading as="h2">Tech stack</Heading>
          <ul>
            <li><strong>Backend</strong> — Go</li>
            <li><strong>Frontend</strong> — React + Mantine</li>
            <li><strong>Database</strong> — PostgreSQL</li>
            <li><strong>Metadata</strong> — AniDB</li>
            <li><strong>Torrent search</strong> — Nyaa.si</li>
          </ul>
        </section>
      </div>
    </main>
  );
}

export default function Home(): JSX.Element {
  const { siteConfig } = useDocusaurusContext();
  return (
    <Layout description={siteConfig.tagline}>
      <Hero />
      <Features />
      <Content />
    </Layout>
  );
}
