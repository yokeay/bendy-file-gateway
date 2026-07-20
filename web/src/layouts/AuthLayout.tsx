import { Outlet } from 'react-router-dom';
import styles from '@/styles/layouts.module.css';

export default function AuthLayout() {
  return (
    <div className={styles.authWrapper}>
      <Outlet />
    </div>
  );
}
