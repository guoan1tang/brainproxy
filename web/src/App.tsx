import { useState, useCallback, useEffect, useMemo } from 'react';
import { Header } from './components/Header';
import { RequestList } from './components/RequestList';
import { NodeGraph } from './components/NodeGraph';
import { RequestDetail } from './components/RequestDetail';
import { ToolSidebar } from './components/ToolSidebar';
import { useWebSocket } from './hooks/useWebSocket';
import type { RequestEvent, WSEvent, ToolNode } from './types';
import './App.css';

function App() {
  const [events, setEvents] = useState<RequestEvent[]>([]);
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [selectedToolName, setSelectedToolName] = useState<string | null>(null);
  const [toolNodes, setToolNodes] = useState<Map<string, ToolNode>>(new Map());
  const [activeTools, setActiveTools] = useState<Set<string>>(new Set());
  const [isBrainActive, setIsBrainActive] = useState(false);

  // Helper: extract nodes from an event's analysis
  const extractNodes = useCallback((analysis: NonNullable<RequestEvent['analysis']>, nodes: Map<string, ToolNode>) => {
    // Tool calls
    for (const tc of analysis.tool_calls || []) {
      const existing = nodes.get(`tool:${tc.name}`);
      nodes.set(`tool:${tc.name}`, {
        name: tc.name,
        count: (existing?.count || 0) + 1,
        category: 'tool',
        lastUsed: new Date(),
      });
    }
    // MCP servers
    for (const server of analysis.mcp_servers || []) {
      if (!nodes.has(`mcp:${server}`)) {
        nodes.set(`mcp:${server}`, { name: server, count: 0, category: 'mcp' });
      }
    }
    // Skills
    for (const skill of analysis.skills_used || []) {
      const existing = nodes.get(`skill:${skill}`);
      nodes.set(`skill:${skill}`, {
        name: skill,
        count: (existing?.count || 0) + 1,
        category: 'skill',
        lastUsed: new Date(),
      });
    }
  }, []);
  // Load historical events on mount
  useEffect(() => {
    fetch('/api/events')
      .then((r) => r.json())
      .then((data: RequestEvent[]) => {
        if (Array.isArray(data)) {
          setEvents(data);
          const nodes = new Map<string, ToolNode>();
          for (const evt of data) {
            if (evt.analysis) {
              extractNodes(evt.analysis, nodes);
            }
          }
          setToolNodes(nodes);
        }
      })
      .catch(() => {});
  }, [extractNodes]);

  const handleEvent = useCallback((event: WSEvent) => {
    if (event.type === 'request.new') {
      setEvents((prev) => [...prev, event.data]);
      setIsBrainActive(true);
      setTimeout(() => setIsBrainActive(false), 2000);

      // Extract MCP servers and skills from request.new (available immediately)
      if (event.data.analysis) {
        setToolNodes((prev) => {
          const next = new Map(prev);
          extractNodes(event.data.analysis!, next);
          return next;
        });
      }
    }
    if (event.type === 'request.complete') {
      const completed = event.data;
      setEvents((prev) => prev.map((e) => (e.id === completed.id ? completed : e)));
      if (completed.analysis) {
        const newActive = new Set<string>();
        setToolNodes((prev) => {
          const next = new Map(prev);
          extractNodes(completed.analysis!, next);
          // Mark called tools as active
          for (const tc of completed.analysis!.tool_calls || []) {
            newActive.add(`tool:${tc.name}`);
          }
          for (const skill of completed.analysis!.skills_used || []) {
            newActive.add(`skill:${skill}`);
          }          return next;
        });
        setActiveTools(newActive);
        setTimeout(() => setActiveTools(new Set()), 3000);
      }
    }
  }, [extractNodes]);

  const { connected } = useWebSocket(handleEvent);
  const selectedEvent = events.find((e) => e.id === selectedId) || null;
  const totalTokens = events.reduce((sum, e) => {
    if (e.analysis) return sum + e.analysis.input_tokens + e.analysis.output_tokens;
    return sum;
  }, 0);

  // When a request is selected, split into active nodes (graph) and all tools (sidebar)
  const { displayNodes, displayActive, displayRequests, displayTokens, sidebarTools } = useMemo(() => {
    if (!selectedEvent?.analysis) {
      // No selection: show all accumulated, no sidebar
      return {
        displayNodes: Array.from(toolNodes.values()),
        displayActive: activeTools,
        displayRequests: events.length,
        displayTokens: totalTokens,
        sidebarTools: null,
      };
    }

    const a = selectedEvent.analysis;
    const req = selectedEvent.request;
    const activeNodes: ToolNode[] = [];
    const active = new Set<string>();

    // Called tools → graph nodes (merged by name)
    const calledToolNames = new Set<string>();
    const toolCallCounts = new Map<string, number>();
    for (const tc of a.tool_calls || []) {
      calledToolNames.add(tc.name);
      toolCallCounts.set(tc.name, (toolCallCounts.get(tc.name) || 0) + 1);
      active.add(`tool:${tc.name}`);
    }
    for (const [name, count] of toolCallCounts) {
      activeNodes.push({ name, count, category: 'tool' });
    }

    // Skills → graph nodes
    for (const s of a.skills_used || []) {
      active.add(`skill:${s}`);
      activeNodes.push({ name: s, count: 1, category: 'skill' });
    }

    // MCP servers → only show in graph if any of its tools were called
    const calledMcpServers = new Set<string>();
    for (const tc of a.tool_calls || []) {
      if (tc.name.startsWith('mcp__')) {
        const parts = tc.name.split('__');
        if (parts.length >= 2) calledMcpServers.add(parts[1]);
      }
    }
    for (const s of calledMcpServers) {
      activeNodes.push({ name: s, count: 1, category: 'mcp' });
      active.add(`mcp:${s}`);
    }

    // All tools → sidebar list (grouped by category)
    const allTools: { name: string; category: 'tool' | 'mcp' | 'skill'; isActive: boolean }[] = [];
    for (const tool of req.tools || []) {
      allTools.push({
        name: tool.name,
        category: tool.name.startsWith('mcp__') ? 'mcp' : 'tool',
        isActive: calledToolNames.has(tool.name),
      });
    }
    for (const s of a.skills_used || []) {
      allTools.push({ name: s, category: 'skill', isActive: true });
    }

    return {
      displayNodes: activeNodes,
      displayActive: active,
      displayRequests: 1,
      displayTokens: (a.input_tokens || 0) + (a.output_tokens || 0),
      sidebarTools: allTools,
    };
  }, [selectedEvent, toolNodes, activeTools, events.length, totalTokens]);

  // Get tool call details for the selected node
  const selectedToolCalls = useMemo(() => {
    if (!selectedToolName || !selectedEvent?.analysis) return null;
    const calls = (selectedEvent.analysis.tool_calls || []).filter(
      (tc) => tc.name === selectedToolName
    );
    return calls.length > 0 ? calls : null;
  }, [selectedToolName, selectedEvent]);

  return (
    <div className="app">
      <Header connected={connected} requestCount={events.length} />
      <div className="app-body">
        <RequestList events={events} selectedId={selectedId} onSelect={(id) => { setSelectedId(id); setSelectedToolName(null); }} />
        <div className="app-main">
          <NodeGraph
            tools={displayNodes}
            activeTools={displayActive}
            totalRequests={displayRequests}
            totalTokens={displayTokens}
            isBrainActive={!selectedEvent && isBrainActive}
            onNodeClick={(name) => setSelectedToolName(name === selectedToolName ? null : name)}
          />
          {selectedToolCalls && (
            <div className="tool-detail-popup">
              <div className="tool-detail-header">
                <h4>🔧 {selectedToolName} <span>({selectedToolCalls.length} calls)</span></h4>
                <button onClick={() => setSelectedToolName(null)}>✕</button>
              </div>
              {selectedToolCalls.map((tc, i) => (
                <div key={tc.id || i} className="tool-detail-call">
                  <pre>{JSON.stringify(tc.input, null, 2)}</pre>
                </div>
              ))}
            </div>
          )}
        </div>
        {sidebarTools && <ToolSidebar tools={sidebarTools} />}
      </div>
      {selectedEvent && (
        <RequestDetail event={selectedEvent} onClose={() => { setSelectedId(null); setSelectedToolName(null); }} />
      )}
    </div>
  );
}

export default App;
