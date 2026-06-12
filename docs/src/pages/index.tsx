import clsx from 'clsx';
import Link from '@docusaurus/Link';
import Layout from '@theme/Layout';
import Heading from '@theme/Heading';
import styles from './index.module.css';

function Hero() {
  return (
    <header className={clsx('hero hero--primary', styles.hero)}>
      <div className="container">
        <Heading as="h1" className="hero__title">kbarr</Heading>
        <p className={styles.heroSubtitle}>
          Self-hosted anime monitoring and automatic downloading.
        </p>
        <div className={styles.heroActions}>
          <Link className="button button--secondary button--lg" to="/docs/installation">
            Get Started
          </Link>
          <Link className="button button--outline button--secondary button--lg" to="/docs/development">
            Development
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


export default function Home(): JSX.Element {
  return (
    <Layout description="A self-hosted anime management application.">
      <Hero />
      <main className={styles.content}>
        <div className="container">
          <div className={styles.inner}>

            <section className={styles.section}>
              <Heading as="h2">Disclaimer</Heading>
              <div className={styles.disclaimer}>
                <p>
                  This is a hobby project. The entire codebase was built using AI (Claude) —
                  no human-written code, with some manual debugging along the way.
                </p>
                <p>
                  It works for personal use but is not actively maintained as a public project.
                  Use it at your own risk. Issues may or may not be addressed.
                </p>
              </div>
            </section>

            <section className={styles.section}>
              <p>
                Add a show, monitor the episodes you want, and kbarr handles the rest —
                searching for matching torrents, queuing them in qBittorrent, and
                organising finished downloads into your media library. Runs via Docker.
              </p>
              <p>
                Metadata is sourced from <strong>AniDB</strong>. Torrent search runs
                through <strong><a href="https://github.com/KingBenny101/kbdex">kbdex</a></strong> (recommended, no API key needed) or <strong>Prowlarr</strong>.
                Downloads go through <strong>qBittorrent</strong>, with more clients
                planned.
              </p>
            </section>

          </div>
        </div>
      </main>
    </Layout>
  );
}
