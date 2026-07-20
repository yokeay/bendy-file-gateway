import { useTheme } from '@/context/ThemeContext';
import { useI18n } from '@/context/I18nContext';
import styles from '@/styles/components.module.css';

export default function Settings() {
  const { theme, toggleTheme } = useTheme();
  const { locale, setLocale, t } = useI18n();

  return (
    <div className={styles.page}>
      <h1 className={styles.pageTitle}>{t('settings.title')}</h1>
      <div className={styles.cardGrid}>
        <div className={styles.card}>
          <div className={styles.cardLabel}>{t('settings.theme')}</div>
          <button
            className={`${styles.btn} ${styles.btnPrimary}`}
            onClick={toggleTheme}
          >
            {theme === 'light' ? t('settings.theme_dark') : t('settings.theme_light')}
          </button>
        </div>
        <div className={styles.card}>
          <div className={styles.cardLabel}>{t('settings.language')}</div>
          <button
            className={`${styles.btn} ${styles.btnPrimary}`}
            onClick={() => setLocale(locale === 'zh' ? 'en' : 'zh')}
          >
            {locale === 'zh' ? 'English' : '中文'}
          </button>
        </div>
      </div>
    </div>
  );
}
