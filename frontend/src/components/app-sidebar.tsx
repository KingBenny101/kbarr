import { useEffect, useState } from 'react';
import { API_URL } from "@/lib/api";
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarGroup,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarRail,
} from '@/components/ui/sidebar';
import { Library, Search, Settings, Globe, Database, Activity, ListTodo } from 'lucide-react';
import { Link, useLocation } from 'react-router-dom';

export function AppSidebar() {
  const location = useLocation();
  const [version, setVersion] = useState('0.0.0');

  useEffect(() => {
    fetch(`${API_URL}/api/version`)
      .then((res) => {
        if (!res.ok) throw new Error('Failed to fetch version');
        return res.json();
      })
      .then((data) => setVersion(data.version))
      .catch(() => setVersion('0.0.0'));
  }, []);

  return (
    <Sidebar collapsible="icon" className="overflow-x-hidden">
      <SidebarHeader>
        <SidebarMenu>
          <SidebarMenuItem>
            <SidebarMenuButton
              size="lg"
              asChild
              className="h-14 overflow-hidden p-0 group-data-[state=collapsed]:justify-center hover:bg-transparent"
            >
              <Link
                to="/"
                className="flex items-center gap-3 overflow-hidden px-3 group-data-[state=collapsed]:px-0"
              >
                <img src="/kbarr.svg" alt="logo" className="size-7 shrink-0" />
                <span className="truncate text-2xl font-bold tracking-tighter group-data-[state=collapsed]:hidden">
                  kbarr
                </span>
              </Link>
            </SidebarMenuButton>
          </SidebarMenuItem>
        </SidebarMenu>
      </SidebarHeader>

      <SidebarContent className='overflow-x-hidden'>
        <SidebarGroup>
          <SidebarGroupLabel>Media</SidebarGroupLabel>
          <SidebarGroupContent>
            <SidebarMenu>
              <SidebarMenuItem>
                <SidebarMenuButton
                  asChild
                  isActive={location.pathname === '/search'}
                  tooltip="Search Anime"
                >
                  <Link to="/search">
                    <Search />
                    <span>Search</span>
                  </Link>
                </SidebarMenuButton>
              </SidebarMenuItem>

              <SidebarMenuItem>
                <SidebarMenuButton
                  asChild
                  isActive={location.pathname === '/' || location.pathname.startsWith('/media')}
                  tooltip="Digital Library"
                >
                  <Link to="/">
                    <Library />
                    <span>Library</span>
                  </Link>
                </SidebarMenuButton>
              </SidebarMenuItem>

              <SidebarMenuItem>
                <SidebarMenuButton
                  asChild
                  isActive={location.pathname === '/monitored'}
                  tooltip="Monitored Items"
                >
                  <Link to="/monitored">
                    <Library className="rotate-90" />
                    <span>Monitored</span>
                  </Link>
                </SidebarMenuButton>
              </SidebarMenuItem>
            </SidebarMenu>
          </SidebarGroupContent>
        </SidebarGroup>

        <SidebarGroup>
          <SidebarGroupLabel>Configuration</SidebarGroupLabel>
          <SidebarGroupContent>
            <SidebarMenu>
              <SidebarMenuItem>
                <SidebarMenuButton asChild isActive={location.pathname === '/settings/general'} tooltip="General Settings">
                  <Link to="/settings/general">
                    <Settings />
                    <span>General</span>
                  </Link>
                </SidebarMenuButton>
              </SidebarMenuItem>
              <SidebarMenuItem>
                <SidebarMenuButton asChild isActive={location.pathname === '/settings/anidb'} tooltip="AniDB Integration">
                  <Link to="/settings/anidb">
                    <Globe />
                    <span>AniDB</span>
                  </Link>
                </SidebarMenuButton>
              </SidebarMenuItem>
              <SidebarMenuItem>
                <SidebarMenuButton asChild isActive={location.pathname === '/settings/prowlarr'} tooltip="Prowlarr Indexer">
                  <Link to="/settings/prowlarr">
                    <Database />
                    <span>Prowlarr</span>
                  </Link>
                </SidebarMenuButton>
              </SidebarMenuItem>
            </SidebarMenu>
          </SidebarGroupContent>
        </SidebarGroup>

        <SidebarGroup>
          <SidebarGroupLabel>System</SidebarGroupLabel>
          <SidebarGroupContent>
            <SidebarMenu>
              <SidebarMenuItem>
                <SidebarMenuButton
                  asChild
                  isActive={location.pathname === '/search-queue'}
                  tooltip="Search Queue"
                >
                  <Link to="/search-queue">
                    <ListTodo />
                    <span>Search Queue</span>
                  </Link>
                </SidebarMenuButton>
              </SidebarMenuItem>
              <SidebarMenuItem>
                <SidebarMenuButton
                  asChild
                  isActive={location.pathname === '/workers'}
                  tooltip="Background Workers"
                >
                  <Link to="/workers">
                    <Activity />
                    <span>Workers</span>
                  </Link>
                </SidebarMenuButton>
              </SidebarMenuItem>
            </SidebarMenu>
          </SidebarGroupContent>
        </SidebarGroup>
      </SidebarContent>

      <SidebarFooter className="group-data-[collapsible=icon]:hidden">
        <SidebarMenu>
          <SidebarMenuItem>
            <SidebarMenuButton size="sm" className="pointer-events-none justify-center opacity-50">
              <span className="truncate text-[10px] uppercase tracking-widest">Version {version}</span>
            </SidebarMenuButton>
          </SidebarMenuItem>
        </SidebarMenu>
      </SidebarFooter>
      <SidebarRail />
    </Sidebar>
  );
}
