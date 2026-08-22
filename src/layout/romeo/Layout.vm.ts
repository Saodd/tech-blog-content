import { useMemo } from 'react';
// import { useLocation } from 'wouter';


export interface NavItem {
  label: string;
  href?: string;
  onClick?: (e: React.MouseEvent<HTMLAnchorElement, MouseEvent>) => void;
}

type ILayoutVM = {
  navItems: NavItem[];
  privacyPolicyItem: NavItem;
};

// Astro项目使用
export function useLayoutVM_Astro(): ILayoutVM {
  return useMemo(() => ({
    navItems: [
      { label: '文章', href: '/blog/list' },
      { label: '影评', href: '/bangumi/list' },
      { label: '关于', href: '/about/me' },
    ],
    privacyPolicyItem: { label: 'Privacy Policy', href: '/about/privacy-policy' },
  }), []);
}
