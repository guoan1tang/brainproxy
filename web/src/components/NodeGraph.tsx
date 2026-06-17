import { useMemo } from 'react';
import { motion } from 'framer-motion';
import { BrainCore } from './BrainCore';
import { FunctionNode } from './FunctionNode';
import type { ToolNode } from '../types';
import './NodeGraph.css';

interface NodeGraphProps {
  tools: ToolNode[];
  activeTools: Set<string>;
  totalRequests: number;
  totalTokens: number;
  isBrainActive: boolean;
  onNodeClick?: (name: string) => void;
}

// Colors by category
const CATEGORY_COLORS: Record<string, string> = {
  tool: '#6366f1',   // indigo
  mcp: '#a855f7',    // purple
  skill: '#10b981',  // green
};

function getNodeColor(node: ToolNode): string {
  if (node.category === 'tool') {
    // Specific tool colors
    if (node.name.includes('search')) return '#10b981';
    if (node.name.includes('code') || node.name.includes('execute')) return '#f59e0b';
    if (node.name.includes('Read') || node.name.includes('file_read')) return '#3b82f6';
    if (node.name.includes('Write') || node.name.includes('file_write')) return '#ef4444';
    if (node.name.includes('Edit')) return '#f97316';
    if (node.name.includes('Bash')) return '#eab308';
    if (node.name.includes('Agent')) return '#ec4899';
    if (node.name.includes('browser')) return '#06b6d4';
  }
  return CATEGORY_COLORS[node.category] || CATEGORY_COLORS.tool;
}

function getNodeIcon(node: ToolNode): string {
  if (node.category === 'mcp') return '🔌';
  if (node.category === 'skill') return '✨';
  // Specific tool icons
  if (node.name.includes('search')) return '🔍';
  if (node.name.includes('code') || node.name.includes('execute')) return '⚡';
  if (node.name.includes('Read') || node.name.includes('file_read')) return '📖';
  if (node.name.includes('Write') || node.name.includes('file_write')) return '📝';
  if (node.name.includes('Edit')) return '✏️';
  if (node.name.includes('Bash')) return '💻';
  if (node.name.includes('Agent')) return '🤖';
  if (node.name.includes('browser')) return '🌐';
  return '🔧';
}

function getNodeKey(node: ToolNode): string {
  return `${node.category}:${node.name}`;
}

function getDisplayLabel(node: ToolNode): string {
  if (node.category === 'mcp') return `${node.name} (MCP)`;
  if (node.category === 'skill') return node.name;
  return node.name;
}

export function NodeGraph({
  tools,
  activeTools,
  totalRequests,
  totalTokens,
  isBrainActive,
  onNodeClick,
}: NodeGraphProps) {
  const nodePositions = useMemo(() => {
    const count = tools.length;
    // Adaptive layout: larger radius and smaller nodes when many tools
    const radius = count > 50 ? 260 : count > 20 ? 240 : 220;
    const centerX = 300;
    const centerY = 280;
    return tools.map((_tool, index) => {
      const angle = (2 * Math.PI * index) / Math.max(count, 1) - Math.PI / 2;
      return {
        x: centerX + radius * Math.cos(angle),
        y: centerY + radius * Math.sin(angle),
      };
    });
  }, [tools]);

  return (
    <div className="node-graph">
      <svg className="node-connections" viewBox="0 0 600 560">
        {tools.map((tool, index) => {
          const pos = nodePositions[index];
          const key = getNodeKey(tool);
          const isActive = activeTools.has(key);
          const color = getNodeColor(tool);
          return (
            <g key={key}>
              <motion.line
                x1={300}
                y1={280}
                x2={pos.x}
                y2={pos.y}
                stroke={isActive ? color : '#334155'}
                strokeWidth={isActive ? 2 : 1}
                initial={{ pathLength: 0 }}
                animate={{
                  pathLength: 1,
                  opacity: isActive ? 0.8 : 0.15,
                }}
                transition={{ duration: 0.5 }}
              />
              {isActive && (
                <motion.circle
                  r={3}
                  fill={color}
                  initial={{ cx: 300, cy: 280 }}
                  animate={{
                    cx: [300, pos.x, 300],
                    cy: [280, pos.y, 280],
                  }}
                  transition={{
                    duration: 2,
                    repeat: Infinity,
                    ease: 'easeInOut',
                  }}
                />
              )}
            </g>
          );
        })}
      </svg>

      <div className="brain-position" style={{ left: 300, top: 280 }}>
        <BrainCore
          totalRequests={totalRequests}
          totalTokens={totalTokens}
          isActive={isBrainActive}
        />
      </div>

      {tools.map((tool, index) => {
        const pos = nodePositions[index];
        const key = getNodeKey(tool);
        const isActive = activeTools.has(key);
        const compact = tools.length > 30;
        return (
          <div
            key={key}
            className={`node-position ${compact ? 'compact' : ''}`}
            style={{ left: pos.x, top: pos.y }}
            title={getDisplayLabel(tool)}
          >
            <FunctionNode
              name={compact ? '' : getDisplayLabel(tool)}
              icon={getNodeIcon(tool)}
              count={compact ? 0 : tool.count}
              isActive={isActive}
              color={getNodeColor(tool)}
              onClick={isActive && onNodeClick ? () => onNodeClick(tool.name) : undefined}
            />
          </div>
        );
      })}
    </div>
  );
}
