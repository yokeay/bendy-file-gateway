import { useAuth } from '@/context/AuthContext';
import { useI18n } from '@/context/I18nContext';
import styles from '@/styles/components.module.css';

export default function Login() {
  const { login } = useAuth();
  const { t } = useI18n();

  return (
    <div className={styles.loginCard}>
      <h1 className={styles.loginTitle}>{t('app.name')}</h1>
      <p className={styles.loginSubtitle}>{t('auth.login')}</p>
      <button className={styles.loginBtn} onClick={login}>
        {t('auth.login_github')}
      </button>
    </div>
  );
}
