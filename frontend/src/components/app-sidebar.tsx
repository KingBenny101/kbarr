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
} from '@/components/ui/sidebar'
import { Library, Search } from 'lucide-react'

interface AppSidebarProps {
  onNavigate: (view: 'list' | 'search') => void
  currentView: 'list' | 'search'
  version: string
}

export function AppSidebar({ onNavigate, currentView, version }: AppSidebarProps) {
  return (
    <Sidebar>
      <SidebarHeader>
        <SidebarMenu>
          <SidebarMenuItem>
            <div className="flex justify-center gap-2 px-2 py-2">
              <img src="/kbarr.svg" alt="kbarr logo" className="w-10 h-10" />
              <h1 className="text-3xl font-bold">kbarr</h1>
            </div>
          </SidebarMenuItem>
        </SidebarMenu>
      </SidebarHeader>
      <SidebarContent>
        <SidebarGroup>
          <SidebarGroupContent>
            <SidebarMenu>
              <SidebarMenuItem>
                <SidebarMenuButton
                  onClick={() => onNavigate('list')}
                  isActive={currentView === 'list'}
                >
                  <Library />
                  <span>Library</span>
                </SidebarMenuButton>
              </SidebarMenuItem>
              <SidebarMenuItem>
                <SidebarMenuButton
                  onClick={() => onNavigate('search')}
                  isActive={currentView === 'search'}
                >
                  <Search />
                  <span>Search</span>
                </SidebarMenuButton>
              </SidebarMenuItem>
            </SidebarMenu>
          </SidebarGroupContent>
        </SidebarGroup>
      </SidebarContent>
      <SidebarFooter>
        <SidebarMenu>
          <SidebarMenuItem>
            <div className="text-xs text-center text-muted-foreground py-2">
              v{version}
            </div>
          </SidebarMenuItem>
        </SidebarMenu>
      </SidebarFooter>
    </Sidebar>
  )
}
