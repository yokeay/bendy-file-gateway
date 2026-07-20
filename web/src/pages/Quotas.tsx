import { useI18n } from '@/context/I18nContext';
import styles from '@/styles/components.module.css';

export default function Quotas() {
  const { t } = useI18n();

  return (
    <div className={styles.page}>
      <h1 className={styles.pageTitle}>{t('quotas.title')}</h1>
      <div className={styles.placeholder}>Quota management coming soon</div>
    </div>
  );
}
