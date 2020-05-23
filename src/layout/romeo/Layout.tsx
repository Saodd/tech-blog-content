import type { FC, PropsWithChildren } from 'react';
import { cn } from '../../lib/cn';
import { useLayoutVM_Astro } from './Layout.vm';
import { LayoutFooter } from './LayoutFooter';
import { LayoutHeader, LAYOUT_HEADER_HEIGHT_CLASS } from './LayoutHeader';

export const Layout: FC<PropsWithChildren> = ({ children }) => {
  const vm = useLayoutVM_Astro();
  return (
    <div className="bg-neutral-02 flex min-h-dvh flex-col text-foreground">
      <LayoutHeader navItems={vm.navItems} />
      <div aria-hidden="true" className={cn(LAYOUT_HEADER_HEIGHT_CLASS, 'shrink-0')} />
      <main className="flex-1">{children}</main>
      <LayoutFooter privacyPolicyItem={vm.privacyPolicyItem} />
    </div>
  );
};
