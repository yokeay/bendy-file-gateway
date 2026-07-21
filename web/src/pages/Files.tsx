import { useEffect, useState, useCallback } from 'react';
import { useI18n } from '@/context/I18nContext';
import { useToast } from '@/components/ui/Toast';
import Modal from '@/components/ui/Modal';
import { api, type FileInfo, formatBytes, formatDate } from '@/lib/api';
import styles from '@/styles/components.module.css';

export default function Files() {
  const { t } = useI18n();
  const { toast } = useToast();
  const [files, setFiles] = useState<FileInfo[]>([]);
  const [loading, setLoading] = useState(true);
  const [prefix, setPrefix] = useState('');
  const [searchInput, setSearchInput] = useState('');
  const [confirmDelete, setConfirmDelete] = useState<FileInfo | null>(null);

  const load = useCallback(() => {
    setLoading(true);
    api
      .listFiles(prefix)
      .then((data) => setFiles(data.files || []))
      .catch(() => setFiles([]))
      .finally(() => setLoading(false));
  }, [prefix]);

  useEffect(() => { load(); }, [load]);

  const handleSearch = () => {
    setPrefix(searchInput);
  };

  const handleDelete = async () => {
    if (!confirmDelete) return;
    try {
      await api.deleteFile(confirmDelete.key);
      toast(t('files.deleted'), 'success');
      setConfirmDelete(null);
      load();
    } catch (err) {
      toast(err instanceof Error ? err.message : t('toast.error'), 'error');
    }
  };

  return (
    <div className={styles.page}>
      <div className={styles.pageHeader}>
        <h1 className={styles.pageTitle}>{t('files.title')}</h1>
      </div>

      <div style={{ display: 'flex', gap: 8, marginBottom: 24 }}>
        <input
          className={styles.formInput}
          style={{ maxWidth: 320 }}
          placeholder={t('files.prefix')}
          value={searchInput}
          onChange={(e) => setSearchInput(e.target.value)}
          onKeyDown={(e) => e.key === 'Enter' && handleSearch()}
        />
        <button className={`${styles.btn} ${styles.btnPrimary}`} onClick={handleSearch}>
          {t('files.browse')}
        </button>
        {prefix && (
          <button className={styles.btn} onClick={() => { setPrefix(''); setSearchInput(''); }}>
            {t('common.back')}
          </button>
        )}
      </div>

      {loading ? (
        <div className={styles.loading}>{t('common.loading')}</div>
      ) : files.length === 0 ? (
        <div className={styles.emptyState}>
          <div className={styles.emptyStateText}>{t('files.no_files')}</div>
        </div>
      ) : (
        <table className={styles.table}>
          <thead>
            <tr>
              <th>{t('common.key')}</th>
              <th>{t('common.size')}</th>
              <th>{t('common.content_type')}</th>
              <th>{t('common.updated')}</th>
              <th>{t('common.actions')}</th>
            </tr>
          </thead>
          <tbody>
            {files.map((f) => (
              <tr key={f.key}>
                <td style={{ fontFamily: 'var(--font-mono)', fontSize: 13 }}>{f.key}</td>
                <td>{formatBytes(f.size)}</td>
                <td>{f.content_type}</td>
                <td>{formatDate(f.last_modified)}</td>
                <td>
                  <button className={`${styles.btn} ${styles.btnDanger}`} onClick={() => setConfirmDelete(f)}>
                    {t('common.delete')}
                  </button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}

      <Modal
        open={!!confirmDelete}
        title={t('common.delete')}
        onClose={() => setConfirmDelete(null)}
        footer={
          <>
            <button className={styles.btn} onClick={() => setConfirmDelete(null)}>{t('common.cancel')}</button>
            <button className={`${styles.btn} ${styles.btnDanger}`} onClick={handleDelete}>{t('common.delete')}</button>
          </>
        }
      >
        <p className={styles.confirmText}>
          {t('files.delete_confirm').replace('{name}', confirmDelete?.key || '')}
        </p>
      </Modal>
    </div>
  );
}
