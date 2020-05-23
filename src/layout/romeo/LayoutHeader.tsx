import type { FC } from 'react';
import { cn } from '../../lib/cn';
import { Menu } from 'lucide-react';
import { type NavItem } from './Layout.vm';
import styles from './LayoutHeader.module.css';

export const LAYOUT_HEADER_HEIGHT_CLASS = 'h-16';

interface LayoutHeaderProps {
  navItems: NavItem[];
}

export const LayoutHeader: FC<LayoutHeaderProps> = ({ navItems }) => {
  return (
    <header className="fixed inset-x-0 top-0 z-50 border-b border-border bg-card/95 shadow-sm backdrop-blur">
      {/* 利用 checkbox 的 :checked 状态驱动移动端菜单展开，无需客户端 JS */}
      <input aria-controls="mobile-nav" className="peer sr-only" id="mobile-menu-toggle" type="checkbox" />
      <div
        className={cn(
          'mx-auto flex w-full max-w-6xl items-center justify-between px-4 sm:px-6 lg:px-8',
          LAYOUT_HEADER_HEIGHT_CLASS,
        )}
      >
        <a className={cn(styles.logo, 'text-2xl text-foreground')} href="/">
          {"Lewin's Blog"}
        </a>

        <nav className="hidden items-center gap-2 md:flex" aria-label="主导航">
          {navItems.map((item) => (
            <a
              className="rounded-md px-3 py-2 text-sm font-medium text-muted-foreground transition-colors hover:bg-accent hover:text-accent-foreground"
              href={item.href || '#'}
              key={item.label}
              onClick={item.onClick}
            >
              {item.label}
            </a>
          ))}
        </nav>

        <label
          aria-label="导航菜单"
          className="inline-flex size-9 cursor-pointer items-center justify-center rounded-md border border-border bg-card text-foreground shadow-sm transition-colors hover:bg-accent hover:text-accent-foreground md:hidden"
          htmlFor="mobile-menu-toggle"
        >
          <Menu aria-hidden="true" className="size-4" />
        </label>
      </div>

      <nav
        aria-label="移动端主导航"
        className="hidden border-t border-border bg-card px-4 py-2 shadow-sm peer-checked:block md:hidden"
        id="mobile-nav"
      >
        <div className="mx-auto flex w-full max-w-6xl flex-col">
          {navItems.map((item) => (
            <a
              className="rounded-md px-3 py-3 text-sm font-medium text-muted-foreground transition-colors hover:bg-accent hover:text-accent-foreground"
              href={item.href || '#'}
              key={item.label}
              onClick={item.onClick}
            >
              {item.label}
            </a>
          ))}
        </div>
      </nav>
    </header>
  );
};
