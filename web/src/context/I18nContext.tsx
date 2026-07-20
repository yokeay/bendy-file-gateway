import { createContext, useContext, useState, useCallback, type ReactNode } from 'react';
import zh from '@/locales/zh.json';
import en from '@/locales/en.json';

type Locale = 'zh' | 'en';

const messages: Record<Locale, Record<string, string>> = { zh, en };

interface I18nState {
  locale: Locale;
  t: (key: string) => string;
  setLocale: (l: Locale) => void;
}

const I18nContext = createContext<I18nState>({
  locale: 'zh',
  t: (key: string) => key,
  setLocale: () => {},
});

export function I18nProvider({ children }: { children: ReactNode }) {
  const [locale, setLocale] = useState<Locale>(() => {
    const stored = localStorage.getItem('bendy-locale');
    if (stored === 'zh' || stored === 'en') return stored;
    return navigator.language.startsWith('zh') ? 'zh' : 'en';
  });

  const t = useCallback(
    (key: string) => messages[locale][key] || key,
    [locale],
  );

  const changeLocale = useCallback((l: Locale) => {
    setLocale(l);
    localStorage.setItem('bendy-locale', l);
  }, []);

  return (
    <I18nContext.Provider value={{ locale, t, setLocale: changeLocale }}>
      {children}
    </I18nContext.Provider>
  );
}

export function useI18n() {
  return useContext(I18nContext);
}
