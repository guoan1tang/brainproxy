import { useMemo } from 'react';
import './ToolSidebar.css';

interface SidebarTool {
  name: string;
  category: 'tool' | 'mcp' | 'skill';
  isActive: boolean;
}

interface ToolSidebarProps {
  tools: SidebarTool[] | null;
}

const CATEGORY_LABELS: Record<string, string> = {
  tool: '🔧 Tools',
  mcp: '🔌 MCP',
  skill: '✨ Skills',
};

const CATEGORY_ICONS: Record<string, string> = {
  tool: '🔧',
  mcp: '🔌',
  skill: '✨',
};

function getToolIcon(name: string): string {
  if (name.includes('search')) return '🔍';
  if (name.includes('code') || name.includes('execute')) return '⚡';
  if (name.includes('Read') || name.includes('file_read')) return '📖';
  if (name.includes('Write') || name.includes('file_write')) return '📝';
  if (name.includes('Edit')) return '✏️';
  if (name.includes('Bash')) return '💻';
  if (name.includes('Agent')) return '🤖';
  if (name.includes('browser')) return '🌐';
  return '🔧';
}

function getDisplayName(name: string): string {
  // Clean MCP names: mcp__yunxiao__create_branch → yunxiao: create_branch
  if (name.startsWith('mcp__')) {
    const parts = name.split('__');
    if (parts.length >= 3) return `${parts[1]}: ${parts.slice(2).join('_')}`;
  }
  return name;
}

export function ToolSidebar({ tools }: ToolSidebarProps) {
  const groups = useMemo(() => {
    if (!tools) return null;
    const grouped: Record<string, SidebarTool[]> = { skill: [], mcp: [], tool: [] };
    for (const t of tools) {
      if (!grouped[t.category]) grouped[t.category] = [];
      grouped[t.category].push(t);
    }
    // Sort: active first, then alphabetical
    for (const cat of Object.keys(grouped)) {
      grouped[cat].sort((a, b) => {
        if (a.isActive !== b.isActive) return a.isActive ? -1 : 1;
        return a.name.localeCompare(b.name);
      });
    }
    return grouped;
  }, [tools]);

  if (!groups) return null;

  const activeCount = tools?.filter((t) => t.isActive).length || 0;
  const totalCount = tools?.length || 0;

  return (
    <div className="tool-sidebar">
      <div className="sidebar-header">
        <h3>Functions</h3>
        <span className="sidebar-count">{activeCount}/{totalCount} active</span>
      </div>
      <div className="sidebar-groups">
        {(['skill', 'mcp', 'tool'] as const).map((cat) => {
          const items = groups[cat];
          if (!items || items.length === 0) return null;
          return (
            <div key={cat} className="sidebar-group">
              <h4 className="sidebar-group-title">{CATEGORY_LABELS[cat]} ({items.length})</h4>
              <div className="sidebar-items">
                {items.map((item) => (
                  <div
                    key={item.name}
                    className={`sidebar-item ${item.isActive ? 'active' : ''}`}
                  >
                    <span className="sidebar-item-icon">
                      {item.category === 'tool' ? getToolIcon(item.name) : CATEGORY_ICONS[item.category]}
                    </span>
                    <span className="sidebar-item-name">{getDisplayName(item.name)}</span>
                    {item.isActive && <span className="sidebar-item-badge">●</span>}
                  </div>
                ))}
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
}
