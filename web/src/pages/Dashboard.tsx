import { useI18n } from '@/context/I18nContext';
import styles from '@/styles/components.module.css';

export default function Dashboard() {
  const { t } = useI18n();

  return (
    <div className={styles.page}>
      <h1 className={styles.pageTitle}>{t('dashboard.title')}</h1>
      <div className={styles.cardGrid}>
        <div className={styles.card}>
          <div className={styles.cardLabel}>{t('dashboard.total_tenants')}</div>
          <div className={styles.cardValue}>0</div>
        </div>
        <div className={styles.card}>
          <div className={styles.cardLabel}>{t('dashboard.total_files')}</div>
          <div className={styles.cardValue}>0</div>
        </div>
        <div className={styles.card}>
          <div className={styles.cardLabel}>{t('dashboard.traffic_used')}</div>
          <div className={styles.cardValue}>0 B</div>
        </div>
        <div className={styles.card}>
          <div className={styles.cardLabel}>{t('dashboard.api_calls')}</div>
          <div className={styles.cardValue}>0</div>
        </div>
      </div>
    </div>
  );
}
