import { useEffect, useState } from 'react';
import { useI18n } from '@/context/I18nContext';
import { api, type DashboardStats, formatBytes } from '@/lib/api';
import styles from '@/styles/components.module.css';

export default function Dashboard() {
  const { t } = useI18n();
  const [stats, setStats] = useState<DashboardStats | null>(null);

  useEffect(() => {
    api.stats().then(setStats).catch(() => {});
  }, []);

  return (
    <div className={styles.page}>
      <h1 className={styles.pageTitle}>{t('dashboard.title')}</h1>
      <div className={styles.cardGrid}>
        <div className={styles.card}>
          <div className={styles.cardLabel}>{t('dashboard.total_tenants')}</div>
          <div className={styles.cardValue}>{stats?.total_tenants ?? '-'}</div>
        </div>
        <div className={styles.card}>
          <div className={styles.cardLabel}>{t('dashboard.total_files')}</div>
          <div className={styles.cardValue}>{stats?.total_files ?? '-'}</div>
        </div>
        <div className={styles.card}>
          <div className={styles.cardLabel}>{t('dashboard.traffic_used')}</div>
          <div className={styles.cardValue}>{stats ? formatBytes(stats.traffic_used) : '-'}</div>
        </div>
        <div className={styles.card}>
          <div className={styles.cardLabel}>{t('dashboard.storage_used')}</div>
          <div className={styles.cardValue}>{stats ? formatBytes(stats.storage_used) : '-'}</div>
        </div>
        <div className={styles.card}>
          <div className={styles.cardLabel}>{t('dashboard.api_calls')}</div>
          <div className={styles.cardValue}>{stats?.api_calls_today ?? '-'}</div>
        </div>
      </div>
    </div>
  );
}
