import { useAuth } from '@/context/AuthContext';
import { useTheme } from '@/context/ThemeContext';
import { useI18n } from '@/context/I18nContext';
import styles from '@/styles/components.module.css';

export default function Header() {
  const { user, logout } = useAuth();
  const { theme, toggleTheme } = useTheme();
  const { locale, setLocale, t } = useI18n();

  return (
    <header className={styles.header}>
      <div className={styles.headerSpacer} />
      <div className={styles.headerActions}>
        <button
          className={styles.headerBtn}
          onClick={() => setLocale(locale === 'zh' ? 'en' : 'zh')}
          title={t('settings.language')}
        >
          {locale === 'zh' ? 'EN' : '中文'}
        </button>
        <button
          className={styles.headerBtn}
          onClick={toggleTheme}
          title={t('settings.theme')}
        >
          {theme === 'light' ? '◑' : '◐'}
        </button>
        {user && (
          <>
            <span className={styles.headerUser}>{user.username}</span>
            <button className={styles.headerBtn} onClick={logout}>
              {t('auth.logout')}
            </button>
          </>
        )}
      </div>
    </header>
  );
}
