import { getCurrencyConfig } from './render';

export const getQuotaPerUnit = () => {
  const raw = parseFloat(localStorage.getItem('quota_per_unit') || '1');
  return Number.isFinite(raw) && raw > 0 ? raw : 1;
};

export const quotaToDisplayAmount = (quota) => {
  const q = Number(quota || 0);
  if (!Number.isFinite(q) || q <= 0) return 0;
  return q / getQuotaPerUnit();
};

export const displayAmountToQuota = (amount) => {
  const val = Number(amount || 0);
  if (!Number.isFinite(val) || val <= 0) return 0;
  return Math.round(val * getQuotaPerUnit());
};
