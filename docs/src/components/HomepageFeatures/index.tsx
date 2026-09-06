import type {ReactNode} from 'react';
import clsx from 'clsx';
import Link from '@docusaurus/Link';
import Heading from '@theme/Heading';
import styles from './styles.module.css';

type FeatureItem = {
  title: string;
  to: string;
  description: ReactNode;
};

const FeatureList: FeatureItem[] = [
  {
    title: 'The Source, Not a Consumer',
    to: '/docs/ecosystem/context-map',
    description: (
      <>
        Defines the canonical process-path catalogue for the fleet.
        fulfillment-execution, wes-work-planning, and workforce-management
        are its intended consumers — this service never reads from them.
      </>
    ),
  },
  {
    title: 'matchPrefix Resolution',
    to: '/docs/ddd/ubiquitous-language',
    description: (
      <>
        A caller-supplied id resolves to a path family by exact match or a
        hyphen-separated prefix match — never a bare substring match — the
        same semantics the retired YAML catalogue already established.
      </>
    ),
  },
  {
    title: 'Deactivation is Terminal',
    to: '/docs/ddd/aggregates-and-invariants',
    description: (
      <>
        Retiring a path is a one-way, idempotent transition. A deactivated
        path is a closed historical record — no mutation is accepted
        against it again.
      </>
    ),
  },
  {
    title: 'Event-Driven Propagation',
    to: '/docs/adr/0001-process-path-management-bounded-context',
    description: (
      <>
        Every change publishes onto warehouse.process-path-management.events.
        No consumer is wired yet in any of the three intended downstream
        repos — that is documented, not silently assumed.
      </>
    ),
  },
];

function Feature({title, to, description}: FeatureItem) {
  return (
    <div className={clsx('col col--3')}>
      <Link to={to} className={styles.featureCard}>
        <Heading as="h3" className={styles.featureTitle}>
          {title}
        </Heading>
        <p className={styles.featureBody}>{description}</p>
      </Link>
    </div>
  );
}

export default function HomepageFeatures(): ReactNode {
  return (
    <section className={styles.features}>
      <div className="container">
        <div className="row">
          {FeatureList.map((props, idx) => (
            <Feature key={idx} {...props} />
          ))}
        </div>
      </div>
    </section>
  );
}
