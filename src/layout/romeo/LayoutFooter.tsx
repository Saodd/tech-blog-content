import type { FC } from 'react';
import { ShieldCheck } from 'lucide-react';
import { type NavItem } from './Layout.vm';

interface LayoutFooterProps {
  privacyPolicyItem: NavItem;
}

export const LayoutFooter: FC<LayoutFooterProps> = ({ privacyPolicyItem }) => {
  return (
    <footer className="border-t border-border bg-card">
      <div className="mx-auto flex min-h-16 w-full max-w-6xl flex-wrap items-center justify-center gap-x-3 gap-y-2 px-4 py-5 text-center text-sm text-muted-foreground sm:px-6 lg:px-8">
        <span>©2019-2026 Lewin Lan</span>
        <Divider />
        <span>粤ICP备19078237号</span>
        <Divider />
        <a
          className="inline-flex items-center gap-1.5 transition duration-200 hover:text-foreground"
          href="https://www.beian.gov.cn/portal/registerSystemInfo?recordcode=44030402003189"
          target="_blank"
          rel="noreferrer"
        >
          <ShieldCheck className="size-4" />
          {'粤公网安备44030402003189号'}
        </a>
        <Divider />
        <a
          className="inline-flex items-center gap-1.5 transition duration-200 hover:text-foreground"
          href={privacyPolicyItem.href || '#'}
          onClick={privacyPolicyItem.onClick}
        >
          {privacyPolicyItem.label}
        </a>
      </div>
    </footer>
  );
};

const Divider: FC = () => {
  return <span className="hidden text-muted-foreground/40 md:inline">|</span>;
};
