import { Outlet } from 'react-router-dom';
import Sidebar from '@/components/ui/Sidebar';
import Header from '@/components/ui/Header';
import styles from '@/styles/layouts.module.css';

export default function AdminLayout() {
  return (
    <div className={styles.wrapper}>
      <Sidebar />
      <div className={styles.main}>
        <Header />
        <main className={styles.content}>
          <Outlet />
        </main>
      </div>
    </div>
  );
}
