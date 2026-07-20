import { NavLink } from 'react-router-dom';
import { useI18n } from '@/context/I18nContext';
import styles from '@/styles/components.module.css';

const navItems = [
  { to: '/', label: 'nav.dashboard', icon: '◫' },
  { to: '/files', label: 'nav.files', icon: '◰' },
  { to: '/tenants', label: 'nav.tenants', icon: '◷' },
  { to: '/quotas', label: 'nav.quotas', icon: '◴' },
  { to: '/settings', label: 'nav.settings', icon: '⚙' },
];

export default function Sidebar() {
  const { t } = useI18n();

  return (
    <aside className={styles.sidebar}>
      <div className={styles.sidebarLogo}>{t('app.name')}</div>
      <nav className={styles.sidebarNav}>
        {navItems.map((item) => (
          <NavLink
            key={item.to}
            to={item.to}
            end={item.to === '/'}
            className={({ isActive }) =>
              `${styles.sidebarLink} ${isActive ? styles.sidebarLinkActive : ''}`
            }
          >
            <span className={styles.sidebarIcon}>{item.icon}</span>
            {t(item.label)}
          </NavLink>
        ))}
      </nav>
    </aside>
  );
}
