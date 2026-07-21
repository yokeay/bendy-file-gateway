import { useEffect, useState, useCallback } from 'react';
import { useI18n } from '@/context/I18nContext';
import { useToast } from '@/components/ui/Toast';
import Modal from '@/components/ui/Modal';
import { api, type Tenant, formatDate } from '@/lib/api';
import styles from '@/styles/components.module.css';

const emptyTenant = (): Partial<Tenant> => ({
  name: '',
  backend: 's3',
  backend_config: '{}',
  status: 'active',
});

const BACKENDS = ['s3', 'aliyun_oss', 'huawei_obs', 'qiniu_kodo', 'tencent_cos', 'tianyi_oos', 'unicom_oss', 'redis', 'postgres', 'mysql'];

export default function Tenants() {
  const { t } = useI18n();
  const { toast } = useToast();
  const [tenants, setTenants] = useState<Tenant[]>([]);
  const [loading, setLoading] = useState(true);
  const [modalOpen, setModalOpen] = useState(false);
  const [editing, setEditing] = useState<Partial<Tenant> | null>(null);
  const [saving, setSaving] = useState(false);
  const [confirmDelete, setConfirmDelete] = useState<Tenant | null>(null);

  const load = useCallback(() => {
    api.listTenants().then(setTenants).finally(() => setLoading(false));
  }, []);

  useEffect(() => { load(); }, [load]);

  const openCreate = () => {
    setEditing(emptyTenant());
    setModalOpen(true);
  };

  const openEdit = (tenant: Tenant) => {
    setEditing({
      ...tenant,
      backend_config: typeof tenant.backend_config === 'string' ? tenant.backend_config : JSON.stringify(tenant.backend_config, null, 2),
    });
    setModalOpen(true);
  };

  const handleSave = async () => {
    if (!editing) return;
    setSaving(true);
    try {
      if (editing.id) {
        await api.updateTenant(editing.id, editing);
        toast(t('tenants.updated'), 'success');
      } else {
        await api.createTenant(editing);
        toast(t('tenants.created'), 'success');
      }
      setModalOpen(false);
      load();
    } catch (err) {
      toast(err instanceof Error ? err.message : t('toast.error'), 'error');
    } finally {
      setSaving(false);
    }
  };

  const handleDelete = async () => {
    if (!confirmDelete) return;
    try {
      await api.deleteTenant(confirmDelete.id);
      toast(t('tenants.deleted'), 'success');
      setConfirmDelete(null);
      load();
    } catch (err) {
      toast(err instanceof Error ? err.message : t('toast.error'), 'error');
    }
  };

  if (loading) return <div className={styles.loading}>{t('common.loading')}</div>;

  return (
    <div className={styles.page}>
      <div className={styles.pageHeader}>
        <h1 className={styles.pageTitle}>{t('tenants.title')}</h1>
        <button className={`${styles.btn} ${styles.btnPrimary}`} onClick={openCreate}>
          + {t('tenants.create')}
        </button>
      </div>

      {tenants.length === 0 ? (
        <div className={styles.emptyState}>
          <div className={styles.emptyStateText}>{t('common.none')}</div>
        </div>
      ) : (
        <table className={styles.table}>
          <thead>
            <tr>
              <th>{t('tenants.name')}</th>
              <th>{t('tenants.backend')}</th>
              <th>{t('common.status')}</th>
              <th>{t('tenants.expiry')}</th>
              <th>{t('common.created')}</th>
              <th>{t('common.actions')}</th>
            </tr>
          </thead>
          <tbody>
            {tenants.map((tenant) => (
              <tr key={tenant.id}>
                <td style={{ fontWeight: 500 }}>{tenant.name}</td>
                <td>{tenant.backend}</td>
                <td>
                  <span className={`${styles.badge} ${tenant.status === 'active' ? styles.badgeActive : styles.badgeInactive}`}>
                    {tenant.status === 'active' ? t('common.active') : t('common.inactive')}
                  </span>
                </td>
                <td>{tenant.expires_at ? formatDate(tenant.expires_at) : t('common.never')}</td>
                <td>{formatDate(tenant.created_at)}</td>
                <td>
                  <button className={styles.btn} onClick={() => openEdit(tenant)}>{t('common.edit')}</button>
                  <button className={`${styles.btn} ${styles.btnDanger}`} onClick={() => setConfirmDelete(tenant)}>{t('common.delete')}</button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}

      {/* Create/Edit Modal */}
      <Modal
        open={modalOpen}
        title={editing?.id ? t('tenants.edit') : t('tenants.create')}
        onClose={() => setModalOpen(false)}
        footer={
          <>
            <button className={styles.btn} onClick={() => setModalOpen(false)}>{t('common.cancel')}</button>
            <button className={`${styles.btn} ${styles.btnPrimary}`} onClick={handleSave} disabled={saving}>
              {saving ? <span className={styles.spinner} /> : t('common.save')}
            </button>
          </>
        }
      >
        <div className={styles.formGroup}>
          <label className={styles.formLabel}>{t('tenants.name')}</label>
          <input
            className={styles.formInput}
            value={editing?.name || ''}
            onChange={(e) => setEditing((prev) => prev ? { ...prev, name: e.target.value } : null)}
          />
        </div>
        <div className={styles.formGroup}>
          <label className={styles.formLabel}>{t('tenants.backend')}</label>
          <select
            className={styles.formSelect}
            value={editing?.backend || 's3'}
            onChange={(e) => setEditing((prev) => prev ? { ...prev, backend: e.target.value } : null)}
          >
            {BACKENDS.map((b) => <option key={b} value={b}>{b}</option>)}
          </select>
        </div>
        <div className={styles.formGroup}>
          <label className={styles.formLabel}>{t('tenants.status')}</label>
          <select
            className={styles.formSelect}
            value={editing?.status || 'active'}
            onChange={(e) => setEditing((prev) => prev ? { ...prev, status: e.target.value } : null)}
          >
            <option value="active">{t('common.active')}</option>
            <option value="inactive">{t('common.inactive')}</option>
          </select>
        </div>
        <div className={styles.formGroup}>
          <label className={styles.formLabel}>{t('tenants.expiry')}</label>
          <input
            type="date"
            className={styles.formInput}
            value={editing?.expires_at?.slice(0, 10) || ''}
            onChange={(e) => setEditing((prev) => prev ? { ...prev, expires_at: e.target.value + 'T00:00:00Z' } : null)}
          />
        </div>
        <div className={styles.formGroup}>
          <label className={styles.formLabel}>{t('tenants.backend_config')}</label>
          <label className={styles.formLabel} style={{ textTransform: 'none', fontWeight: 400, fontSize: 11 }}>{t('tenants.config_help')}</label>
          <textarea
            className={styles.formTextarea}
            value={editing?.backend_config || ''}
            onChange={(e) => setEditing((prev) => prev ? { ...prev, backend_config: e.target.value } : null)}
          />
        </div>
      </Modal>

      {/* Delete Confirmation */}
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
          {t('tenants.delete_confirm').replace('{name}', confirmDelete?.name || '')}
        </p>
      </Modal>
    </div>
  );
}
