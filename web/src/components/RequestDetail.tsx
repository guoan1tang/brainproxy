import type { RequestEvent } from '../types';
import './RequestDetail.css';

interface RequestDetailProps {
  event: RequestEvent | null;
  onClose: () => void;
}

export function RequestDetail({ event, onClose }: RequestDetailProps) {
  if (!event) return null;
  return (
    <div className="request-detail">
      <div className="detail-header">
        <h3>Request {event.id.slice(0, 8)}</h3>
        <button className="detail-close" onClick={onClose}>✕</button>
      </div>
      {(() => {
        const calls = event.analysis?.tool_calls || [];
        const skills = event.analysis?.skills_used || [];
        // Merge tools by name with count
        const toolCounts = new Map<string, number>();
        const calledMcpServers = new Set<string>();
        for (const tc of calls) {
          toolCounts.set(tc.name, (toolCounts.get(tc.name) || 0) + 1);
          if (tc.name.startsWith('mcp__')) {
            const parts = tc.name.split('__');
            if (parts.length >= 2) calledMcpServers.add(parts[1]);
          }
        }
        if (toolCounts.size === 0 && calledMcpServers.size === 0 && skills.length === 0) return null;
        return (
          <div className="detail-tools">
            <span className="detail-tools-label">Activated:</span>
            {Array.from(toolCounts.entries()).map(([name, count]) => (
              <span key={name} className="detail-tool-tag">🔧 {name}{count > 1 ? ` ×${count}` : ''}</span>
            ))}
            {Array.from(calledMcpServers).map((s) => (
              <span key={s} className="detail-tool-tag mcp-tag">🔌 {s}</span>
            ))}
            {skills.map((s) => (
              <span key={s} className="detail-tool-tag skill-tag">✨ {s}</span>
            ))}
          </div>
        );
      })()}
      <div className="detail-columns">
        <div className="detail-column">
          <h4>Input</h4>
          {(() => {
            const msgs = event.request.messages || [];
            const loadedSkills: { name: string; description: string; msgIndex: number }[] = [];
            const availableSkills: { name: string; description: string }[] = [];
            const listMarker = 'The following skills are available for use with the Skill tool:';
            // Also scan system prompt blocks for skill lists
            const sysBlocks = event.request.system;
            let sysText = '';
            if (typeof sysBlocks === 'string') sysText = sysBlocks;
            else if (Array.isArray(sysBlocks)) {
              for (const block of sysBlocks) {
                if (block && typeof block === 'object' && block.text) sysText += block.text;
              }
            }

            // Collect all text sources to scan for skill lists
            const textSources: string[] = [sysText];

            for (let i = 0; i < msgs.length; i++) {
              const msg = msgs[i];
              let text = '';
              const content = msg.content;
              if (typeof content === 'string') text = content;
              else if (Array.isArray(content)) {
                for (const block of content) {
                  if (block && typeof block === 'object' && block.text) text += block.text;
                }
              }
              textSources.push(text);

              // Extract loaded skills (Base directory marker)
              if (msg.role === 'user') {
                const marker = 'Base directory for this skill:';
                let idx = text.indexOf(marker);
                while (idx !== -1) {
                  const rest = text.slice(idx + marker.length);
                  const lineEnd = rest.indexOf('\n');
                  const path = (lineEnd === -1 ? rest : rest.slice(0, lineEnd)).trim();
                  const pathParts = path.split('/');
                  const name = pathParts[pathParts.length - 1] || '';
                  const afterPath = lineEnd === -1 ? '' : rest.slice(lineEnd);
                  const descMatch = afterPath.match(/#\s+(.+)/);
                  const description = descMatch ? descMatch[1].trim() : '';
                  if (name && path.includes('.claude/skills/')) {
                    loadedSkills.push({ name, description, msgIndex: i });
                  }
                  idx = text.indexOf(marker, idx + marker.length);
                }
              }

              // Extract available skills list (may appear in multiple messages/system blocks)
              let searchFrom = 0;
              while (true) {
                const listIdx = text.indexOf(listMarker, searchFrom);
                if (listIdx < 0) break;
                const after = text.slice(listIdx + listMarker.length);
                const lines = after.split('\n');
                for (const line of lines) {
                  const trimmed = line.trim();
                  if (!trimmed.startsWith('- ')) {
                    if (availableSkills.length > 0) break;
                    continue;
                  }
                  const entry = trimmed.slice(2);
                  const colonIdx = entry.indexOf(':');
                  if (colonIdx > 0) {
                    const name = entry.slice(0, colonIdx).trim();
                    const desc = entry.slice(colonIdx + 1).trim().slice(0, 120);
                    const existing = availableSkills.find((s) => s.name === name);
                    if (existing) {
                      if (desc && !existing.description) existing.description = desc;
                    } else {
                      availableSkills.push({ name, description: desc });
                    }
                  } else {
                    const name = entry.trim();
                    if (!availableSkills.find((s) => s.name === name)) {
                      availableSkills.push({ name, description: '' });
                    }
                  }
                }
                searchFrom = listIdx + listMarker.length;
              }
            }

            // Also scan system prompt for skill lists
            for (const srcText of [sysText]) {
              let searchFrom2 = 0;
              while (true) {
                const listIdx = srcText.indexOf(listMarker, searchFrom2);
                if (listIdx < 0) break;
                const after = srcText.slice(listIdx + listMarker.length);
                const lines = after.split('\n');
                for (const line of lines) {
                  const trimmed = line.trim();
                  if (!trimmed.startsWith('- ')) {
                    if (availableSkills.length > 0) break;
                    continue;
                  }
                  const entry = trimmed.slice(2);
                  const colonIdx = entry.indexOf(':');
                  if (colonIdx > 0) {
                    const name = entry.slice(0, colonIdx).trim();
                    const desc = entry.slice(colonIdx + 1).trim().slice(0, 120);
                    const existing = availableSkills.find((s) => s.name === name);
                    if (existing) {
                      if (desc && !existing.description) existing.description = desc;
                    } else {
                      availableSkills.push({ name, description: desc });
                    }
                  } else {
                    const name = entry.trim();
                    if (!availableSkills.find((s) => s.name === name)) {
                      availableSkills.push({ name, description: '' });
                    }
                  }
                }
                searchFrom2 = listIdx + listMarker.length;
              }
            }

            // Parse <available_skills> XML blocks from all text sources
            const xmlSkillMap = new Map<string, string>();
            const availableSkillsRegex = /<available_skills>([\s\S]*?)<\/available_skills>/g;
            const skillBlockRegex = /<skill>([\s\S]*?)<\/skill>/g;
            const nameRegex = /<name>([\s\S]*?)<\/name>/;
            const descRegex = /<description>([\s\S]*?)<\/description>/;

            const allTextSources: string[] = [sysText];
            for (let i = 0; i < msgs.length; i++) {
              const msg = msgs[i];
              let text = '';
              const content = msg.content;
              if (typeof content === 'string') text = content;
              else if (Array.isArray(content)) {
                for (const block of content) {
                  if (block && typeof block === 'object' && block.text) text += block.text;
                }
              }
              allTextSources.push(text);
            }

            for (const srcText of allTextSources) {
              let xmlMatch;
              availableSkillsRegex.lastIndex = 0;
              while ((xmlMatch = availableSkillsRegex.exec(srcText)) !== null) {
                const xmlContent = xmlMatch[1];
                let skillMatch;
                skillBlockRegex.lastIndex = 0;
                while ((skillMatch = skillBlockRegex.exec(xmlContent)) !== null) {
                  const skillBlock = skillMatch[1];
                  const nameMatch = nameRegex.exec(skillBlock);
                  const descMatch = descRegex.exec(skillBlock);
                  if (nameMatch) {
                    const name = nameMatch[1].trim();
                    const desc = descMatch ? descMatch[1].trim() : '';
                    const existing = xmlSkillMap.get(name);
                    if (!existing || (desc && !existing)) {
                      xmlSkillMap.set(name, desc);
                    }
                  }
                }
              }
            }

            // Merge XML-parsed skills into availableSkills
            for (const [name, description] of xmlSkillMap) {
              const existing = availableSkills.find((s) => s.name === name);
              if (existing) {
                if (description && !existing.description) existing.description = description;
              } else {
                availableSkills.push({ name, description });
              }
            }

            // Sort: skills with descriptions first
            availableSkills.sort((a, b) => {
              if (a.description && !b.description) return -1;
              if (!a.description && b.description) return 1;
              return a.name.localeCompare(b.name);
            });

            if (availableSkills.length === 0 && loadedSkills.length === 0) return null;
            return (
              <div className="detail-section">
                {availableSkills.length > 0 && (
                  <details>
                    <summary>📋 Available Skills ({availableSkills.length})</summary>
                    <div className="skill-list">
                      {availableSkills.map((s, i) => (
                        <div key={i} className={`skill-item ${loadedSkills.some(l => l.name === s.name) ? 'skill-loaded' : ''}`}>
                          <span className="skill-name">{s.name}</span>
                          {s.description && <span className="skill-desc">{s.description}</span>}
                          {loadedSkills.some(l => l.name === s.name) && <span className="skill-badge">loaded</span>}
                        </div>
                      ))}
                    </div>
                  </details>
                )}
                {loadedSkills.length > 0 && (
                  <details open>
                    <summary>✨ Skills Loaded ({loadedSkills.length})</summary>
                    <div className="skill-list">
                      {loadedSkills.map((s, i) => (
                        <div key={i} className="skill-item skill-loaded">
                          <span className="skill-name">✨ {s.name}</span>
                          {s.description && <span className="skill-desc">{s.description}</span>}
                          <span className="skill-msg">msg[{s.msgIndex}]</span>
                        </div>
                      ))}
                    </div>
                  </details>
                )}
              </div>
            );
          })()}
          {(() => {
            // Extract system-reminder blocks from messages
            const reminders: { content: string; msgIndex: number; role: string }[] = [];
            const msgs = event.request.messages || [];
            for (let i = 0; i < msgs.length; i++) {
              const msg = msgs[i];
              let text = '';
              const content = msg.content;
              if (typeof content === 'string') text = content;
              else if (Array.isArray(content)) {
                for (const block of content) {
                  if (block && typeof block === 'object' && block.text) text += block.text;
                }
              }
              const regex = /<system-reminder>([\s\S]*?)<\/system-reminder>/g;
              let match;
              while ((match = regex.exec(text)) !== null) {
                const reminderText = match[1].trim();
                if (reminderText.length > 0) {
                  reminders.push({ content: reminderText.slice(0, 500), msgIndex: i, role: msg.role });
                }
              }
            }
            if (reminders.length === 0) return null;
            return (
              <div className="detail-section">
                <details>
                  <summary>⚙️ System Reminders ({reminders.length})</summary>
                  <div className="reminder-list">
                    {reminders.map((r, i) => (
                      <div key={i} className="reminder-item">
                        <div className="reminder-header">msg[{r.msgIndex}] {r.role}</div>
                        <pre className="reminder-content">{r.content}{r.content.length >= 500 ? '\n...(truncated)' : ''}</pre>
                      </div>
                    ))}
                  </div>
                </details>
              </div>
            );
          })()}
          <div className="detail-section">
            <details>
              <summary>Messages ({event.request.messages?.length || 0})</summary>
              <pre className="detail-json">{JSON.stringify(event.request.messages, null, 2)}</pre>
            </details>
          </div>
          {event.request.system && (
            <div className="detail-section">
              <details>
                <summary>System Prompt</summary>
                <pre className="detail-json">
                  {typeof event.request.system === 'string'
                    ? event.request.system
                    : JSON.stringify(event.request.system, null, 2)}
                </pre>
              </details>
            </div>
          )}
          {event.request.tools && event.request.tools.length > 0 && (
            <div className="detail-section">
              <details>
                <summary>Tools ({event.request.tools.length})</summary>
                <pre className="detail-json">{JSON.stringify(event.request.tools, null, 2)}</pre>
              </details>
            </div>
          )}
        </div>
        <div className="detail-column">
          <h4>Output</h4>
          {event.response ? (
            <>
              <div className="detail-section">
                <details open>
                  <summary>Response Content</summary>
                  <pre className="detail-json">{JSON.stringify(event.response.content, null, 2)}</pre>
                </details>
              </div>
              {event.analysis && (
                <div className="detail-meta">
                  <div className="meta-item"><span className="meta-label">Model</span><span>{event.analysis.model}</span></div>
                  <div className="meta-item"><span className="meta-label">Input tokens</span><span>{event.analysis.input_tokens}</span></div>
                  <div className="meta-item"><span className="meta-label">Output tokens</span><span>{event.analysis.output_tokens}</span></div>
                  {(event.analysis.cache_read_input_tokens || 0) > 0 && (
                    <div className="meta-item"><span className="meta-label">Cache read</span><span className="cache-hit">{event.analysis.cache_read_input_tokens}</span></div>
                  )}
                  {(event.analysis.cache_creation_input_tokens || 0) > 0 && (
                    <div className="meta-item"><span className="meta-label">Cache create</span><span className="cache-create">{event.analysis.cache_creation_input_tokens}</span></div>
                  )}
                  <div className="meta-item"><span className="meta-label">Stop reason</span><span>{event.analysis.stop_reason}</span></div>
                  <div className="meta-item"><span className="meta-label">Duration</span><span>{event.analysis.duration_ms}ms</span></div>
                </div>
              )}
            </>
          ) : (
            <div className="detail-pending">Waiting for response...</div>
          )}
        </div>
      </div>
    </div>
  );
}
