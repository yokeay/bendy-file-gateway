import { useAuth } from '@/context/AuthContext';
import { useTheme } from '@/context/ThemeContext';
import { useI18n } from '@/context/I18nContext';
import styles from '@/styles/components.module.css';

export default function Settings() {
  const { user } = useAuth();
  const { theme, toggleTheme } = useTheme();
  const { locale, setLocale, t } = useI18n();

  return (
    <div className={styles.page}>
      <h1 className={styles.pageTitle}>{t('settings.title')}</h1>

      <div className={styles.card} style={{ marginBottom: 16, maxWidth: 600 }}>
        <h3 style={{ fontSize: 15, fontWeight: 600, marginBottom: 16 }}>{t('settings.profile')}</h3>
        <div className={styles.quotaRow}>
          <span className={styles.quotaRowLabel}>{t('settings.github_user')}</span>
          <span className={styles.quotaRowValue}>{user?.username || '-'}</span>
        </div>
      </div>

      <div className={styles.card} style={{ marginBottom: 16, maxWidth: 600 }}>
        <h3 style={{ fontSize: 15, fontWeight: 600, marginBottom: 16 }}>{t('settings.theme')}</h3>
        <div className={styles.quotaRow}>
          <span className={styles.quotaRowLabel}>{t('settings.theme')}</span>
          <div style={{ display: 'flex', gap: 8 }}>
            <button
              className={`${styles.btn} ${theme === 'light' ? styles.btnPrimary : ''}`}
              onClick={() => theme !== 'light' && toggleTheme()}
            >
              {t('settings.theme_light')}
            </button>
            <button
              className={`${styles.btn} ${theme === 'dark' ? styles.btnPrimary : ''}`}
              onClick={() => theme !== 'dark' && toggleTheme()}
            >
              {t('settings.theme_dark')}
            </button>
          </div>
        </div>
      </div>

      <div className={styles.card} style={{ marginBottom: 16, maxWidth: 600 }}>
        <h3 style={{ fontSize: 15, fontWeight: 600, marginBottom: 16 }}>{t('settings.language')}</h3>
        <div className={styles.quotaRow}>
          <span className={styles.quotaRowLabel}>{t('settings.language')}</span>
          <div style={{ display: 'flex', gap: 8 }}>
            <button
              className={`${styles.btn} ${locale === 'zh' ? styles.btnPrimary : ''}`}
              onClick={() => setLocale('zh')}
            >
              {t('settings.lang_zh')}
            </button>
            <button
              className={`${styles.btn} ${locale === 'en' ? styles.btnPrimary : ''}`}
              onClick={() => setLocale('en')}
            >
              {t('settings.lang_en')}
            </button>
          </div>
        </div>
      </div>

      <div className={styles.card} style={{ maxWidth: 600 }}>
        <h3 style={{ fontSize: 15, fontWeight: 600, marginBottom: 16 }}>{t('settings.version')}</h3>
        <div className={styles.quotaRow}>
          <span className={styles.quotaRowLabel}>{t('settings.version')}</span>
          <span className={styles.quotaRowValue}>0.1.0</span>
        </div>
      </div>
    </div>
  );
}
