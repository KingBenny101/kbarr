import Link from '@docusaurus/Link';
import Layout from '@theme/Layout';
import Heading from '@theme/Heading';
import styles from './index.module.css';

function Hero() {
  return (
    <header className={styles.hero}>
      <div className={styles.heroContent}>
        <svg className={styles.heroLogo} viewBox="0 0 32 32" width="80" height="80">
          <rect width="32" height="32" rx="7" fill="#D4B000"/>
          <polygon points="16,3 26.2,7.9 28.7,18.9 21.6,27.7 10.4,27.7 3.3,18.9 5.8,7.9" fill="none" stroke="#fff" strokeWidth="1.5" strokeLinejoin="round"/>
          <polygon points="21.3,8.7 24.6,18.8 16,25 7.4,18.8 10.7,8.7" fill="none" stroke="#fff" strokeWidth="1.5" strokeLinejoin="round"/>
          <polygon points="16,11 20.3,18.5 11.7,18.5" fill="none" stroke="#fff" strokeWidth="1.5" strokeLinejoin="round"/>
        </svg>
        <h1 className={styles.heroTitle}>kbarr</h1>
        <p className={styles.heroSubtitle}>
          Self-hosted automatic anime torrent downloader.
        </p>
        <div className={styles.heroActions}>
          <Link className={`${styles.heroButton} ${styles.buttonPrimary}`} to="/docs/installation">
            Get Started
          </Link>
          <Link className={`${styles.heroButton} ${styles.buttonSecondary}`} href="https://github.com/KingBenny101/kbarr">
            GitHub
          </Link>
        </div>
      </div>
    </header>
  );
}

function Features() {
  const features = [
    {
      title: 'Browse & Discover',
      description: 'Explore anime by genre, season, and popularity through the AniList API.',
      icon: (
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" className={styles.featureIcon}>
          <circle cx="12" cy="12" r="10" />
          <path d="M8 14s1.5 2 4 2 4-2 4-2" />
          <circle cx="9" cy="9" r="1" />
          <circle cx="15" cy="9" r="1" />
        </svg>
      ),
    },
    {
      title: 'Auto-monitoring',
      description: 'Add a show once and let kbarr monitor it for new episodes automatically.',
      icon: (
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" className={styles.featureIcon}>
          <circle cx="12" cy="12" r="9" />
          <polyline points="12 7 12 12 16 14" />
        </svg>
      ),
    },
    {
      title: 'Indexers',
      description: 'Search for releases via kbdex (no API key needed) or Prowlarr. Additional indexers can be added.',
      icon: (
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" className={styles.featureIcon}>
          <circle cx="11" cy="11" r="8" />
          <path d="m21 21-4.35-4.35" />
        </svg>
      ),
    },
    {
      title: 'Downloading',
      description: 'Queue matched torrents in qBittorrent. Additional download clients can be added.',
      icon: (
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" className={styles.featureIcon}>
          <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4" />
          <polyline points="7 10 12 15 17 10" />
          <line x1="12" y1="15" x2="12" y2="3" />
        </svg>
      ),
    },
    {
      title: 'Rich Metadata',
      description: 'Pull comprehensive anime information from AniDB, including descriptions and episode counts.',
      icon: (
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" className={styles.featureIcon}>
          <path d="M4 19.5A2.5 2.5 0 0 1 6.5 17H20" />
          <path d="M6.5 2H20v20H6.5A2.5 2.5 0 0 1 4 19.5v-15A2.5 2.5 0 0 1 6.5 2z" />
        </svg>
      ),
    },
    {
      title: 'Subtitle Preference',
      description: 'Prefer torrent releases that already include subtitles in your chosen language.',
      icon: (
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" className={styles.featureIcon}>
          <path d="M3 6h18M3 10h18M3 14h18M3 18h18" />
        </svg>
      ),
    },
  ];

  return (
    <section className={styles.features}>
      {features.map((feature, idx) => (
        <div key={idx} className={styles.featureCard}>
          {feature.icon}
          <h3 className={styles.featureTitle}>{feature.title}</h3>
          <p className={styles.featureDescription}>{feature.description}</p>
        </div>
      ))}
    </section>
  );
}

function HowItWorks() {
  return (
    <section className={styles.howItWorks}>
      <h2 className="section-title" style={{ textAlign: 'center', marginBottom: '3rem' }}>How it works</h2>
      <div className={styles.stepsContainer}>
        <div className={styles.step}>
          <div className={styles.stepNumber}>1</div>
          <h3 className={styles.stepTitle}>Add a show</h3>
          <p className={styles.stepDescription}>
            Browse anime or search by title using the built-in search.
          </p>
        </div>
        <div className={styles.step}>
          <div className={styles.stepNumber}>2</div>
          <h3 className={styles.stepTitle}>kbarr handles it</h3>
          <p className={styles.stepDescription}>
            Monitors for new episodes, finds torrents, queues downloads.
          </p>
        </div>
        <div className={styles.step}>
          <div className={styles.stepNumber}>3</div>
          <h3 className={styles.stepTitle}>Watch & enjoy</h3>
          <p className={styles.stepDescription}>
            Files are organized in your media library and optionally trigger a Jellyfin refresh.
          </p>
        </div>
      </div>
    </section>
  );
}

function Disclaimer() {
  return (
    <section className={styles.disclaimer}>
      <h3 className={styles.disclaimerTitle}>⚠️ Disclaimer</h3>
      <p className={styles.disclaimerText}>
        This is a hobby project built entirely using AI (Claude). It works for personal use but is not actively maintained as an open-source project. Use at your own risk — issues may or may not be addressed.
      </p>
    </section>
  );
}

export default function Home(): JSX.Element {
  return (
    <Layout description="Self-hosted automatic anime torrent downloader.">
      <Hero />
      <Features />
      <HowItWorks />
      <Disclaimer />
    </Layout>
  );
}
