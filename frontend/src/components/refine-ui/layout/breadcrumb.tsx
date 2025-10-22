'use client';

import {
  Breadcrumb as ShadcnBreadcrumb,
  BreadcrumbItem as ShadcnBreadcrumbItem,
  BreadcrumbList as ShadcnBreadcrumbList,
  BreadcrumbPage as ShadcnBreadcrumbPage,
  BreadcrumbSeparator as ShadcnBreadcrumbSeparator,
} from '@/components/ui/breadcrumb';
import {
  matchResourceFromRoute,
  useBreadcrumb,
  useGo,
  useResourceParams,
} from '@refinedev/core';
import { Home } from 'lucide-react';
import { Fragment, useMemo } from 'react';

export function Breadcrumb() {
  const go = useGo();
  const { breadcrumbs } = useBreadcrumb();
  const { resources } = useResourceParams();
  const rootRouteResource = matchResourceFromRoute('/', resources);

  const breadCrumbItems = useMemo(() => {
    const list: {
      key: string;
      href: string;
      Component: React.ReactNode;
    }[] = [];

    list.push({
      key: 'breadcrumb-item-home',
      href: rootRouteResource.matchedRoute ?? '/',
      Component: (
        <button
          type="button"
          onClick={() => go({ to: rootRouteResource.matchedRoute ?? '/' })}
          className="flex cursor-pointer items-center border-none bg-transparent p-0 transition-colors hover:text-foreground"
        >
          {rootRouteResource?.resource?.meta?.icon ?? (
            <Home className="h-4 w-4" />
          )}
        </button>
      ),
    });

    for (const { label, href } of breadcrumbs) {
      list.push({
        key: `breadcrumb-item-${label}`,
        href: href ?? '',
        Component: href ? (
          <button
            type="button"
            onClick={() => go({ to: href })}
            className="cursor-pointer border-none bg-transparent p-0 transition-colors hover:text-foreground"
          >
            {label}
          </button>
        ) : (
          <span>{label}</span>
        ),
      });
    }

    return list;
  }, [breadcrumbs, go, rootRouteResource]);

  return (
    <ShadcnBreadcrumb>
      <ShadcnBreadcrumbList>
        {breadCrumbItems.map((item, index) => {
          if (index === breadCrumbItems.length - 1) {
            return (
              <ShadcnBreadcrumbPage key={item.key}>
                {item.Component}
              </ShadcnBreadcrumbPage>
            );
          }

          return (
            <Fragment key={item.key}>
              <ShadcnBreadcrumbItem key={item.key}>
                {item.Component}
              </ShadcnBreadcrumbItem>
              <ShadcnBreadcrumbSeparator />
            </Fragment>
          );
        })}
      </ShadcnBreadcrumbList>
    </ShadcnBreadcrumb>
  );
}

Breadcrumb.displayName = 'Breadcrumb';
