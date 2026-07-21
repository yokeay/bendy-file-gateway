import { useEffect, useState, useCallback } from 'react';
import { useI18n } from '@/context/I18nContext';
import { useToast } from '@/components/ui/Toast';
import { api, type TenantQuota, formatBytes, formatDate } from '@/lib/api';
import styles from '@/styles/components.module.css';

export default function Quotas() {
  const { t } = useI18n();
  const { toast } = useToast();
  const [quotas, setQuotas] = useState<TenantQuota[]>([]);
  const [loading, setLoading] = useState(true);

  const load = useCallback(() => {
    api.listQuotas().then(setQuotas).finally(() => setLoading(false));
  }, []);

  useEffect(() => { load(); }, [load]);

  const updateField = async (quota: TenantQuota, field: string, value: number) => {
    try {
      await api.updateQuota(quota.id, { [field]: value });
      toast(t('quotas.updated'), 'success');
      load();
    } catch (err) {
      toast(err instanceof Error ? err.message : t('toast.error'), 'error');
    }
  };

  const handleBlur = (quota: TenantQuota, field: string) => (e: React.FocusEvent<HTMLInputElement>) => {
    const newVal = parseInt(e.target.value, 10);
    if (isNaN(newVal) || newVal < 0) return;
    const oldVal = (quota as unknown as Record<string, unknown>)[field] as number;
    if (newVal !== oldVal) {
      updateField(quota, field, newVal);
    }
  };

  const handleKeyDown = (_q: TenantQuota, _field: string) => (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Enter') {
      (e.target as HTMLInputElement).blur();
    }
  };

  if (loading) return <div className={styles.loading}>{t('common.loading')}</div>;

  return (
    <div className={styles.page}>
      <h1 className={styles.pageTitle}>{t('quotas.title')}</h1>

      {quotas.length === 0 ? (
        <div className={styles.emptyState}>
          <div className={styles.emptyStateText}>{t('common.none')}</div>
        </div>
      ) : (
        <table className={styles.table}>
          <thead>
            <tr>
              <th>{t('quotas.tenant')}</th>
              <th>{t('quotas.traffic_used')}</th>
              <th>{t('quotas.traffic_limit')}</th>
              <th>{t('quotas.api_calls_used')}</th>
              <th>{t('quotas.api_calls_limit')}</th>
              <th>{t('quotas.storage_used')}</th>
              <th>{t('quotas.storage_limit')}</th>
              <th>{t('quotas.expiry')}</th>
            </tr>
          </thead>
          <tbody>
            {quotas.map((q) => (
              <tr key={q.id}>
                <td style={{ fontWeight: 500 }}>{q.tenant_name}</td>
                <td>{formatBytes(q.traffic_used)}</td>
                <td>
                  <input
                    type="number"
                    className={styles.formInput}
                    defaultValue={q.traffic_limit}
                    onBlur={handleBlur(q, 'traffic_limit')}
                    onKeyDown={handleKeyDown(q, 'traffic_limit')}
                    style={{ width: 100 }}
                    min={0}
                  />
                </td>
                <td>{q.api_calls_used.toLocaleString()}</td>
                <td>
                  <input
                    type="number"
                    className={styles.formInput}
                    defaultValue={q.api_calls_limit}
                    onBlur={handleBlur(q, 'api_calls_limit')}
                    onKeyDown={handleKeyDown(q, 'api_calls_limit')}
                    style={{ width: 100 }}
                    min={0}
                  />
                </td>
                <td>{formatBytes(q.storage_used)}</td>
                <td>
                  <input
                    type="number"
                    className={styles.formInput}
                    defaultValue={q.storage_limit}
                    onBlur={handleBlur(q, 'storage_limit')}
                    onKeyDown={handleKeyDown(q, 'storage_limit')}
                    style={{ width: 100 }}
                    min={0}
                  />
                </td>
                <td>{q.expires_at ? formatDate(q.expires_at) : t('common.never')}</td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </div>
  );
}
