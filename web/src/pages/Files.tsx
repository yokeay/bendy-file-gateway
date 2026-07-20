import { useI18n } from '@/context/I18nContext';
import styles from '@/styles/components.module.css';

export default function Files() {
  const { t } = useI18n();

  return (
    <div className={styles.page}>
      <h1 className={styles.pageTitle}>{t('files.title')}</h1>
      <div className={styles.placeholder}>File management coming soon</div>
    </div>
  );
}
